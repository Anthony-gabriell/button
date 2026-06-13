// Package engine contém toda a lógica de simulação do Butta.
//
// Princípio de design: este pacote é PURO — não conhece HTTP, banco de
// dados ou frontend. Recebe dados, devolve dados. Isso torna ele 100%
// testável com `go test` e reutilizável (CLI, API, worker, o que for).
package engine

import (
	"fmt"
	"math/rand"
)

// Position representa a posição de um botão em campo.
// Em Go, criamos um tipo próprio em vez de usar string solta —
// isso dá segurança de tipo: o compilador impede valores inválidos.
type Position string

const (
	Goleiro  Position = "GOL"
	Defensor Position = "DEF"
	Meia     Position = "MEI"
	Atacante Position = "ATA"
)

// Rarity define o tier visual/colecionável do botão (estilo gacha).
type Rarity string

const (
	Comum    Rarity = "COMUM"
	Raro     Rarity = "RARO"
	Epico    Rarity = "EPICO"
	Lendario Rarity = "LENDARIO"
)

// Button é a unidade básica do jogo: um botão colecionável.
//
// As tags `json:"..."` controlam como o struct vira JSON na API.
// Repare: nomes exportados (maiúscula) = visíveis fora do pacote.
type Button struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`    // nome fictício (sem direito de imagem!)
	Country  string   `json:"country"` // seleção
	Position Position `json:"position"`
	Rating   int      `json:"rating"` // 50–99, define força e preço
	Price    int      `json:"price"`  // custo em points na montagem
	Rarity   Rarity   `json:"rarity"`
}

// rarityFor deriva a raridade a partir do rating.
// Função não exportada (minúscula) = detalhe interno do pacote.
func rarityFor(rating int) Rarity {
	switch {
	case rating >= 90:
		return Lendario
	case rating >= 82:
		return Epico
	case rating >= 72:
		return Raro
	default:
		return Comum
	}
}

// priceFor converte rating em preço.
// Curva intencional: lendários custam desproporcionalmente mais,
// forçando o trade-off "1 craque + elenco fraco" vs "time equilibrado".
// É essa decisão que torna a montagem divertida.
func priceFor(rating int) int {
	base := rating // piso linear
	switch {
	case rating >= 90:
		return base + (rating-90)*16 + 60 // 90→150 ... 99→294
	case rating >= 82:
		return base + (rating-82)*5 + 15 // 82→97 ... 89→132
	case rating >= 72:
		return base + 5
	default:
		return base - 10
	}
}

// countries usadas para gerar o pool. Nomes de países não são IP de ninguém.
var countries = []string{
	"Brasil", "Argentina", "França", "Alemanha", "Espanha",
	"Inglaterra", "Portugal", "Itália", "Holanda", "Uruguai",
	"Croácia", "Marrocos", "Japão", "México", "EUA", "Bélgica",
}

// GeneratePool cria o pool de botões disponíveis para montagem.
//
// rng injetado de fora (em vez de rand global) = DETERMINISMO:
// mesma seed → mesmo pool. Essencial para o modo ranqueado
// (todos os jogadores do dia montam a partir do mesmo pool)
// e para testes reproduzíveis.
func GeneratePool(rng *rand.Rand, size int) []Button {
	// distribuição de posições num pool: ~10% GOL, 35% DEF, 30% MEI, 25% ATA
	positions := []Position{Goleiro, Defensor, Defensor, Defensor, Meia, Meia, Meia, Atacante, Atacante, Atacante}

	pool := make([]Button, 0, size) // make com capacidade evita realocações
	for i := 0; i < size; i++ {
		// Distribuição de rating: muitos medianos, poucos craques.
		// rand.NormFloat64 ≈ curva normal (média 0, desvio 1).
		rating := int(68 + rng.NormFloat64()*9)
		if rating < 50 {
			rating = 50
		}
		if rating > 99 {
			rating = 99
		}

		country := countries[rng.Intn(len(countries))]
		pos := positions[rng.Intn(len(positions))]

		pool = append(pool, Button{
			ID:       i + 1,
			Name:     fmt.Sprintf("%s do %s", nicknameFor(pos, rng), country),
			Country:  country,
			Position: pos,
			Rating:   rating,
			Price:    priceFor(rating),
			Rarity:   rarityFor(rating),
		})
	}
	return pool
}

// nicknameFor gera apelidos fictícios de botão — clima de várzea/lenda,
// zero risco jurídico. Depois você pluga nomes melhores ou um gerador.
func nicknameFor(p Position, rng *rand.Rand) string {
	byPos := map[Position][]string{
		Goleiro:  {"Paredão", "Muralha", "Gato", "Polvo"},
		Defensor: {"Xerife", "Cadeado", "Rocha", "General"},
		Meia:     {"Maestro", "Cérebro", "Mágico", "Regente"},
		Atacante: {"Furacão", "Artilheiro", "Raio", "Matador"},
	}
	names := byPos[p]
	return names[rng.Intn(len(names))]
}
