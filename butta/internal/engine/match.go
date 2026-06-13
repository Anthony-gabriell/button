package engine

import (
	"math"
	"math/rand"
)

// ============================================================
// SIMULAÇÃO DE PARTIDA
//
// Modelo em 3 camadas (do macro pro micro):
//   1. Força relativa  → quantos gols cada time "espera" fazer (xG)
//   2. Poisson         → sorteia o placar real a partir do xG
//   3. Timeline        → distribui os eventos pelos 90 minutos
//
// O frontend recebe a lista de eventos e "toca o replay".
// O servidor é a única fonte de verdade (anti-cheat de graça).
// ============================================================

// EventType enumera o que pode acontecer numa partida.
type EventType string

const (
	EventGol    EventType = "GOL"
	EventCartao EventType = "CARTAO"
	EventChance EventType = "CHANCE" // quase-gol: tensão pro replay
	EventApito  EventType = "APITO"  // fim de jogo
)

// Event é um acontecimento da partida, pronto pra virar JSON.
type Event struct {
	Minute int       `json:"minute"`
	Type   EventType `json:"type"`
	Team   string    `json:"team"`   // nome do time
	Actor  string    `json:"actor"`  // nome do botão envolvido
	Detail string    `json:"detail"` // texto livre pro frontend exibir
}

// MatchResult é o pacote completo que a API devolve.
type MatchResult struct {
	HomeTeam  string  `json:"home_team"`
	AwayTeam  string  `json:"away_team"`
	HomeGoals int     `json:"home_goals"`
	AwayGoals int     `json:"away_goals"`
	HomeOVR   float64 `json:"home_ovr"`
	AwayOVR   float64 `json:"away_ovr"`
	Events    []Event `json:"events"`
	Winner    string  `json:"winner"` // em mata-mata nunca há empate (pênaltis)
	Penalties bool    `json:"penalties"`
}

// luckFactor controla o peso da zebra. 0 = só estatística (chato),
// 1 = loteria pura (montar time não importa). 0.30 = ponto doce:
// time forte ganha mais, mas o "será que dessa vez?" continua vivo.
const luckFactor = 0.30

// expectedGoals calcula o xG de um time contra outro.
//
// A intuição da fórmula:
//   - ataque vs defesa adversária define a vantagem
//   - tanh comprime a vantagem num intervalo suave (-1..1),
//     evitando placares absurdos quando a diferença é grande
//   - baseGoals ≈ média de gols de um time num jogo normal
func expectedGoals(attack, oppDefense float64) float64 {
	const baseGoals = 1.35
	advantage := (attack - oppDefense) / 12.0 // 12 pts de rating = vantagem clara
	return baseGoals * math.Exp(math.Tanh(advantage))
}

// poisson sorteia um inteiro com distribuição de Poisson (algoritmo de Knuth).
// É o modelo estatístico clássico para gols no futebol: eventos raros
// e independentes ao longo do tempo.
func poisson(rng *rand.Rand, lambda float64) int {
	l := math.Exp(-lambda)
	k, p := 0, 1.0
	for {
		p *= rng.Float64()
		if p <= l {
			return k
		}
		k++
	}
}

// Simulate roda uma partida completa entre dois times.
// knockout=true → empate vai pra pênaltis (sempre sai um vencedor).
func Simulate(rng *rand.Rand, home, away Team, knockout bool) MatchResult {
	// 1. xG de cada lado, com a sorte misturada
	homeXG := mixLuck(rng, expectedGoals(home.Attack(), away.Defense()))
	awayXG := mixLuck(rng, expectedGoals(away.Attack(), home.Defense()))

	// 2. Poisson transforma expectativa em placar
	homeGoals := poisson(rng, homeXG)
	awayGoals := poisson(rng, awayXG)

	result := MatchResult{
		HomeTeam: home.Name, AwayTeam: away.Name,
		HomeGoals: homeGoals, AwayGoals: awayGoals,
		HomeOVR: round1(home.Overall()), AwayOVR: round1(away.Overall()),
	}

	// 3. Timeline: espalha os eventos pelos 90 minutos
	result.Events = buildTimeline(rng, home, away, homeGoals, awayGoals)

	// 4. Vencedor (com pênaltis se mata-mata)
	switch {
	case homeGoals > awayGoals:
		result.Winner = home.Name
	case awayGoals > homeGoals:
		result.Winner = away.Name
	case knockout:
		result.Penalties = true
		// Pênalti é quase moeda: leve peso pro time de maior overall.
		if rng.Float64() < 0.5+(home.Overall()-away.Overall())/200 {
			result.Winner = home.Name
		} else {
			result.Winner = away.Name
		}
	}
	return result
}

// mixLuck mistura o xG calculado com ruído aleatório.
// Lerp clássico: valor*(1-f) + ruído*f.
func mixLuck(rng *rand.Rand, xg float64) float64 {
	noise := 0.4 + rng.Float64()*2.2 // ruído entre 0.4 e 2.6 gols esperados
	return xg*(1-luckFactor) + noise*luckFactor
}

// buildTimeline cria a narrativa da partida minuto a minuto.
func buildTimeline(rng *rand.Rand, home, away Team, hg, ag int) []Event {
	var events []Event

	// Gols: minuto aleatório, autor sorteado com peso por posição
	for i := 0; i < hg; i++ {
		events = append(events, goalEvent(rng, home))
	}
	for i := 0; i < ag; i++ {
		events = append(events, goalEvent(rng, away))
	}

	// Chances perdidas e cartões dão tensão ao replay (2 a 5 extras)
	for i, n := 0, 2+rng.Intn(4); i < n; i++ {
		t := home
		if rng.Intn(2) == 0 {
			t = away
		}
		typ, detail := EventChance, "Quase! Bola raspa a trave"
		if rng.Intn(3) == 0 {
			typ, detail = EventCartao, "Cartão amarelo"
		}
		events = append(events, Event{
			Minute: 1 + rng.Intn(90), Type: typ, Team: t.Name,
			Actor: t.Buttons[rng.Intn(len(t.Buttons))].Name, Detail: detail,
		})
	}

	sortEventsByMinute(events)
	events = append(events, Event{Minute: 90, Type: EventApito, Detail: "Fim de jogo!"})
	return events
}

// goalEvent sorteia o autor do gol: atacantes têm 3x mais chance que
// meias, que têm 3x mais que defensores. Goleiro não faz gol (ainda 😄).
func goalEvent(rng *rand.Rand, t Team) Event {
	weights := map[Position]int{Atacante: 9, Meia: 3, Defensor: 1, Goleiro: 0}
	var pool []Button
	for _, b := range t.Buttons {
		for i := 0; i < weights[b.Position]; i++ {
			pool = append(pool, b)
		}
	}
	scorer := pool[rng.Intn(len(pool))]
	return Event{
		Minute: 1 + rng.Intn(90), Type: EventGol, Team: t.Name,
		Actor: scorer.Name, Detail: "GOOOL de " + scorer.Name + "!",
	}
}

// sortEventsByMinute: insertion sort simples — listas minúsculas,
// não vale importar sort por uma coisa dessas... brincadeira: vale,
// mas escrever um sort à mão uma vez na vida é estudo de Go. ;)
func sortEventsByMinute(events []Event) {
	for i := 1; i < len(events); i++ {
		for j := i; j > 0 && events[j].Minute < events[j-1].Minute; j-- {
			events[j], events[j-1] = events[j-1], events[j] // swap idiomático
		}
	}
}

func round1(f float64) float64 { return math.Round(f*10) / 10 }
