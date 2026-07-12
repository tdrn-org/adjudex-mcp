# AGENT.md — adjudex-mcp
meta: [[20_Meta/Go-Development]], [[20_Meta/SvelteKit-Development]]

> **adjudex** is an MCP server for tracking and analyzing stock prices,
> with an embedded SvelteKit web frontend.
> The name is an acronym: **A**gent ju**D**y e**X**change — only project insiders know why.

---

## 1. Vision & Scope

adjudex provides:
- **Portfolio management** — define and persist watchlists and holdings
- **Quote retrieval** — fetch current and historical stock prices via multi-provider pipeline (Twelve Data → Alpha Vantage → Demo fallback)
- **Technical indicators** — SMA, EMA, RSI, MACD via store-first resolution
- **Alerting** — define price/indicator triggers with state machine (armed → triggered → acknowledged)
- **Trading strategies** — define, save, and backtest mean-reversion strategies
- **MCP interface** — all core functions exposed as 24 MCP tools (stdio + SSE transport)
- **REST API** — 12 endpoints with Swagger documentation for SvelteKit frontend
- **Web UI** — SvelteKit frontend embedded via `embed.FS`, served by the Go binary

The server runs as a single self-contained binary. No Docker required for basic usage.

---

## 2. Technology Stack

| Layer | Choice | Notes |
|---|---|---|
| Language | Go 1.23+ | Idiomatic Go, no CGO |
| MCP library | `github.com/mark3labs/mcp-go` v1.6.1 | stdio + SSE transport, `json.RawMessage` args |
| Frontend | SvelteKit 2.x + Svelte 5 | Static build, SPA mode (`ssr = false`), TailwindCSS v4 |
| Database | SQLite via `go-database` | Local persistence, auto-migrations |
| Quotes | Twelve Data → Alpha Vantage → Demo | Multi-provider chain with affinity cache |
| HTTP | `go-httpserver` with CertificateProvider | TLS + SPA fallback handler |
| Metrics | Prometheus | `/metrics` endpoint, MCP tool call counters |
| Swagger | swaggo/swag | Code-first OpenAPI from handler annotations |
| Config | `go-conf` | File + env vars, triple-layer (defaults → env → CLI flags) |
| Build | `Makefile` | Targets: `build`, `test`, `fmt`, `web`, `clean`, `dist` |
| Linter | `golangci-lint` | Config in `.golangci.yml` |

---

## 3. Own Libraries (prefer over external alternatives)

Always check this list before reaching for an external package.
All modules follow the path pattern `github.com/tdrn-org/<name>`.

| Library | Module path | Purpose |
|---|---|---|
| go-log | `github.com/tdrn-org/go-log` | All logging — **mandatory**, no `log` or `slog` directly |
| go-httpserver | `github.com/tdrn-org/go-httpserver` | HTTP server setup for embedded frontend + REST API |
| go-conf | `github.com/tdrn-org/go-conf` | Configuration loading (file + env) |
| go-tlsconf | `github.com/tdrn-org/go-tlsconf` | TLS setup for HTTPS mode |
| go-database | `github.com/tdrn-org/go-database` | SQLite access, transactions, schema migrations |

External libraries are only permitted when no own alternative exists.
**Document the reason in a comment or ADR when adding an external dependency.**

---

## 4. Repository Structure

