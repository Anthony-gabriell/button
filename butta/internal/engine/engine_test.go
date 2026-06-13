package engine

import (
	"errors"
	"math/rand"
	"testing"
)

// ============================================================
// TESTES — o jeito Go: table-driven tests.
// Rode com:  go test ./... -v
// ============================================================

// helper: monta um time válido com rating fixo para os testes.
// t.Helper() faz o erro apontar pra linha do TESTE, não do helper.
func makeTeam(t *testing.T, name string, rating int) Team {
	t.Helper()
	team := Team{Name: name, Country: "Teste"}
	positions := []Position{
		Goleiro,
		Defensor, Defensor, Defensor, Defensor,
		Meia, Meia, Meia,
		Atacante, Atacante, Atacante,
	}
	for i, p := range positions {
		team.Buttons = append(team.Buttons, Button{
			ID: i, Name: "Botão", Position: p, Rating: rating, Price: 90,
		})
	}
	return team
}

// --- Validação de montagem -------------------------------------------------

func TestTeamValidate(t *testing.T) {
	// A "tabela": cada caso é um cenário. Adicionar cenário novo = 1 linha.
	tests := []struct {
		name    string
		mutate  func(*Team) // altera o time válido pra criar o cenário
		wantErr error       // erro sentinela esperado (nil = válido)
	}{
		{"time valido passa", func(tm *Team) {}, nil},
		{"estoura orcamento", func(tm *Team) {
			for i := range tm.Buttons {
				tm.Buttons[i].Price = 200 // 11 × 200 = 2200 > 1100
			}
		}, ErrOverBudget},
		{"menos de 11 botoes", func(tm *Team) {
			tm.Buttons = tm.Buttons[:9]
		}, ErrWrongSize},
		{"sem goleiro", func(tm *Team) {
			tm.Buttons[0].Position = Atacante
		}, ErrNoGoalkeeper},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { // subtestes nomeados
			team := makeTeam(t, "Valida FC", 70)
			tt.mutate(&team)

			err := team.Validate()
			if tt.wantErr == nil && err != nil {
				t.Fatalf("esperava válido, veio erro: %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("esperava %v, veio %v", tt.wantErr, err)
			}
		})
	}
}

// --- Determinismo: o contrato do modo ranqueado ----------------------------

func TestSimulateDeterministic(t *testing.T) {
	home := makeTeam(t, "Casa", 80)
	away := makeTeam(t, "Fora", 75)

	// Mesma seed → resultado IDÊNTICO. Se este teste quebrar,
	// o modo ranqueado quebrou junto.
	r1 := Simulate(rand.New(rand.NewSource(42)), home, away, true)
	r2 := Simulate(rand.New(rand.NewSource(42)), home, away, true)

	if r1.HomeGoals != r2.HomeGoals || r1.AwayGoals != r2.AwayGoals || r1.Winner != r2.Winner {
		t.Fatalf("mesma seed gerou resultados diferentes: %+v vs %+v", r1, r2)
	}
}

// --- Sanidade estatística: time forte vence MAIS, mas não SEMPRE -----------

func TestStrongerTeamWinsMore(t *testing.T) {
	strong := makeTeam(t, "Craques", 88)
	weak := makeTeam(t, "Perna de Pau", 65)

	rng := rand.New(rand.NewSource(7))
	wins := 0
	const n = 2000
	for i := 0; i < n; i++ {
		if Simulate(rng, strong, weak, true).Winner == strong.Name {
			wins++
		}
	}

	rate := float64(wins) / n
	t.Logf("time forte venceu %.1f%% das %d partidas", rate*100, n)

	// Forte deve vencer bem mais da metade...
	if rate < 0.62 {
		t.Errorf("time forte venceu só %.1f%% — montar time não está valendo a pena", rate*100)
	}
	// ...mas zebra PRECISA existir (senão o replay não tem graça).
	if rate > 0.97 {
		t.Errorf("time forte venceu %.1f%% — cadê a zebra?", rate*100)
	}
}

// --- Torneio completo -------------------------------------------------------

func TestRunTournament(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	pool := GeneratePool(rng, 300)
	player := makeTeam(t, "Butta United", 85)

	result := RunTournament(rng, player, pool, 68)

	if result.Champion && result.StageOut != "" {
		t.Error("campeão não pode ter fase de eliminação")
	}
	if !result.Champion && result.StageOut == "" {
		t.Error("eliminado precisa registrar em qual fase caiu")
	}
	if len(result.Matches) == 0 || len(result.Matches) > 4 {
		t.Errorf("número de partidas inválido: %d", len(result.Matches))
	}
	if result.Score <= 0 && result.Matches[0].HomeGoals > 0 {
		t.Error("fez gol mas pontuação ficou zerada")
	}
	t.Logf("campeão=%v, caiu em=%q, score=%d, partidas=%d",
		result.Champion, result.StageOut, result.Score, len(result.Matches))
}

// --- Pool -------------------------------------------------------------------

func TestGeneratePool(t *testing.T) {
	pool := GeneratePool(rand.New(rand.NewSource(1)), 500)

	if len(pool) != 500 {
		t.Fatalf("pool com tamanho errado: %d", len(pool))
	}
	for _, b := range pool {
		if b.Rating < 50 || b.Rating > 99 {
			t.Errorf("rating fora da faixa: %+v", b)
		}
		if b.Price <= 0 {
			t.Errorf("preço inválido: %+v", b)
		}
	}

	// Determinismo do pool: contrato do ranqueado diário.
	again := GeneratePool(rand.New(rand.NewSource(1)), 500)
	if pool[0] != again[0] || pool[499] != again[499] {
		t.Error("mesma seed gerou pools diferentes")
	}
}
