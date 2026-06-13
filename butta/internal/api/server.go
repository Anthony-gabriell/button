// Package api expõe o engine via HTTP. Stdlib pura (net/http) —
// desde o Go 1.22 o ServeMux nativo aceita método + path pattern,
// então não precisamos de chi/gin pra um MVP.
package api

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/thony/butta/internal/engine"
	"github.com/thony/butta/web"
)

// dailySeed gera a seed do "ranqueado do dia": todo mundo que jogar
// hoje monta a partir do MESMO pool. Amanhã, pool novo. Simples e justo.
func dailySeed() int64 {
	now := time.Now().UTC()
	return int64(now.Year()*10000 + int(now.Month())*100 + now.Day())
}

// NewServer monta as rotas e devolve um http.Handler pronto.
// Receber/devolver interfaces (http.Handler) facilita teste e composição.
func NewServer() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/pool", handlePool)
	mux.HandleFunc("POST /api/play", handlePlay)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Frontend embutido no binário: "GET /" serve o web/index.html.
	// http.FileServerFS (Go 1.22) serve direto de um embed.FS.
	mux.Handle("GET /", http.FileServerFS(web.FS))

	return withCORS(mux) // middleware: padrão decorator em Go
}

// --- Handlers ---------------------------------------------------------------

// GET /api/pool → pool de botões do dia + orçamento.
func handlePool(w http.ResponseWriter, r *http.Request) {
	rng := rand.New(rand.NewSource(dailySeed()))
	pool := engine.GeneratePool(rng, 300)

	writeJSON(w, http.StatusOK, map[string]any{
		"budget":    engine.Budget,
		"team_size": engine.TeamSize,
		"pool":      pool,
	})
}

// playRequest é o corpo esperado no POST /api/play.
type playRequest struct {
	TeamName  string `json:"team_name"`
	ButtonIDs []int  `json:"button_ids"` // IDs escolhidos do pool do dia
}

// POST /api/play → valida a escalação e roda o torneio completo.
func handlePlay(w http.ResponseWriter, r *http.Request) {
	var req playRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}

	// Reconstrói o pool do dia e indexa por ID.
	// NUNCA confie em dados de botão vindos do cliente (preço/rating
	// adulterado = cheat). O cliente só manda IDs; o servidor resolve.
	rng := rand.New(rand.NewSource(dailySeed()))
	pool := engine.GeneratePool(rng, 300)
	byID := make(map[int]engine.Button, len(pool))
	for _, b := range pool {
		byID[b.ID] = b
	}

	team := engine.Team{Name: req.TeamName}
	if team.Name == "" {
		team.Name = "Meu Time"
	}
	for _, id := range req.ButtonIDs {
		b, ok := byID[id]
		if !ok {
			writeError(w, http.StatusBadRequest, "botão inexistente no pool do dia")
			return
		}
		team.Buttons = append(team.Buttons, b)
	}

	if err := team.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// A partida em si usa seed ALEATÓRIA (cada run é única) —
	// só o pool de montagem é fixo no dia.
	matchRng := rand.New(rand.NewSource(time.Now().UnixNano()))
	result := engine.RunTournament(matchRng, team, pool, 68)

	writeJSON(w, http.StatusOK, result)
}

// --- Helpers ----------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("erro ao serializar resposta: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// withCORS libera o frontend local. Em produção, restrinja a origem.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