```
adjudex-mcp/
├── AGENT.md                  ← This file
├── ARCHITECTURE.md            ← ADRs and architecture decisions (Judy maintains)
├── Makefile
├── go.mod                     ← module: github.com/tdrn-org/adjudex-mcp
├── go.sum
├── .golangci.yml
│
├── cmd/
│   └── adjudex-mcp/
│       └── main.go            ← Entry point, wires everything
│
├── config/                    ← Configuration (go-conf based)
│   ├── config.go
│   ├── defaults.go
│   ├── logging.go
│   ├── metrics.go
│   ├── quote_service.go
│   ├── server.go
│   └── store.go
│
├── internal/
│   ├── domain/                ← Pure domain types & interfaces, NO external imports
│   │   ├── doc.go
│   │   ├── alert.go           ← Alert, AlertCondition, AlertState, IndicatorSpec
│   │   ├── portfolio.go       ← Portfolio, Position, Holding
│   │   ├── quote.go           ← Quote, PriceHistory, IndicatorType, IndicatorValue
│   │   ├── strategy.go        ← Strategy, Trade, BacktestResult
│   │   └── store.go           ← Repository interfaces (PortfolioStore, QuoteStore, etc.)
│   │
│   ├── data/                  ← go-database models + Store implementation
│   │   ├── data.go            ← Store struct (implements ALL domain interfaces)
│   │   ├── data_test.go
│   │   └── model/             ← Per-entity model files
│   │       ├── model.go       ← Schema V1, DB2Time/Time2DB helpers
│   │       ├── alert.go
│   │       ├── portfolio.go
│   │       ├── position.go
│   │       ├── quote.go
│   │       ├── strategy.go
│   │       └── trade.go
│   │
│   ├── stock/                 ← Quote fetching
│   │   ├── stock.go           ← QuoteService (store-first, affinity cache, maxAge)
│   │   └── tracker/           ← Provider implementations
│   │       ├── tracker.go     ← Provider interface
│   │       ├── demo/          ← Deterministic random-walk demo provider
│   │       ├── twelvedata/    ← Twelve Data REST API
│   │       └── alphavantage/  ← Alpha Vantage REST API
│   │
│   ├── adapters/middleware/
│   │   ├── mcp/               ← MCP tool registration (24 tools)
│   │   │   ├── mcp.go         ← Runtime interface, server init
│   │   │   ├── alerts.go
│   │   │   ├── portfolio.go
│   │   │   ├── quotes.go
│   │   │   ├── strategies.go
│   │   │   ├── symbols.go
│   │   │   └── trades.go
│   │   ├── rest/              ← REST API (12 endpoints + Swagger)
│   │   │   ├── rest.go        ← Mount + wiring
│   │   │   ├── api.go         ← Handler implementations
│   │   │   ├── docs.go        ← Swagger docs
│   │   │   ├── swagger.json   ← Generated OpenAPI spec
│   │   │   └── swagger.yaml
│   │   └── metrics/           ← Prometheus metrics
│   │       └── metrics.go
│   │
│   ├── web/                   ← Embedded frontend handler
│   │   ├── handler.go         ← Serves embed.FS via go-httpserver with SPA fallback
│   │   └── static/            ← SvelteKit build output (git-ignored, generated)
│   │       └── .gitkeep
│   │
│   └── buildinfo/             ← Version info (ldflags injection)
│
├── server.go                  ← Server lifecycle, job registration
├── server_jobs.go             ← Background jobs (quote fetching, alert evaluation)
├── config.go                  ← Top-level config wiring
│
└── web/                       ← SvelteKit source (to be created in Phase 7)
    ├── package.json
    ├── svelte.config.js       ← adapter-static → ../internal/web/static
    ├── vite.config.ts
    └── src/
        ├── app.html
        ├── app.css             ← TailwindCSS v4 (@import "tailwindcss")
        ├── lib/
        │   └── api.ts          ← Typed fetch() wrappers
        └── routes/
            ├── +layout.svelte  ← Navigation + dark theme shell
            ├── +layout.ts      ← export const ssr = false
            ├── +page.svelte    ← Portfolio list dashboard
            └── portfolio/
                └── [id]/
                    ├── +page.svelte  ← Portfolio detail + holdings
                    └── +page.ts      ← load() via API
```

---

## 5. Architecture Principles

### Domain layer isolation
`internal/domain/` contains only pure Go types and interfaces.
It **must not** import anything outside the standard library.
All external dependencies (DB, HTTP, MCP) depend on domain — never the reverse.

### Explicit errors
- Return errors explicitly; never use `panic()` for recoverable conditions.
- Wrap errors with context: `fmt.Errorf("portfolio %s: load: %w", id, err)`
- Use `go-log` for all logging — no direct `log`, `slog`, or `fmt.Print*` for log output.

### Interfaces at boundaries
Every external dependency (DB store, quote provider, MCP server) is accessed via an interface.
This enables unit testing with mocks without spinning up real services.

