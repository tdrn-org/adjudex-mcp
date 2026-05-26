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
