# ARCHITECTURE.md — adjudex-mcp

Architecture Decision Records for adjudex — Agent juDy eXchange.

Maintained by Judy (Hermes Agent). Each ADR documents a design decision
made during development, following the review checkpoint process defined
in AGENT.md §9.

---

## ADR-001: Project Genesis

**Date**: 2026-05-25
**Status**: Accepted

**Decision**: Initialize adjudex as a Go-based MCP server for stock tracking
and portfolio analysis, following Clean Architecture with domain isolation,
embedded SvelteKit frontend, SQLite persistence, and tdrn-org libraries.

**Reason**: The project was conceived by Holger during a vacation in Paleo
Faliro as a companion to the mean-reversion trading strategy. Judy will be
the primary consumer of the MCP tools, with a minimal web UI for human review.

**Consequences**:
- Repository: github.com/tdrn-org/adjudex-mcp
- Binary name: adjudex
- Phase 0 (skeleton) delivered during Holger's return flight on 2026-05-25
- Phase 1 (domain types) pending review after landing
- Initial quote provider: mock (Yahoo Finance placeholder)
- Frontend: backend-first, SvelteKit is secondary

---

## ADR-002: Consorsbank API as Target Quote Provider

**Date**: 2026-05-25
**Status**: Proposed (pending investigation)

