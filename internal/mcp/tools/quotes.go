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
	"fmt"
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
	return store.GetLatestQuote(ctx, args.Symbol)
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
	from, err := time.Parse(time.RFC3339, args.From)
	if err != nil {
		return nil, fmt.Errorf("quote_history: parse from: %w", err)
	}
	to, err := time.Parse(time.RFC3339, args.To)
	if err != nil {
		return nil, fmt.Errorf("quote_history: parse to: %w", err)
	}
	return store.GetQuotes(ctx, args.Symbol, from, to)
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
	indicatorType := domain.IndicatorType(args.IndicatorType)
	// Load enough history for the indicator calculation.
	// Use 2x the period for SMA/EMA stability, 5x for RSI.
	lookback := args.Period * 5
	to := time.Now()
	from := to.AddDate(0, 0, -lookback)

	quotes, err := store.GetQuotes(ctx, args.Symbol, from, to)
	if err != nil {
		return nil, fmt.Errorf("quote_indicator: load quotes: %w", err)
	}
	if len(quotes) < args.Period {
		return nil, fmt.Errorf("quote_indicator: need at least %d quotes, got %d", args.Period, len(quotes))
	}

	closes := make([]float64, len(quotes))
	for i, q := range quotes {
		closes[i] = q.Close
	}

	var value float64
	switch indicatorType {
	case domain.IndicatorRSI:
		value = computeRSI(closes, args.Period)
	case domain.IndicatorSMA:
		value = computeSMA(closes, args.Period)
	case domain.IndicatorEMA:
		value = computeEMA(closes, args.Period)
	case domain.IndicatorMACD:
		value = computeMACD(closes, args.Period)
	default:
		return nil, fmt.Errorf("quote_indicator: unknown indicator type %q", args.IndicatorType)
	}

	return &domain.IndicatorValue{
		Symbol:    args.Symbol,
		Type:      indicatorType,
		Period:    args.Period,
		Value:     value,
		Timestamp: quotes[len(quotes)-1].Timestamp,
	}, nil
}

// computeRSI calculates the Relative Strength Index for the last period from closes.
func computeRSI(closes []float64, period int) float64 {
	n := len(closes)
	if n < period+1 {
		return 0
	}
	var gains, losses float64
	for i := n - period; i < n; i++ {
		change := closes[i] - closes[i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}
	if losses == 0 {
		return 100
	}
	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

// computeSMA returns the simple moving average of the last period closes.
func computeSMA(closes []float64, period int) float64 {
	n := len(closes)
	if n < period {
		return 0
	}
	sum := 0.0
	for i := n - period; i < n; i++ {
		sum += closes[i]
	}
	return sum / float64(period)
}

// computeEMA returns the exponential moving average of closes.
func computeEMA(closes []float64, period int) float64 {
	n := len(closes)
	if n < period {
		return 0
	}
	multiplier := 2.0 / float64(period+1)
	ema := closes[0]
	for _, price := range closes[1:] {
		ema = (price-ema)*multiplier + ema
	}
	return ema
}

// computeMACD returns the MACD line value (12-period EMA minus 26-period EMA).
func computeMACD(closes []float64, period int) float64 {
	_ = period // MACD period is fixed; period parameter controls signal line
	if len(closes) < 26 {
		return 0
	}
	ema12 := computeEMA(closes, 12)
	ema26 := computeEMA(closes, 26)
	return ema12 - ema26
}

// Keep imported types referenced for the stub phase.
var _ = time.Now
