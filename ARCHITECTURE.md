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
  A bug fix in a tool function automatically fixes both the MCP and REST interfaces.
