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

package domain

import (
	"errors"
	"math"
	"testing"

	"github.com/tdrn-org/go-finance"
)

// eurPosition is a position bought in EUR.
func eurPosition(quantity, entryPrice float64) Position {
	return Position{
		Listing:    finance.Listing{},
		Currency:   finance.CurrencyEUR,
		Quantity:   quantity,
		EntryPrice: entryPrice,
	}
}

// usdPrice is a quote for the same instrument, reported in USD.
func usdPrice(price float64) finance.ListingPrice {
	return finance.ListingPrice{Currency: finance.CurrencyUSD, Price: price}
}

func approx(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestNewHoldingSameCurrencyHasPnL(t *testing.T) {
	pos := eurPosition(10, 100)
	holding := NewHolding(pos, finance.ListingPrice{Currency: finance.CurrencyEUR, Price: 110})
	if holding.PnL == nil {
		t.Fatal("expected PnL for same-currency holding")
	}
	if !approx(holding.EntryValue.Value, 1000) || !approx(holding.MarketValue.Value, 1100) {
		t.Fatalf("got entry %v market %v", holding.EntryValue.Value, holding.MarketValue.Value)
	}
	if !approx(holding.PnL.Value, 100) {
		t.Fatalf("got PnL %v, want 100", holding.PnL.Value)
	}
}

func TestNewHoldingMixedCurrencyHasNoPnL(t *testing.T) {
	// A position bought in EUR on an instrument quoted in USD has no meaningful
	// PnL until a rate is applied — the holding must not invent one.
	holding := NewHolding(eurPosition(10, 100), usdPrice(150))
	if holding.PnL != nil {
		t.Fatalf("expected no PnL for mixed-currency holding, got %v", holding.PnL)
	}
	if holding.MarketValue.Currency != finance.CurrencyUSD {
		t.Fatalf("market value currency %v, want USD", holding.MarketValue.Currency)
	}
	if holding.EntryValue.Currency != finance.CurrencyEUR {
		t.Fatalf("entry value currency %v, want EUR", holding.EntryValue.Currency)
	}
}

func TestSummarizeSingleCurrencyNeedsNoRates(t *testing.T) {
	holdings := Holdings{
		NewHolding(eurPosition(10, 100), finance.ListingPrice{Currency: finance.CurrencyEUR, Price: 110}),
		NewHolding(eurPosition(5, 200), finance.ListingPrice{Currency: finance.CurrencyEUR, Price: 180}),
	}
	summary, err := holdings.Summarize(finance.CurrencyEUR, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Currency != finance.CurrencyEUR {
		t.Fatalf("currency %v, want EUR", summary.Currency)
	}
	// market 1100 + 900 = 2000, entry 1000 + 1000 = 2000
	if !approx(summary.MarketValue, 2000) || !approx(summary.PnL, 0) {
		t.Fatalf("got market %v pnl %v", summary.MarketValue, summary.PnL)
	}
	if len(summary.Rates) != 0 {
		t.Fatalf("expected no rates, got %v", summary.Rates)
	}
}

func TestSummarizeMixedCurrencyAppliesRate(t *testing.T) {
	// 10 shares, entry 100 EUR, quoted 150 USD, USD->EUR at 0.9:
	// entry 1000 EUR, market 1500 USD = 1350 EUR, PnL 350 EUR.
	holdings := Holdings{NewHolding(eurPosition(10, 100), usdPrice(150))}
	rates := []finance.ExchangeRate{testRate(finance.CurrencyUSD, finance.CurrencyEUR, 0.9)}

	summary, err := holdings.Summarize(finance.CurrencyEUR, rates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approx(summary.MarketValue, 1350) {
		t.Fatalf("market value %v, want 1350", summary.MarketValue)
	}
	if !approx(summary.PnL, 350) {
		t.Fatalf("pnl %v, want 350", summary.PnL)
	}
	if !approx(summary.PnLPercent, 35) {
		t.Fatalf("pnl percent %v, want 35", summary.PnLPercent)
	}
	if len(summary.Rates) != 1 || summary.Rates[0].Base != finance.CurrencyUSD {
		t.Fatalf("expected the applied USD->EUR rate to be reported, got %v", summary.Rates)
	}
}

func TestSummarizeMixedCurrencyWithoutRateFails(t *testing.T) {
	holdings := Holdings{NewHolding(eurPosition(10, 100), usdPrice(150))}
	summary, err := holdings.Summarize(finance.CurrencyEUR, nil)
	if !errors.Is(err, ErrRateNotAvailable) {
		t.Fatalf("got %v, want ErrRateNotAvailable", err)
	}
	if summary != nil {
		t.Fatal("a failed summary must not return a partial result")
	}
}

func TestSummarizeMixedPortfolioTotalsInTarget(t *testing.T) {
	// One EUR position and one USD position (bought in USD), totalled in EUR.
	usdPos := Position{Currency: finance.CurrencyUSD, Quantity: 10, EntryPrice: 100}
	holdings := Holdings{
		NewHolding(eurPosition(10, 100), finance.ListingPrice{Currency: finance.CurrencyEUR, Price: 100}),
		NewHolding(usdPos, finance.ListingPrice{Currency: finance.CurrencyUSD, Price: 110}),
	}
	rates := []finance.ExchangeRate{testRate(finance.CurrencyUSD, finance.CurrencyEUR, 0.5)}

	summary, err := holdings.Summarize(finance.CurrencyEUR, rates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// EUR: entry 1000 market 1000 | USD: entry 1000 = 500 EUR, market 1100 = 550 EUR
	if !approx(summary.MarketValue, 1550) || !approx(summary.PnL, 50) {
		t.Fatalf("got market %v pnl %v", summary.MarketValue, summary.PnL)
	}
}

func TestConvertedToSetsPnL(t *testing.T) {
	holdings := Holdings{NewHolding(eurPosition(10, 100), usdPrice(150))}
	rates := []finance.ExchangeRate{testRate(finance.CurrencyUSD, finance.CurrencyEUR, 0.9)}

	converted, err := holdings.ConvertedTo(finance.CurrencyEUR, rates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(converted) != 1 {
		t.Fatalf("got %d holdings, want 1", len(converted))
	}
	h := converted[0]
	if h.MarketValue.Currency != finance.CurrencyEUR || !approx(h.MarketValue.Value, 1350) {
		t.Fatalf("market value %v, want EUR 1350", h.MarketValue)
	}
	if h.PnL == nil || !approx(h.PnL.Value, 350) {
		t.Fatalf("pnl %v, want 350", h.PnL)
	}
	// The quote itself stays in its original currency — provenance is not rewritten.
	if h.CurrentPrice.Currency != finance.CurrencyUSD || !approx(h.CurrentPrice.Price, 150) {
		t.Fatalf("current price %v, want USD 150", h.CurrentPrice)
	}
}

func TestConvertedToMissingRateFails(t *testing.T) {
	holdings := Holdings{NewHolding(eurPosition(10, 100), usdPrice(150))}
	if _, err := holdings.ConvertedTo(finance.CurrencyEUR, nil); !errors.Is(err, ErrRateNotAvailable) {
		t.Fatalf("got %v, want ErrRateNotAvailable", err)
	}
}