**Decision**: Investigate the official Consorsbank Trading API
(https://developer.consorsbank.de/documentation) as the primary quote
and portfolio provider for adjudex.

**Reason**: Holger is an active Consorsbank customer and the bank provides
a full Trading API with endpoints for Orders, Securities, Securities-Account,
and Profile — plus a dedicated Sandbox for development. This would give
adjudex direct access to real portfolio data, order placement, and trade
history — eliminating the need for third-party providers (Yahoo, Alpha Vantage).

**API Capabilities** (from documentation):
- Portfolio analysis and multi-portfolio apps
- Securities account access (holdings, transactions)
- Order placement with ExAnteCost (pre-trade cost estimation)
- Sandbox environment for testing without real money
- Authentication via OAuth 2.0 (customer-level access)

**Consequences**:
- adjudex moves from a "tracking" tool to a "trading companion" with live data
- Consorsbank provider replaces the Yahoo/mock placeholder in `internal/stock/`
- Need to design `internal/stock/consorsbank/` as a dedicated adapter package
- OAuth flow must be implemented for customer authentication
- Sandbox-first development: test against simulated portfolio before real access
- Phase 2026-05-25: Holger to verify his account's API access level

---

## ADR-003: Domain Model Design (Phase 1)

**Date**: 2026-05-25
**Status**: Accepted (reviewed by Holger pre-flight)

**Decision**: Five domain files with pure standard-library types defining
the complete adjudex domain model:

| File | Types | Responsibility |
|------|-------|---------------|
| `portfolio.go` | Portfolio, Position, Holding | Portfolio management + derived holding values |
| `quote.go` | Quote, PriceHistory, IndicatorType, IndicatorValue | Market data snapshots + technical indicators |
| `alert.go` | Alert, AlertCondition, AlertState, IndicatorSpec | Notification triggers with state machine |
| `strategy.go` | Strategy, StrategyParams, Trade, TradeDirection, TradeStatus, BacktestResult | Trading strategies, execution, and backtesting |
| `store.go` | PortfolioStore, QuoteStore, AlertStore, TradeStore, StrategyStore | Persistence interfaces (no implementations) |

**Reason**: Phase 1 of the build order (AGENT.md §10). Domain isolation
enforced: zero external imports beyond `context` and `time`. The `Alert` type
was added as a standalone concept (not merged into Indicator) per Holger's
design direction — Indicator is data, Alert is stateful action.

**Key design decisions**:
- `Holding` is a derived type, not persisted (computed from Position + Quote)
- `Alert` has a full state machine: Armed → Triggered → Acknowledged (or Rearmed)
- `StrategyParams` is a value object within Strategy, not a separate entity
- All Store interfaces accept `context.Context` for cancellation/deadline support
- `BacktestResult` includes Sharpe ratio and max drawdown for strategy evaluation

**Domain Layer Isolation verified**: `go list -f '{{.Imports}}' ./internal/domain/`
returns only `[context time]` — no external packages imported.

**Consequences**:
- Phase 2 (MCP tool signatures) can reference these types directly
- Phase 3 (SQLite store) implements these interfaces
- Future PIM integration stays separate — adjudex is trading-only (per F2 decision)

---

## ADR-004: MCP Tool Signatures (Phase 2)

**Date**: 2026-05-26
**Status**: Accepted (reviewed by Holger)

**Decision**: Define 21 MCP tool signatures across five domain groups, all as
Phase 5 stubs (`panic("not implemented: Phase 5")`). Implemented in
`internal/mcp/tools/` with one file per domain group.

**Tool inventory**:

| # | Group | File | Tools |
|---|-------|------|-------|
| H1–H7 | Portfolio | `portfolio.go` | create, get, list, delete, add_position, remove_position, get_holdings |
| H8–H10 | Quotes | `quotes.go` | get, history, indicator |
| H11–H14 | Alerts | `alerts.go` | create, list, acknowledge, delete |
| H15–H19 | Strategy | `strategies.go` | create, get, list, delete, backtest |
| H20–H21 | Trades | `trades.go` | list, get |

**Reason**: Phase 2 of the build order (AGENT.md §10). The signatures form a
complete contract that Phase 5 implementations must fulfill. Designing them now
exposes the full API surface early, allowing frontend development (Phases 7–8)
to proceed in parallel with backend implementation.

**Key design decisions**:

- **Derived operations**: `portfolio_get_holdings` (H7) joins Position + Quote;
  `quote_indicator` (H10) computes from Quote history; `strategy_backtest` (H19)
  simulates execution across all three stores. These are not thin CRUD wrappers —
  they represent adjudex's core value proposition.
- **State machine**: `alert_acknowledge` (H13) calls `domain.Alert.Acknowledge()`
  rather than manipulating state directly — the domain model owns the transition.
- **Store signatures preserved**: Every tool maps to exactly the Store interfaces
  defined in Phase 1 (`internal/domain/store.go`). No store changes needed.
- **All stubs panic intentionally**: Phase 2 is a contract, not an implementation.
  The stubs compile and type-check against the domain model but defer execution
  until Phase 5.

**Consequences**:
- Phase 3 (SQLite store) can be implemented independently — the tools define
  exactly what the store must provide.
- Phase 5 (MCP server wiring) has a clear target: 21 tool registration calls.
- Frontend API design (Phase 6) now knows the complete data surface.
- `strategy_backtest` (H19) is the riskiest tool — it requires all three stores
  plus the Strategy domain logic. Its signature was deliberately included now to
  force early design thinking about the backtest architecture.

---

## ADR-005: Quote Provider Interface & Mock Implementation (Phase 4)

**Date**: 2026-05-27
**Status**: Accepted

**Decision**: Define a `stock.Provider` interface and a Yahoo mock implementation
that generates plausible synthetic historical data for development. The mock
will be replaced by the Consorsbank API (ADR-002) once account access is verified.

**Provider Interface** (`internal/stock/provider.go`):
- `FetchQuote(ctx, symbol) → (*domain.Quote, error)` — latest quote
- `FetchHistory(ctx, symbol, from, to) → ([]domain.Quote, error)` — date range

**Yahoo Mock** (`internal/stock/yahoo/yahoo.go`):
- Deterministic random walk: seeded RNG per day ensures reproducible output
- ~1.5% daily volatility with micro-drift, realistic OHLCV fields
- Business-day-aware: weekends skipped, holidays ignored
- `FetchQuote` returns a single synthetic quote anchored at "now"
- `FetchHistory(Mon–Fri)` returns exactly 5 quotes in chronological order
- Zero external API dependencies — pure development scaffolding

**Tests** (`internal/stock/yahoo/yahoo_test.go`):
8 tests covering: basic FetchQuote, 5-day history, weekend-only (returns empty),
invalid date range (from > to), deterministic output (same seed = same prices),
chronological sort order, all OHLCV fields populated, and interface compliance.

**Key design decisions**:
- Provider interface at `internal/stock/provider.go` (not in domain) — providers
  are IO-bound, domain is pure. Interface lives at the package boundary.
- Compile-time check: `var _ stock.Provider = (*yahoo.Provider)(nil)`
- No external dependencies: `math/rand/v2` replaces third-party random libraries
- `FetchHistory` returns sorted ascending — stores can append without re-sorting

**Consequences**:
- Phase 5 (MCP server wiring) can inject the provider via interface
- Testing is fully deterministic (no flaky integration tests)
- Consorsbank adapter will implement the same interface (drop-in replacement)
- adjudex can now serve live mock data end-to-end without any external services

---

## ADR-006: MCP Server Wiring & Tool Implementations (Phase 5)

**Date**: 2026-05-27
**Status**: Accepted

**Decision**: Wire all 21 MCP tool signatures from Phase 2 into the `mcp-go` server
with full implementations that connect domain logic, SQLite stores, and the Yahoo
mock provider. Expose the server via both stdio and SSE transport.

**Architecture**:

- **`internal/mcp/server.go`**: Creates and configures the `mcp-go` server.
  `NewServer(dbFile, transport)` opens the SQLite store, creates the Yahoo provider,
  and calls all `Register*` functions.
- **`internal/mcp/tools/register.go`**: Central wiring hub. Contains five
  `Register*` functions (`RegisterPortfolioTools`, etc.) that call `srv.AddTool`
  for each tool group. This is where domain stores and providers are injected
  into the tool handler closures.
- **`cmd/adjudex/main.go`**: Parses `--transport` (stdio/sse) and `--db` flags,
  then calls `mcp.NewServer()`.

**StoreSet pattern**: A lightweight struct bundling all five domain store interfaces
is passed to tool registration functions. This avoids individual parameter explosions
for tools that need multiple stores (e.g., `portfolio_get_holdings` needs both
PortfolioStore and QuoteStore).

**21 tools implemented end-to-end**:

| # | Tool | Input | Output |
|---|------|-------|--------|
| 1 | `portfolio_create` | name, description | Portfolio |
| 2 | `portfolio_get` | id | Portfolio + Positions |
| 3 | `portfolio_list` | — | []Portfolio |
| 4 | `portfolio_delete` | id | confirmation |
| 5 | `portfolio_add_position` | portfolio_id, symbol, shares, avg_price | Position |
| 6 | `portfolio_remove_position` | portfolio_id, position_id | confirmation |
| 7 | `portfolio_get_holdings` | portfolio_id | []Holding (with current price) |
| 8 | `quote_get` | symbol | Quote |
| 9 | `quote_history` | symbol, from, to | []Quote |
| 10 | `quote_indicator` | symbol, indicator_type, period | IndicatorValue |
| 11 | `alert_create` | portfolio_id, symbol, condition, threshold | Alert |
| 12 | `alert_list` | portfolio_id | []Alert |
| 13 | `alert_acknowledge` | id | Alert (state: acknowledged) |
| 14 | `alert_delete` | id | confirmation |
| 15 | `strategy_create` | name, type, params | Strategy |
| 16 | `strategy_get` | id | Strategy |
| 17 | `strategy_list` | — | []Strategy |
| 18 | `strategy_delete` | id | confirmation |
| 19 | `strategy_backtest` | strategy_id, symbol, from, to | BacktestResult |
| 20 | `trade_list` | strategy_id | []Trade |
| 21 | `trade_get` | id | Trade |

**Key design decisions**:

- **No `panic` stubs remain**: All 21 tools have proper implementations. The domain
  Store interfaces from Phase 1 are fully realized via the Phase 3 SQLite
  implementations.
- **Quote provider injection**: The Yahoo mock is created once in `server.go` and
  passed into `RegisterQuoteTools` and `RegisterStrategyTools`. When the Consorsbank
  adapter (ADR-002) is ready, only `server.go` changes — all tools are provider-agnostic.
- **Indicator calculations**: `quote_indicator` (tool #10) implements RSI, SMA, EMA,
  and MACD using standard formulas on historical quote data. Computed entirely from
  stored quotes — no external API calls.
- **Backtest engine**: `strategy_backtest` (tool #19) simulates mean-reversion trading
  across a date range using the strategy's parameters. It generates synthetic Trades
  and computes a BacktestResult with Sharpe ratio and max drawdown.
- **`marshalResult` helper**: All tool handlers return `*mcp.CallToolResult` via a
  shared JSON marshaling utility, ensuring consistent error formatting.
- **Transport abstraction**: `--transport stdio` for local/Judy usage, `--transport sse`
  for remote/HTTP access. Both handled by `mcp-go` without code duplication.

**Consequences**:

- adjudex is now a fully functional MCP server — connect any MCP client and use all
  21 tools immediately.
- Phase 6 (REST API) can be designed knowing the complete backend surface.
- Phase 7–8 (SvelteKit UI) has a running backend to develop against.
- Testing deferred: store and provider have unit tests; tool-level integration tests
  are a future task.
- The mock provider generates realistic synthetic data — adjudex is demo-ready with
- zero external dependencies.

---

## ADR-007: REST API Design (Phase 6)

**Date**: 2026-05-28
**Status**: Accepted

**Decision**: Expose all 21 adjudex MCP tools as REST endpoints under `/api/v1/` using
`go-httpserver` (which wraps `net/http`). Each endpoint mirrors exactly one MCP tool
from Phase 5, reusing the same `tools.*` functions directly — zero logic duplication.

**Approval**: Holger approved the route design (F1), choice of `go-httpserver` (F2),
and synchronous backtest (F3). [REVIEW REQUIRED] checkpoint per AGENT.md §9 satisfied.

**Route Table** (21 endpoints):

| Method | Path | MCP Tool |
|--------|------|----------|
| POST | `/api/v1/portfolios` | `portfolio_create` |
| GET | `/api/v1/portfolios` | `portfolio_list` |
| GET | `/api/v1/portfolios/{id}` | `portfolio_get` |
| DELETE | `/api/v1/portfolios/{id}` | `portfolio_delete` |
| POST | `/api/v1/portfolios/{id}/positions` | `portfolio_add_position` |
| DELETE | `/api/v1/portfolios/{pid}/positions/{posid}` | `portfolio_remove_position` |
| GET | `/api/v1/portfolios/{id}/holdings` | `portfolio_get_holdings` |
| GET | `/api/v1/quotes/{symbol}` | `quote_get` |
| GET | `/api/v1/quotes/{symbol}/history` | `quote_history` |
| GET | `/api/v1/quotes/{symbol}/indicator` | `quote_indicator` |
| POST | `/api/v1/alerts` | `alert_create` |
| GET | `/api/v1/alerts` | `alert_list` |
| PUT | `/api/v1/alerts/{id}/acknowledge` | `alert_acknowledge` |
| DELETE | `/api/v1/alerts/{id}` | `alert_delete` |
| POST | `/api/v1/strategies` | `strategy_create` |
| GET | `/api/v1/strategies` | `strategy_list` |
| GET | `/api/v1/strategies/{id}` | `strategy_get` |
| DELETE | `/api/v1/strategies/{id}` | `strategy_delete` |
| POST | `/api/v1/strategies/{id}/backtest` | `strategy_backtest` |
| GET | `/api/v1/trades` | `trade_list` |
| GET | `/api/v1/trades/{id}` | `trade_get` |

**Implementation**:

- `internal/api/router.go` — route registration on `go-httpserver.Instance` using
  Go 1.22+ path patterns (`{id}`, `{pid}`, `{posid}`, `{symbol}`). Injects
  `mcp.StoreSet` into a shared `handler` struct.
- `internal/api/portfolio.go`, `quotes.go`, `alerts.go`, `strategies.go`, `trades.go` —
  one file per domain, each calling the corresponding `tools.*` functions directly.
- JSON helpers: `writeJSON`, `writeError`, `decodeBody`, `handleNotFound`.
- `cmd/adjudex/main.go` — new `--transport http` mode creates a `go-httpserver`
  instance, calls `api.Router()`, and serves.

**Schema fix**: `store.Config()` was not passing adjudex schema scripts to
`go-database`, causing `UpdateSchema` to be a no-op. Fixed by adding
`sqlite.WithSchemaScripts(Schema()...)` to the config constructor. All existing
functionality (stdio/SSE) also benefits from this fix.

**Smoke test results** (all passed):
- `POST /api/v1/portfolios` → 201 Created with full Portfolio JSON
- `GET /api/v1/portfolios` → 200 OK with array
- `GET /api/v1/portfolios/{id}` → 200 OK with positions
- `DELETE /api/v1/portfolios/{id}` → 200 OK
- `POST /api/v1/strategies` → 201 Created
- `DELETE /api/v1/strategies/{id}` → 200 OK, confirmed empty list after
- `GET /api/v1/trades`, `alerts` → 200 OK with empty arrays

**Consequences**:

- The SvelteKit frontend (Phases 7–8) can now communicate with adjudex via REST
  without needing MCP protocol knowledge.
- `--transport http` is the default mode for serving the embedded web UI.
- `--transport stdio` remains for Judy's MCP integration.
- `--transport sse` remains for remote MCP clients.
- No new external dependencies beyond `go-httpserver` (tdrn-org library).
- All endpoint implementations are thin wrappers — business logic stays in `tools.*`.
  - A bug fix in a tool function automatically fixes both the MCP and REST interfaces.

  ---

  ## ADR-008: Phase 8 — Quote & Chart View (SvelteKit)

  **Date**: 2026-05-29
  **Status**: Accepted

  **Decision**: Build the quote exploration and price chart UI as a second SvelteKit
  route (`/quotes`) using pure SVG components with zero external charting
  dependencies.

  **Key decisions**:
  - **Pure SVG charts** (`PriceChart.svelte`): Rejected chart.js (heavy, canvas-based)
    and LayerCake (complex, still requires manual drawing). SVG `<polyline>` + `<text>`
    elements with computed scales cover all needed chart types (price line, SMA overlay)
    with zero additional bundle weight.
  - **Client-side SMA(20)**: Computed from history closes in the browser — avoids an
    additional API round-trip for indicator data.
  - **Route-based navigation**: Second route `/quotes` with tab navigation in the
    layout, per Holger's preference (rejected hash-based approach).
  - **Quote card**: OHLCV summary with color-coded change percentage, mirroring
    professional trading UIs.

  **Consequences**:
  - Phase 8 delivered 4 files: `types.ts` (3 new types), `api.ts` (3 new endpoints),
    `PriceChart.svelte` (SVG chart component), `routes/quotes/+page.svelte` (full page).
  - `+layout.svelte` updated with Portfolio | Quotes tab navigation.
  - Zero new npm dependencies — pure SVG + Svelte 5 runes.
  - API bug fix: `quoteHistory` handler now defaults to 1-month range when no
    `from`/`to` parameters are provided (was panicking on empty string parse).

  ## ADR-009: Phase 9 — Configuration, TLS, and Packaging

  **Date**: 2026-05-29
  **Status**: Accepted

  **Decision**: Implement a triple-layer configuration system (defaults → env vars →
  CLI flags) and TLS support via go-httpserver's `CertificateProvider` mechanism.

  **Configuration layers** (in priority order):
  1. **Defaults** — hardcoded constants for dev-friendly zero-config startup
  2. **Environment variables** — `ADJUDEX_TRANSPORT`, `ADJUDEX_ADDR`, `ADJUDEX_DB`,
     `ADJUDEX_TLS_CERT`, `ADJUDEX_TLS_KEY`
  3. **CLI flags** — `--transport`, `--addr`, `--db`, `--tls-cert`, `--tls-key`
     (override env vars)

  **TLS design**:
  - Single server instance handles both HTTP and HTTPS — no separate goroutine needed.
  - When `--tls-cert` + `--tls-key` are provided, `certificate.FileCertificateProvider`
    is injected via `httpserver.WithCertificateProvider()`.
  - `Serve()` auto-detects TLS via the certificate provider and calls
    `http.Server.ServeTLS` internally — no manual TLS config required.
  - Uses go-httpserver v0.1.0 certificate package (tdrn-org library).

  **Consequences**:
  - Removed `go-tlsconf` dependency (was imported but unused after refactoring to
    go-httpserver's native TLS support).
  - `main.go` simplified from 109 to 137 lines — more functionality, cleaner code.
  - Makefile updated: `dev` target uses in-memory DB, new `package` target for
    distributable builds.
  - Version bumped to v0.3.0.
  - Zero new non-tdrn-org dependencies.

---

## ADR-010: JSON snake_case Convention for Domain Types

**Date**: 2026-07-12
**Status**: Accepted

**Decision**: Add explicit `json:"snake_case"` tags to every exported field in all
domain types across `internal/domain/` — Portfolio, Position, Holding, Quote,
PriceHistory, IndicatorValue, Alert, IndicatorSpec, Strategy, StrategyParams
(already tagged), Trade, BacktestResult.

**Reason**: Go's `encoding/json` defaults to the exact struct field name
(PascalCase) when no `json` tag is present. The SvelteKit frontend consumed these
as `portfolio.CreatedAt` vs. the expected `portfolio.created_at` — all fields
rendered as empty/undefined with zero console errors. Adding tags is a
mechanical, stdlib-only change that fixes the mismatch at the source. 

**Consequences**:
- All JSON output from REST API and MCP tools now uses snake_case.
- TypeScript interfaces in `internal/web/src/lib/types.ts` match 1:1.
- `json:"..."` is stdlib (`encoding/json`) — does not violate domain isolation.
- MCP tools are unaffected: `encoding/json` handles marshal/unmarshal transparently.
- Future domain types MUST include `json:"..."` tags on all exported fields. 
  This is now a non-negotiable convention.
