// O main é só "fiação": lê config, monta o servidor e sobe.
// Lógica de verdade mora em internal/ — main magro é main saudável.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/thony/butta/internal/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      api.NewServer(),
		ReadTimeout:  5 * time.Second,  // timeouts explícitos:
		WriteTimeout: 10 * time.Second, // servidor sem timeout é bug esperando acontecer
	}

	log.Printf("⚽ Butta no ar em http://localhost:%s", port)
	log.Printf("   GET  /api/pool   → pool de botões do dia")
	log.Printf("   POST /api/play   → joga o torneio")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
