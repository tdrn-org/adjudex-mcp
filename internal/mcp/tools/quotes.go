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

// --- Tool: quote_get ---

// QuoteGetArgs are the parameters for quote_get.
type QuoteGetArgs struct {
	Symbol string `json:"symbol"`
}

// QuoteGet returns the latest quote for a symbol.
// Store: domain.QuoteStore.GetLatestQuote
func QuoteGet(ctx context.Context, store domain.QuoteStore, args QuoteGetArgs) (*domain.Quote, error) {
	panic("not implemented: Phase 5")
}

// --- Tool: quote_history ---

// QuoteHistoryArgs are the parameters for quote_history.
type QuoteHistoryArgs struct {
	Symbol string `json:"symbol"`
	From   string `json:"from"` // RFC3339
	To     string `json:"to"`   // RFC3339
}

// QuoteHistory returns historical quotes for a symbol within a date range.
// Store: domain.QuoteStore.GetQuotes
func QuoteHistory(ctx context.Context, store domain.QuoteStore, args QuoteHistoryArgs) ([]domain.Quote, error) {
	_ = ctx
	_ = store
	_ = args
	panic("not implemented: Phase 5")
}

// --- Tool: quote_indicator ---

// QuoteIndicatorArgs are the parameters for quote_indicator.
type QuoteIndicatorArgs struct {
	Symbol        string `json:"symbol"`
	IndicatorType string `json:"indicator_type"` // rsi, sma, ema, macd
	Period        int    `json:"period"`
}

// QuoteIndicator computes a technical indicator from historical quotes.
// Computed: loads quotes via QuoteStore.GetQuotes, then calculates the indicator.
func QuoteIndicator(ctx context.Context, store domain.QuoteStore, args QuoteIndicatorArgs) (*domain.IndicatorValue, error) {
	_ = ctx
	_ = store
	_ = args
	panic("not implemented: Phase 5")
}

// Ensure imported types are referenced (avoid compile errors in stub phase).
var _ = time.Now
