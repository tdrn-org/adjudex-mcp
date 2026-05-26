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

// --- Tool: trade_list ---

// TradeListArgs are the parameters for trade_list.
type TradeListArgs struct {
	Symbol     string `json:"symbol,omitempty"`     // optional: filter by symbol
	StrategyID string `json:"strategy_id,omitempty"` // optional: filter by strategy
}

// TradeList returns trades, optionally filtered by symbol or strategy.
// Stores: domain.TradeStore.ListTrades / domain.TradeStore.ListTradesByStrategy
func TradeList(ctx context.Context, store domain.TradeStore, args TradeListArgs) ([]domain.Trade, error) {
	panic("not implemented: Phase 5")
}

// --- Tool: trade_get ---

// TradeGetArgs are the parameters for trade_get.
type TradeGetArgs struct {
	ID string `json:"id"`
}

// TradeGet retrieves a single trade by ID.
// Store: domain.TradeStore.GetTrade
func TradeGet(ctx context.Context, store domain.TradeStore, args TradeGetArgs) (*domain.Trade, error) {
	panic("not implemented: Phase 5")
}

// Ensure imported types are referenced (avoid compile errors in stub phase).
var _ = time.Now
