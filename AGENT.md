# AGENT.md — adjudex-mcp

> **adjudex** is an MCP server for tracking and analyzing stock prices,
> with an embedded SvelteKit web frontend.
> The name is an acronym: **A**gent ju**D**y e**X**change — only project insiders know why.

---

## 1. Vision & Scope

adjudex provides:
- **Portfolio management** — define and persist watchlists and holdings
- **Quote retrieval** — fetch current and historical stock prices from external data sources
- **Trend analysis** — basic indicators (moving averages, relative performance)
- **Alerting** - define triggers based on trends and get notified
- **MCP interface** — all core functions exposed as MCP tools (stdio + SSE transport)
- **Web UI** — SvelteKit frontend embedded via `embed.FS`, served by the Go binary

The server runs as a single self-contained binary. No Docker required for basic usage.

---

## 2. Technology Stack

| Layer | Choice | Notes |
|---|---|---|
| Language | Go 1.23+ | Idiomatic Go, no CGO |
| MCP library | `github.com/mark3labs/mcp-go` | stdio + SSE transport |
| Frontend | SvelteKit (static build) | Embedded via `embed.FS` |
| Database | SQLite via `go-database` | Local persistence, no external DB |
| Build | `Makefile` | Targets: `build`, `test`, `lint`, `dev`, `web` |
| Linter | `golangci-lint` | Config in `.golangci.yml` |

---

## 3. Own Libraries (prefer over external alternatives)

Always check this list before reaching for an external package.
All modules follow the path pattern `github.com/tdrn-org/<name>`.

| Library | Module path | Purpose |
|---|---|---|
| go-log | `github.com/tdrn-org/go-log` | All logging — **mandatory**, no `log` or `slog` directly |
| go-httpserver | `github.com/tdrn-org/go-httpserver` | HTTP server setup for the embedded frontend |
| go-conf | `github.com/tdrn-org/go-conf` | Configuration loading (file + env) |
| go-tlsconf | `github.com/tdrn-org/go-tlsconf` | TLS setup for HTTPS mode |
| go-database | `github.com/tdrn-org/go-database` | SQLite access, transactions, schema migrations |
| go-pool | `github.com/tdrn-org/go-pool` | Connection pooling if needed |

External libraries are only permitted when no own alternative exists.
**Document the reason in a comment or ADR when adding an external dependency.**

---

## 4. Repository Structure

```
adjudex-mcp/
├── AGENT.md                  ← This file
├── ARCHITECTURE.md            ← ADRs and architecture decisions (agent maintains this)
├── Makefile
├── go.mod                     ← module: github.com/tdrn-org/adjudex-mcp
├── go.sum
├── .golangci.yml
│
├── cmd/
│   └── adjudex-mcp/
│       └── main.go            ← Entry point, wires everything together
│
├── internal/
│   ├── domain/                ← Pure domain types & interfaces, NO external imports
│   │   ├── portfolio.go       ← Portfolio, Position, Holding types
│   │   ├── quote.go           ← Quote, PriceHistory, Indicator types
│   │   └── store.go           ← Repository interfaces (PortfolioStore, QuoteStore)
│   │
│   ├── stock/                 ← Quote fetching implementations
│   │   ├── provider.go        ← Provider interface
│   │   └── yahoo/             ← Yahoo Finance implementation (or other source)
│   │       └── yahoo.go
│   │
│   ├── store/                 ← SQLite implementations of domain repository interfaces
│   │   ├── portfolio_store.go
│   │   └── quote_store.go
│   │
│   ├── mcp/                   ← MCP server setup and tool definitions
│   │   ├── server.go          ← Server init, transport selection
│   │   └── tools/
│   │       ├── portfolio.go   ← MCP tools: portfolio management
│   │       └── quotes.go      ← MCP tools: quote retrieval & analysis
│   │
│   └── web/                   ← Embedded frontend handler
│       ├── handler.go         ← Serves embed.FS via go-httpserver
│       └── static/            ← SvelteKit build output (git-ignored, generated)
│           └── .gitkeep
│
└── web/                       ← SvelteKit source
    ├── package.json
    ├── svelte.config.js       ← adapter-static, outDir: ../internal/web/static
    ├── vite.config.ts
    └── src/
        ├── app.html
        ├── lib/
        └── routes/
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

### No global state
No `init()` functions that register global state.
No package-level vars for runtime state — pass dependencies explicitly.

### Configuration
All runtime configuration via `go-conf`.
Config file: `adjudex-mcp.yaml` (location via `--config` flag or `ADJUDEX_MCP_CONFIG` env var).
API keys and secrets only via environment variables, never in config files.

---

## 6. MCP Interface Design Guidelines

- Each MCP tool has a **single, clear responsibility**.
- Tool names: `snake_case`, verb-noun pattern (e.g. `get_quote`, `add_position`).
- All tools return structured JSON-compatible results.
- Error responses use the MCP error format, not plain strings.
- Tools are grouped by domain area (portfolio tools vs. quote tools).

---

## 7. Frontend Integration

- SvelteKit is built with `adapter-static` into `internal/web/static/`.
- The Go build embeds that directory via `//go:embed static` in `internal/web/handler.go`.
- The web UI communicates with the backend via a REST API served by `go-httpserver`.
- API routes are prefixed with `/api/v1/`.
- The frontend is always served at `/` — the Go binary is the only process to run.

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

| Phase | Deliverable | Review? |
|---|---|---|
| 0 | Repo skeleton, `go.mod`, Makefile, SvelteKit init | No |
| 1 | Domain types & interfaces (`internal/domain/`) | **Yes** |
| 2 | MCP tool signatures (no implementation) | **Yes** |
| 3 | SQLite store implementations | No |
| 4 | First quote provider (fetch & persist) | No |
| 5 | MCP server wiring + tool implementations | No |
| 6 | REST API for frontend | **Yes** |
| 7 | SvelteKit UI — portfolio view | No |
| 8 | SvelteKit UI — quote & chart view | No |
| 9 | Configuration, TLS, packaging | No |

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
