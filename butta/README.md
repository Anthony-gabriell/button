# ⚽ Butta — fantasy de futebol de botão

Monte seu time com **1100 points**, encare o mata-mata contra bots e compartilhe sua campanha. Backend 100% Go, stdlib pura, zero dependências externas.

## Rodando

```bash
go test ./... -v        # roda os testes do engine
go run ./cmd/server     # sobe TUDO em http://localhost:8080 (jogo + API)
```

## Testando a API na mão

```bash
# Pool de botões do dia (mesma seed pra todo mundo = ranqueado justo)
curl localhost:8080/api/pool

# Jogar o torneio (IDs de botões escolhidos do pool, 11 no total, 1 GOL)
curl -X POST localhost:8080/api/play \
  -d '{"team_name":"Meu Time","button_ids":[1,5,8,12,20,33,41,57,62,78,90]}'
```

## Estrutura

```
butta/
├── cmd/server/main.go        # entry point (main magro, só fiação)
├── internal/
│   ├── engine/               # ❤️ lógica pura, sem HTTP/DB — 100% testável
│   │   ├── button.go         # Button, raridades, geração do pool (seed)
│   │   ├── team.go           # Team, orçamento, validação, força por setor
│   │   ├── match.go          # simulação: xG → Poisson → timeline de eventos
│   │   ├── tournament.go     # mata-mata vs bots com dificuldade crescente
│   │   └── engine_test.go    # table-driven tests
│   └── api/server.go         # HTTP stdlib (Go 1.22 routing) + CORS
└── web/                      # frontend embutido no binário (go:embed)
    ├── web.go                # //go:embed index.html
    └── index.html            # jogo completo: montagem, replay, share
```

## Decisões de design (o porquê)

- **Engine puro + API fina**: a simulação não sabe que HTTP existe. Amanhã ela roda num worker, num CLI ou num bot de WhatsApp sem mudar uma linha.
- **Determinismo por seed**: pool do dia = mesma seed pra todos (ranqueado justo). Partida = seed aleatória (cada run é única).
- **Servidor é a fonte de verdade**: o cliente só manda IDs de botões. Preço, rating e resultado são resolvidos no backend → cheat impossível.
- **Timeline de eventos como JSON**: o frontend "toca o replay" da lista de eventos. Animação barata, anti-cheat de graça.
- **luckFactor = 0.30**: time forte vence ~87% contra time muito mais fraco. Montar bem importa, mas a zebra vive (validado em `TestStrongerTeamWinsMore`).

## Roadmap (anti-beta-eterno: uma fase por vez)

1. ~~Frontend~~ ✅ feito: montagem, replay animado e card de share (vanilla + SVG, embutido no Go)
2. Login (Google) + ranking diário → gancho: "seu nome no ranking"
3. Ghost PvP: oponente = time salvo de usuário real
4. Coins, packs, coleção gacha (modo livre, separado do ranqueado)
5. Campo 2D animado (a feature "uau", só com tração)

## Roteiro de estudo Go neste código

| Conceito | Onde ver |
|---|---|
| Tipos próprios + constantes tipadas | `button.go` (Position, Rarity) |
| Erros sentinela + `errors.Is` + `%w` | `team.go` |
| Métodos de valor vs ponteiro | `team.go` (comentário em `Cost`) |
| Injeção de dependência (`*rand.Rand`) | `button.go`, `match.go` |
| Table-driven tests + subtests + `t.Helper` | `engine_test.go` |
| ServeMux 1.22 (método + path) + middleware | `api/server.go` |
| Slices: `make` com cap, `copy`, `Shuffle` | `tournament.go` |
