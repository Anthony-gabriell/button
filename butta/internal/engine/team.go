package engine

import (
	"errors"
	"fmt"
)

// Budget é o orçamento padrão de montagem (a regra de ouro do jogo).
const Budget = 1100

// TeamSize é o número de botões titulares.
const TeamSize = 11

// Erros sentinela: padrão idiomático em Go para erros conhecidos.
// Quem chama pode testar com errors.Is(err, engine.ErrOverBudget).
var (
	ErrOverBudget   = errors.New("custo do time excede o orçamento")
	ErrWrongSize    = errors.New("o time precisa ter exatamente 11 botões")
	ErrNoGoalkeeper = errors.New("o time precisa ter exatamente 1 goleiro")
)

// Team é uma escalação montada (pelo usuário ou por um bot).
type Team struct {
	Name    string   `json:"name"`
	Country string   `json:"country"`
	Buttons []Button `json:"buttons"`
}

// Cost soma o preço de todos os botões.
// Repare no receiver `(t Team)`: método de VALOR (não altera o struct).
// Use ponteiro (*Team) só quando precisar modificar — regra prática de Go.
func (t Team) Cost() int {
	total := 0
	for _, b := range t.Buttons { // range: o jeito Go de iterar
		total += b.Price
	}
	return total
}

// Overall é a força do time: média dos ratings.
// Float para não perder precisão na simulação.
func (t Team) Overall() float64 {
	if len(t.Buttons) == 0 {
		return 0
	}
	sum := 0
	for _, b := range t.Buttons {
		sum += b.Rating
	}
	return float64(sum) / float64(len(t.Buttons))
}

// Attack e Defense separam a força por setor — isso deixa a simulação
// mais rica: time com ataque forte e defesa fraca gera jogos de 4x3.
func (t Team) Attack() float64 { return t.sectorAvg(Atacante, Meia) }
func (t Team) Defense() float64 {
	return t.sectorAvg(Defensor, Goleiro)
}

func (t Team) sectorAvg(positions ...Position) float64 {
	sum, count := 0, 0
	for _, b := range t.Buttons {
		for _, p := range positions {
			if b.Position == p {
				sum += b.Rating
				count++
			}
		}
	}
	if count == 0 {
		return 50 // setor vazio = setor fraquíssimo (punição natural)
	}
	return float64(sum) / float64(count)
}

// Validate confere as regras de montagem. Devolve erro descritivo
// (o frontend mostra a mensagem direto pro usuário).
func (t Team) Validate() error {
	if len(t.Buttons) != TeamSize {
		return fmt.Errorf("%w: tem %d", ErrWrongSize, len(t.Buttons))
	}

	goleiros := 0
	for _, b := range t.Buttons {
		if b.Position == Goleiro {
			goleiros++
		}
	}
	if goleiros != 1 {
		return fmt.Errorf("%w: tem %d", ErrNoGoalkeeper, goleiros)
	}

	if cost := t.Cost(); cost > Budget {
		// %w embrulha o erro sentinela mantendo a mensagem extra —
		// errors.Is continua funcionando. Esse é O padrão de erro em Go.
		return fmt.Errorf("%w: custo %d, orçamento %d", ErrOverBudget, cost, Budget)
	}
	return nil
}
