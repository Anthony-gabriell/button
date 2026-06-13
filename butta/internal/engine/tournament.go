package engine

import (
	"fmt"
	"math/rand"
)

// ============================================================
// TORNEIO MATA-MATA
//
// O time do jogador enfrenta bots em sequência:
// Oitavas → Quartas → Semi → FINAL (4 partidas).
// A dificuldade dos bots sobe a cada fase — curva de tensão.
// ============================================================

// Stage nomeia cada fase (vai direto pro frontend).
var stages = []string{"OITAVAS", "QUARTAS", "SEMIFINAL", "FINAL"}

// TournamentResult é o resultado completo da campanha.
type TournamentResult struct {
	Champion bool          `json:"champion"`  // o jogador foi campeão?
	StageOut string        `json:"stage_out"` // onde caiu ("" se campeão)
	Matches  []MatchResult `json:"matches"`
	Score    int           `json:"score"` // pontuação pro ranking/share
}

// botNames: times fictícios com clima de várzea lendária.
var botNames = []string{
	"Estrela do Norte", "Furacão da Vila", "Real Periferia",
	"Galáticos FC", "Trovão Azul", "Império do Botão",
	"Dragões do Sul", "Lendas da Mesa",
}

// RunTournament joga a campanha completa do time do usuário.
//
// botBaseOVR: overall médio dos bots na primeira fase (ex.: 70).
// A cada fase os bots ficam +4 de rating — a final é sempre dura.
func RunTournament(rng *rand.Rand, player Team, pool []Button, botBaseOVR int) TournamentResult {
	result := TournamentResult{}

	for i, stage := range stages {
		bot := generateBot(rng, pool, botBaseOVR+i*4)

		match := Simulate(rng, player, bot, true) // true = mata-mata
		result.Matches = append(result.Matches, match)

		// Pontuação: gols marcados valem, saldo vale, avançar vale muito
		result.Score += match.HomeGoals*10 + (match.HomeGoals-match.AwayGoals)*5

		if match.Winner != player.Name {
			result.StageOut = stage
			return result // caiu: campanha encerrada
		}
		result.Score += 50 * (i + 1) // bônus crescente por fase vencida
	}

	result.Champion = true
	result.Score += 200 // bônus de título
	return result
}

// generateBot monta um adversário com overall alvo aproximado.
// Estratégia simples: filtra botões num intervalo de rating em volta
// do alvo e completa o 4-3-3 clássico (1 GOL, 4 DEF, 3 MEI, 3 ATA).
func generateBot(rng *rand.Rand, pool []Button, targetOVR int) Team {
	need := map[Position]int{Goleiro: 1, Defensor: 4, Meia: 3, Atacante: 3}
	team := Team{Name: botNames[rng.Intn(len(botNames))], Country: "Bot"}

	// Embaralha uma CÓPIA do pool (não mutar o original — boa prática:
	// funções não devem ter efeitos colaterais surpresa em quem chamou).
	shuffled := make([]Button, len(pool))
	copy(shuffled, pool)
	rng.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	// 1º passe: respeita a janela de rating do alvo (+/-6)
	for _, b := range shuffled {
		if need[b.Position] > 0 && abs(b.Rating-targetOVR) <= 6 {
			team.Buttons = append(team.Buttons, b)
			need[b.Position]--
		}
	}
	// 2º passe: completa com o que tiver (pool pequeno não pode travar)
	for _, b := range shuffled {
		if need[b.Position] > 0 {
			team.Buttons = append(team.Buttons, b)
			need[b.Position]--
		}
	}

	team.Name = fmt.Sprintf("%s (OVR %.0f)", team.Name, team.Overall())
	return team
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