### Store-First Quote Resolution
Background job ticker keeps quotes warm in SQLite. `QuoteService.ResolveQuote()` checks:
1. Store (maxAge 15min) → return cached
2. Provider affinity cache → prefer the provider that last succeeded
3. Fallback chain → Twelve Data → Alpha Vantage → Demo

MCP tools and REST handlers call `QuoteService.ResolveQuote()` — one call, all three tiers.

### No global state
No `init()` functions that register global state.
No package-level vars for runtime state — pass dependencies explicitly.

### Configuration (triple-layer)
`defaults.go` constants → `ADJUDEX_*` env vars → CLI flags.
API keys only via environment variables, never in config files.

---

## 6. MCP Interface Design Guidelines

- Each MCP tool has a **single, clear responsibility**.
- Tool names: `snake_case`, verb-noun pattern (e.g. `quote_get`, `portfolio_create`).
- All tools return structured JSON-compatible results.
- Error responses use the MCP error format, not plain strings.
- Tools are grouped by domain area (portfolio tools vs. quote tools).
- Tool count: **24 tools** across 6 domains (portfolio, quotes, alerts, strategies, trades, symbols).

---

## 7. Frontend Integration

- SvelteKit built with `adapter-static` into `internal/web/static/`.
- Go build embeds via `//go:embed all:static` in `internal/web/handler.go`.
- **SPA mode** (`ssr = false` in `+layout.ts`) — required for `embed.FS` serving.
- **SPA fallback** — Go handler returns `index.html` for unknown paths to support client-side routing.
- Web UI communicates via REST API at `/api/v1/`.
- API response format: JSON objects with `snake_case` keys matching Go struct tags.
- Dark theme default via TailwindCSS v4 custom properties.

---

## 8. Testing Expectations

- Unit tests for all domain logic and store implementations.
- Integration tests (build tag `//go:build integration`) for external provider calls.
- `go test ./...` must pass with no external services running.
- Test files sit next to the code they test (`*_test.go`).

---

## 9. Review Checkpoints — STOP and request approval

The agent **must pause** and present a summary for review before proceeding when:

1. **New MCP tool signatures** — name, parameters, return type
2. **Domain model changes** — new types or changes to existing ones in `internal/domain/`
3. **Database schema decisions** — new tables, columns, indexes
4. **External dependency additions** — any new entry in `go.mod` not from `tdrn-org`
5. **API endpoint additions** — new REST routes exposed to the frontend
6. **Architecture deviations** — anything that contradicts this document

Format for review requests:
```
[REVIEW REQUIRED]
Topic: <short title>
Proposal: <what is proposed>
Reason: <why this approach>
Alternatives considered: <what else was evaluated>
```

Do not implement until explicit approval is given.

---

## 10. Iterative Build Order

Follow this sequence. Do not skip phases without approval.

| Phase | Deliverable | Status |
|---|---|---|
| 0 | Repo skeleton, `go.mod`, Makefile, SvelteKit init | ✅ Done |
| 1 | Domain types & interfaces (`internal/domain/`) | ✅ Done |
| 2 | MCP tool signatures (24 tools across 6 domains) | ✅ Done |
| 3 | SQLite store implementations (go-database, schema V1) | ✅ Done |
| 4 | Quote providers (Demo, Twelve Data, Alpha Vantage) | ✅ Done |
| 5 | MCP server wiring + tool implementations | ✅ Done |
| 6 | REST API (12 endpoints + Swagger) | ✅ Done |
| 7 | SvelteKit UI — portfolio view | 🔜 Next |
| 8 | SvelteKit UI — quote & chart view | 🔜 After |
| 9 | Configuration, TLS, packaging | ⏳ Later |

---

## 11. ARCHITECTURE.md Maintenance

After each approved Review Checkpoint, the agent appends an ADR (Architecture Decision Record)
to `ARCHITECTURE.md` in this format:

```markdown
## ADR-NNN: <title>
**Date**: YYYY-MM-DD
**Status**: Accepted
**Decision**: ...
**Reason**: ...
**Consequences**: ...
```
