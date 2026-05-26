/*
 * Copyright 2026 Holger de Carne
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tools

import (
	"context"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
)

// --- Tool: strategy_create ---

// StrategyCreateArgs are the parameters for strategy_create.
type StrategyCreateArgs struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Params      domain.StrategyParams `json:"params"`
}

// StrategyCreate creates a new trading strategy.
// Store: domain.StrategyStore.SaveStrategy
func StrategyCreate(ctx context.Context, store domain.StrategyStore, args StrategyCreateArgs) (*domain.Strategy, error) {
	panic("not implemented: Phase 5")
}

// --- Tool: strategy_get ---

// StrategyGetArgs are the parameters for strategy_get.
type StrategyGetArgs struct {
	ID string `json:"id"`
}

// StrategyGet retrieves a strategy by ID.
// Store: domain.StrategyStore.GetStrategy
func StrategyGet(ctx context.Context, store domain.StrategyStore, args StrategyGetArgs) (*domain.Strategy, error) {
	panic("not implemented: Phase 5")
}

// --- Tool: strategy_list ---

// StrategyListArgs are the parameters for strategy_list (no parameters).
type StrategyListArgs struct{}

// StrategyList returns all strategies.
// Store: domain.StrategyStore.ListStrategies
func StrategyList(ctx context.Context, store domain.StrategyStore, args StrategyListArgs) ([]domain.Strategy, error) {
	panic("not implemented: Phase 5")
}

// --- Tool: strategy_delete ---

// StrategyDeleteArgs are the parameters for strategy_delete.
type StrategyDeleteArgs struct {
	ID string `json:"id"`
}

// StrategyDelete removes a strategy.
// Store: domain.StrategyStore.DeleteStrategy
func StrategyDelete(ctx context.Context, store domain.StrategyStore, args StrategyDeleteArgs) error {
	panic("not implemented: Phase 5")
}

// --- Tool: strategy_backtest ---

// StrategyBacktestArgs are the parameters for strategy_backtest.
type StrategyBacktestArgs struct {
	StrategyID string `json:"strategy_id"`
	Symbol     string `json:"symbol"`
	From       string `json:"from"` // RFC3339
	To         string `json:"to"`   // RFC3339
}

// StrategyBacktest runs a strategy against historical data and returns results.
// Complex operation: loads Strategy + Quote history + Trade history, simulates execution.
// Stores: domain.StrategyStore.GetStrategy + domain.QuoteStore.GetQuotes + domain.TradeStore.ListTradesByStrategy
func StrategyBacktest(ctx context.Context, strategyStore domain.StrategyStore, quoteStore domain.QuoteStore, tradeStore domain.TradeStore, args StrategyBacktestArgs) (*domain.BacktestResult, error) {
	_ = ctx
	_ = strategyStore
	_ = quoteStore
	_ = tradeStore
	_ = args
	panic("not implemented: Phase 5")
}

// Ensure imported types are referenced (avoid compile errors in stub phase).
var _ = time.Now
