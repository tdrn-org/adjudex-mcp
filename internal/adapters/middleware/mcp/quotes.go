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

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tdrn-org/adjudex-mcp/internal/domain"
)

func addQuoteTools(server *mcp.Server, runtime Runtime) {
	addQuoteGetTool(server, runtime)
	addQuoteHistoryTool(server, runtime)
	addQuoteIndicatorTool(server, runtime)
}

func addQuoteGetTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "quote_get",
		Description: "Fetches the latest quote for a symbol. Uses cached data when available, falls back to live provider.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"symbol":{"type":"string","description":"The ticker symbol (e.g. NVDA, CRWV, SPCX)."}},"required":["symbol"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Symbol string `json:"symbol"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		quote, err := runtime.QuoteService().ResolveQuote(ctx, args.Symbol)
		if err != nil {
			return nil, fmt.Errorf("resolving quote for %q: %w", args.Symbol, err)
		}
		return newToolResult(quote)
	})
}

func addQuoteHistoryTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "quote_history",
		Description: "Returns historical quotes for a symbol within a date range. Uses cached data when available, falls back to live provider.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"symbol":{"type":"string","description":"The ticker symbol."},"from":{"type":"string","description":"Start date (RFC3339 format, e.g. 2026-06-01T00:00:00Z). Defaults to 30 days ago."},"to":{"type":"string","description":"End date (RFC3339 format). Defaults to now."}},"required":["symbol"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Symbol string `json:"symbol"`
			From   string `json:"from"`
			To     string `json:"to"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		now := time.Now()
		to := now
		from := now.AddDate(0, -1, 0)

		if args.From != "" {
			var err error
			from, err = time.Parse(time.RFC3339, args.From)
			if err != nil {
				return nil, fmt.Errorf("parsing from date: %w", err)
			}
		}
		if args.To != "" {
			var err error
			to, err = time.Parse(time.RFC3339, args.To)
			if err != nil {
				return nil, fmt.Errorf("parsing to date: %w", err)
			}
		}

		quotes, err := runtime.DataStore().GetQuotes(ctx, args.Symbol, from, to)
		if err != nil || len(quotes) == 0 {
			runtime.Logger().Info("quotes not in store, fetching live", "symbol", args.Symbol)
			quotes, err = runtime.QuoteService().FetchHistory(ctx, args.Symbol, from, to)
			if err != nil {
				return nil, fmt.Errorf("fetching history for %q: %w", args.Symbol, err)
			}
		}

		history := domain.PriceHistory{
			Symbol: args.Symbol,
			Quotes: quotes,
		}
		return newToolResult(history)
	})
}

func addQuoteIndicatorTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "quote_indicator",
		Description: "Computes a technical indicator (SMA, EMA, RSI, MACD) for a symbol over a given period.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"symbol":{"type":"string","description":"The ticker symbol."},"type":{"type":"string","description":"Indicator type: sma, ema, rsi, macd."},"period":{"type":"integer","description":"Lookback period (e.g. 20 for SMA-20). Defaults to 20."}},"required":["symbol","type"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Symbol string `json:"symbol"`
			Type   string `json:"type"`
			Period int    `json:"period"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		if args.Period <= 0 {
			args.Period = 20
		}

		indicatorType := domain.IndicatorType(args.Type)
		switch indicatorType {
		case domain.IndicatorSMA, domain.IndicatorEMA, domain.IndicatorRSI, domain.IndicatorMACD:
		default:
			return nil, fmt.Errorf("unknown indicator type %q (valid: sma, ema, rsi, macd)", args.Type)
		}

		now := time.Now()
		from := now.AddDate(0, -3, 0)
		quotes, err := runtime.DataStore().GetQuotes(ctx, args.Symbol, from, now)
		if err != nil || len(quotes) < args.Period {
			runtime.Logger().Info("not enough quotes in store, fetching live", "symbol", args.Symbol)
			quotes, err = runtime.QuoteService().FetchHistory(ctx, args.Symbol, from, now)
			if err != nil {
				return nil, fmt.Errorf("fetching history for %q: %w", args.Symbol, err)
			}
		}

		if len(quotes) < args.Period {
			return nil, fmt.Errorf("not enough data for %q: need %d quotes, got %d", args.Symbol, args.Period, len(quotes))
		}

		values := computeIndicator(quotes, indicatorType, args.Period)
		return newToolResult(values)
	})
}

func computeIndicator(quotes []domain.Quote, indicatorType domain.IndicatorType, period int) []domain.IndicatorValue {
	switch indicatorType {
	case domain.IndicatorSMA:
		return computeSMA(quotes, period)
	case domain.IndicatorEMA:
		return computeEMA(quotes, period)
	default:
		return nil
	}
}

func computeSMA(quotes []domain.Quote, period int) []domain.IndicatorValue {
	if len(quotes) < period {
		return nil
	}
	values := make([]domain.IndicatorValue, 0, len(quotes)-period+1)
	for i := period - 1; i < len(quotes); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += quotes[j].Close
		}
		values = append(values, domain.IndicatorValue{
			Symbol:    quotes[i].Symbol,
			Type:      domain.IndicatorSMA,
			Period:    period,
			Value:     sum / float64(period),
			Timestamp: quotes[i].Timestamp,
		})
	}
	return values
}

func computeEMA(quotes []domain.Quote, period int) []domain.IndicatorValue {
	if len(quotes) < period {
		return nil
	}
	values := make([]domain.IndicatorValue, 0, len(quotes)-period+1)
	multiplier := 2.0 / float64(period+1)

	sum := 0.0
	for i := 0; i < period; i++ {
		sum += quotes[i].Close
	}
	ema := sum / float64(period)
	values = append(values, domain.IndicatorValue{
		Symbol:    quotes[period-1].Symbol,
		Type:      domain.IndicatorEMA,
		Period:    period,
		Value:     ema,
		Timestamp: quotes[period-1].Timestamp,
	})

	for i := period; i < len(quotes); i++ {
		ema = (quotes[i].Close-ema)*multiplier + ema
		values = append(values, domain.IndicatorValue{
			Symbol:    quotes[i].Symbol,
			Type:      domain.IndicatorEMA,
			Period:    period,
			Value:     ema,
			Timestamp: quotes[i].Timestamp,
		})
	}
	return values
}
