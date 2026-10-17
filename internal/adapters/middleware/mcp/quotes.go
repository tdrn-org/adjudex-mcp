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
	"github.com/tdrn-org/go-finance"
)

func addQuoteTools(server *mcp.Server, runtime Runtime) {
	addQuoteGetTool(server, runtime)
	addQuoteHistoryTool(server, runtime)
	addQuoteIndicatorTool(server, runtime)
	addRateGetTool(server, runtime)
}

// validateListing guards the API boundary against a listing that carries no
// identity — for example because a client used the wrong field name or passed a
// ticker instead of the opaque listing value.
//
// finance.Listing.Validate rejects an empty listing (ErrEmptyListing) as well as
// one without provider, id keys or instrument name. Without this check one
// malformed client call takes the whole server down: a listing without internal
// data dereferences nil on its first getter.
func validateListing(listing finance.Listing) error {
	if _, err := listing.Validate(); err != nil {
		return fmt.Errorf(`invalid listing: pass the "listing" value returned by symbol_search: %w`, err)
	}
	return nil
}

// addRateGetTool registers the exchange-rate tool.
//
// Quoting and converting are kept apart on purpose: a quote always keeps the
// currency its provider reported, and a client that wants another currency asks
// for the rate explicitly and applies it itself. That way the rate used is
// always known and can be reported, instead of being baked into a price.
func addRateGetTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "rate_get",
		Description: "Returns exchange rates converting the given base currencies into a target currency. Quotes are always returned in their original currency; use these rates to present them in your own.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"target":{"type":"string","description":"Target currency (e.g. EUR). Defaults to the configured display currency."},"bases":{"type":"array","items":{"type":"string"},"description":"Base currencies to convert from (e.g. [\"USD\",\"CHF\"])."}},"required":["bases"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Target finance.Currency   `json:"target"`
			Bases  []finance.Currency `json:"bases"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}
		target := args.Target
		if target == "" {
			target = runtime.DisplayCurrency()
		}
		rates, err := runtime.MarketDataProvider().QueryExchangeRates(ctx, target, args.Bases)
		if err != nil {
			return nil, fmt.Errorf("querying rates into %s: %w", target, err)
		}
		return newToolResult(rates)
	})
}

func addQuoteGetTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "quote_get",
		Description: "Fetches the latest quote for a listing. Uses cached data when available, falls back to live provider. Pass the listing value returned by symbol_search.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"listing":{"type":"string","description":"The listing value (opaque base64 string) returned by symbol_search."}},"required":["listing"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Listing finance.Listing `json:"listing"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}
		if err := validateListing(args.Listing); err != nil {
			return nil, err
		}

		quote, err := runtime.MarketDataProvider().QueryQuote(ctx, args.Listing)
		if err != nil {
			return nil, fmt.Errorf("resolving quote for %q: %w", args.Listing.String(), err)
		}
		return newToolResult(quote)
	})
}

func addQuoteHistoryTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "quote_history",
		Description: "Returns historical quotes for a listing within a date range. Uses cached data when available, falls back to live provider. Pass the listing value returned by symbol_search.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"listing":{"type":"string","description":"The listing value (opaque base64 string) returned by symbol_search."},"from":{"type":"string","description":"Start date (RFC3339 format, e.g. 2026-06-01T00:00:00Z). Defaults to 30 days ago."},"to":{"type":"string","description":"End date (RFC3339 format). Defaults to now."}},"required":["listing"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Listing finance.Listing `json:"listing"`
			From    string          `json:"from"`
			To      string          `json:"to"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}
		if err := validateListing(args.Listing); err != nil {
			return nil, err
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

		quotes, err := runtime.DataStore().GetQuotes(ctx, args.Listing, from, to)
		if err != nil || len(quotes) == 0 {
			return nil, fmt.Errorf("fetching history for %q: %w", args.Listing.String(), err)
		}

		history := domain.PriceHistory{
			Listing: args.Listing,
			Quotes:  quotes,
		}
		return newToolResult(history)
	})
}

func addQuoteIndicatorTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "quote_indicator",
		Description: "Computes a technical indicator (SMA, EMA, RSI, MACD) for a listing over a given period. Pass the listing value returned by symbol_search.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"listing":{"type":"string","description":"The listing value (opaque base64 string) returned by symbol_search."},"type":{"type":"string","description":"Indicator type: sma, ema, rsi, macd."},"period":{"type":"integer","description":"Lookback period (e.g. 20 for SMA-20). Defaults to 20."}},"required":["listing","type"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Listing finance.Listing `json:"listing"`
			Type    string          `json:"type"`
			Period  int             `json:"period"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}
		if err := validateListing(args.Listing); err != nil {
			return nil, err
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
		quotes, err := runtime.DataStore().GetQuotes(ctx, args.Listing, from, now)
		if err != nil || len(quotes) < args.Period {
			return nil, fmt.Errorf("fetching history for %q: %w", args.Listing.String(), err)
		}

		if len(quotes) < args.Period {
			return nil, fmt.Errorf("not enough data for %q: need %d quotes, got %d", args.Listing.String(), args.Period, len(quotes))
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
			Listing:   quotes[i].Listing,
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
		Listing:   quotes[period-1].Listing,
		Type:      domain.IndicatorEMA,
		Period:    period,
		Value:     ema,
		Timestamp: quotes[period-1].Timestamp,
	})

	for i := period; i < len(quotes); i++ {
		ema = (quotes[i].Close-ema)*multiplier + ema
		values = append(values, domain.IndicatorValue{
			Listing:   quotes[i].Listing,
			Type:      domain.IndicatorEMA,
			Period:    period,
			Value:     ema,
			Timestamp: quotes[i].Timestamp,
		})
	}
	return values
}
