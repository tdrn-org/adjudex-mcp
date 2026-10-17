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

// resolveListing replaces the listing a caller passed with the one the store
// holds for its ID.
//
// A listing on the wire carries the ID only: instrument name and type are not
// part of it, and a caller cannot be trusted to report them. Resolving against
// the store proves that the server issued this listing in the first place and
// supplies the description the response needs. A listing the store does not know
// is rejected - it must be obtained via listing_search first.
func resolveListing(ctx context.Context, runtime Runtime, listing finance.Listing) (finance.Listing, error) {
	resolved, err := runtime.DataStore().ResolveListing(ctx, listing)
	if err != nil {
		return listing, fmt.Errorf(`cannot use listing %q: %w`, listing.ID(), err)
	}
	return resolved, nil
}

// validateOptionalListing guards a listing that only acts as a filter: an unset
// listing means "no filter" and must not be rejected.
//
// A filter is matched by ID, so nothing is resolved here. An ID that matches
// nothing simply yields an empty result - unlike a listing that is used, which
// has to resolve, see [resolveListing].
func validateOptionalListing(listing finance.Listing) error {
	if listing.ID() == "" {
		return nil
	}
	_, err := finance.ParseListingID(listing.ID())
	if err != nil {
		return fmt.Errorf("invalid listing filter: %w", err)
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
		Description: "Fetches the latest quote for a listing. Uses cached data when available, falls back to live provider. Pass the listing ID returned by listing_search.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"listing":{"type":"string","description":"The listing ID (e.g. \"twelvedata:ticker=AAPL:mic=XNGS\") returned by listing_search."}},"required":["listing"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Listing finance.Listing `json:"listing"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}
		listing, err := resolveListing(ctx, runtime, args.Listing)
		if err != nil {
			return nil, err
		}

		quote, err := runtime.MarketDataProvider().QueryQuote(ctx, listing)
		if err != nil {
			return nil, fmt.Errorf("resolving quote for %q: %w", listing.String(), err)
		}
		return newToolResult(quote)
	})
}

func addQuoteHistoryTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "quote_history",
		Description: "Returns historical quotes for a listing within a date range. Uses cached data when available, falls back to live provider. Pass the listing ID returned by listing_search.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"listing":{"type":"string","description":"The listing ID (e.g. \"twelvedata:ticker=AAPL:mic=XNGS\") returned by listing_search."},"from":{"type":"string","description":"Start date (RFC3339 format, e.g. 2026-06-01T00:00:00Z). Defaults to 30 days ago."},"to":{"type":"string","description":"End date (RFC3339 format). Defaults to now."}},"required":["listing"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Listing finance.Listing `json:"listing"`
			From    string          `json:"from"`
			To      string          `json:"to"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}
		listing, err := resolveListing(ctx, runtime, args.Listing)
		if err != nil {
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

		quotes, err := runtime.DataStore().GetQuotes(ctx, listing, from, to)
		if err != nil || len(quotes) == 0 {
			return nil, fmt.Errorf("fetching history for %q: %w", listing.String(), err)
		}

		history := domain.PriceHistory{
			Listing: listing,
			Quotes:  quotes,
		}
		return newToolResult(history)
	})
}

func addQuoteIndicatorTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "quote_indicator",
		Description: "Computes a technical indicator (SMA, EMA, RSI, MACD) for a listing over a given period. Pass the listing ID returned by listing_search.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"listing":{"type":"string","description":"The listing ID (e.g. \"twelvedata:ticker=AAPL:mic=XNGS\") returned by listing_search."},"type":{"type":"string","description":"Indicator type: sma, ema, rsi, macd."},"period":{"type":"integer","description":"Lookback period (e.g. 20 for SMA-20). Defaults to 20."}},"required":["listing","type"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Listing finance.Listing `json:"listing"`
			Type    string          `json:"type"`
			Period  int             `json:"period"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}
		listing, err := resolveListing(ctx, runtime, args.Listing)
		if err != nil {
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
		quotes, err := runtime.DataStore().GetQuotes(ctx, listing, from, now)
		if err != nil || len(quotes) < args.Period {
			return nil, fmt.Errorf("fetching history for %q: %w", listing.String(), err)
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
