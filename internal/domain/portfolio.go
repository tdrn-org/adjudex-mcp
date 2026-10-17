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
	"encoding/json"
	"fmt"
	"time"

	"github.com/tdrn-org/go-finance"
)

// Portfolio represents a named collection of positions (watchlist or holdings).
type Portfolio struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Positions   []Position `json:"positions"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Position represents a tracked security within a portfolio.
//
// Currency is the currency EntryPrice is denominated in — a booked fact that is
// fixed once the position exists. It is NOT the currency the instrument is
// quoted in: a position bought in EUR on a US stock keeps an EUR entry price
// while its quote comes in USD.
type Position struct {
	ID         string           `json:"id"`
	Listing    finance.Listing  `json:"listing"`
	Currency   finance.Currency `json:"currency"`
	Quantity   float64          `json:"quantity"`
	EntryPrice float64          `json:"entry_price"`
	EntryDate  time.Time        `json:"entry_date"`
	Notes      string           `json:"notes,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

// MarshalJSON adds the human readable listing description to every position.
//
// The listing ID names an instrument but does not describe it, so a UI could only
// render the ID where the instrument name belongs. Embedding ListingInfo keeps the
// payload self-describing while the ID stays usable for follow-up requests. It is
// derived on the way out, so the persisted representation and the domain value
// remain unchanged.
//
// The alias type is essential: embedding Position itself would re-enter this
// method and overflow the stack.
func (p Position) MarshalJSON() ([]byte, error) {
	type positionAlias Position
	return json.Marshal(struct {
		positionAlias
		ListingInfo ListingInfo `json:"listing_info"`
	}{positionAlias(p), listingInfoOrZero(p.Listing)})
}

// listingInfoOrZero describes a listing for display without panicking on an
// unset or invalid one — marshalling must never fail because of a broken listing.
func listingInfoOrZero(listing finance.Listing) ListingInfo {
	if _, err := listing.Validate(); err != nil {
		return ListingInfo{}
	}
	return NewListingInfo(listing)
}

// Holding is a Position enriched with current market data.
// It is not persisted — derived at query time from Position + Quote.
//
// EntryValue and MarketValue are separate amounts because they are usually
// denominated in different currencies: the entry price comes from the booking,
// the market price from the quote. PnL is therefore only set when both sides
// share a currency; otherwise the caller has to convert explicitly via
// [Holdings.Summarize].
type Holding struct {
	Position
	// CurrentPrice is the price the market value is based on, including its
	// currency and provenance.
	CurrentPrice finance.ListingPrice `json:"current_price"`
	// EntryValue is quantity x entry price, denominated in the position currency.
	EntryValue Amount `json:"entry_value"`
	// MarketValue is quantity x current price, denominated in the price currency.
	MarketValue Amount `json:"market_value"`
	// PnL is only set when EntryValue and MarketValue share a currency.
	PnL *Amount `json:"pnl,omitempty"`
	// PnLPercent is only set when PnL is set.
	PnLPercent *float64 `json:"pnl_percent,omitempty"`
}

// MarshalJSON keeps the holding payload flat.
//
// Holding embeds Position, which has its own MarshalJSON. Without an explicit
// implementation the embedded position would be nested under a "Position" key and
// break every consumer of the holdings endpoint, so the fields are listed
// explicitly here.
func (h Holding) MarshalJSON() ([]byte, error) {
	type positionAlias Position
	return json.Marshal(struct {
		positionAlias
		CurrentPrice finance.ListingPrice `json:"current_price"`
		EntryValue   Amount               `json:"entry_value"`
		MarketValue  Amount               `json:"market_value"`
		PnL          *Amount              `json:"pnl,omitempty"`
		PnLPercent   *float64             `json:"pnl_percent,omitempty"`
		ListingInfo  ListingInfo          `json:"listing_info"`
	}{
		positionAlias: positionAlias(h.Position),
		CurrentPrice:  h.CurrentPrice,
		EntryValue:    h.EntryValue,
		MarketValue:   h.MarketValue,
		PnL:           h.PnL,
		PnLPercent:    h.PnLPercent,
		ListingInfo:   listingInfoOrZero(h.Listing),
	})
}

// NewHolding derives a Holding from a Position and its current price.
// Returns by-value (not pointer) because Holding is a derived Value Object —
// it is never persisted or mutated, only computed and consumed.
func NewHolding(pos Position, currentPrice finance.ListingPrice) Holding {
	entryValue := NewAmount(pos.Currency, pos.Quantity*pos.EntryPrice)
	marketValue := NewAmount(currentPrice.Currency, pos.Quantity*currentPrice.Price)
	holding := Holding{
		Position:     pos,
		CurrentPrice: currentPrice,
		EntryValue:   entryValue,
		MarketValue:  marketValue,
	}
	if pnl, err := marketValue.Sub(entryValue); err == nil {
		var pnlPercent float64
		if entryValue.Value > 0 {
			pnlPercent = pnl.Value / entryValue.Value * 100
		}
		holding.PnL = &pnl
		holding.PnLPercent = &pnlPercent
	}
	return holding
}

type Holdings []Holding

// ConvertedTo returns the holdings with entry and market values expressed in
// target, using the given exchange rates. PnL becomes available as soon as both
// sides share the target currency, so a position bought in EUR and quoted in USD
// gains a PnL only after this explicit conversion.
//
// CurrentPrice is left untouched: it is the quote as the provider reported it,
// in its original currency, and stays that way.
func (holdings Holdings) ConvertedTo(target finance.Currency, rates []finance.ExchangeRate) (Holdings, error) {
	converted := make(Holdings, 0, len(holdings))
	for _, holding := range holdings {
		entry, err := holding.EntryValue.Convert(target, rates)
		if err != nil {
			return nil, fmt.Errorf("convert %q: entry value: %w", holding.Listing.ID(), err)
		}
		market, err := holding.MarketValue.Convert(target, rates)
		if err != nil {
			return nil, fmt.Errorf("convert %q: market value: %w", holding.Listing.ID(), err)
		}
		h := holding
		h.EntryValue = entry
		h.MarketValue = market
		if pnl, err := market.Sub(entry); err == nil {
			var pnlPercent float64
			if entry.Value > 0 {
				pnlPercent = pnl.Value / entry.Value * 100
			}
			h.PnL = &pnl
			h.PnLPercent = &pnlPercent
		}
		converted = append(converted, h)
	}
	return converted, nil
}

// HoldingsSummary is a portfolio total expressed in exactly one currency.
//
// Rates records every exchange rate that was applied to reach that currency.
// Without it the total would be arithmetically correct but unverifiable: the
// reader could not tell which rates produced it.
type HoldingsSummary struct {
	// Currency is the currency the total is expressed in.
	Currency finance.Currency `json:"currency"`
	// MarketValue is the total market value in Currency.
	MarketValue float64 `json:"market_value"`
	// PnL is the total profit/loss in Currency.
	PnL float64 `json:"pnl"`
	// PnLPercent is PnL relative to the total entry value.
	PnLPercent float64 `json:"pnl_percent"`
	// Rates lists the exchange rates applied to build this summary.
	Rates []finance.ExchangeRate `json:"rates,omitempty"`
}

// Summarize totals the holdings in the given target currency.
//
// A portfolio may hold positions in several currencies, so reaching a single
// total requires conversion. If any needed exchange rate is missing,
// [ErrRateNotAvailable] is returned and NO partial total is produced: a total
// that silently omits a position is worse than no total at all, because it
// looks complete.
func (holdings Holdings) Summarize(target finance.Currency, rates []finance.ExchangeRate) (*HoldingsSummary, error) {
	totalEntry := ZeroAmount(target)
	totalMarket := ZeroAmount(target)
	applied := make([]finance.ExchangeRate, 0, 2)
	appliedKeys := make(map[string]bool)

	convert := func(a Amount) (Amount, error) {
		if a.Currency == target {
			return a, nil
		}
		rate, err := FindExchangeRate(rates, a.Currency, target)
		if err != nil {
			return Amount{}, err
		}
		key := string(rate.Base) + "/" + string(rate.Quote)
		if !appliedKeys[key] {
			appliedKeys[key] = true
			applied = append(applied, *rate)
		}
		return Amount{Currency: target, Value: a.Value * rate.Rate}, nil
	}

	for _, holding := range holdings {
		entry, err := convert(holding.EntryValue)
		if err != nil {
			return nil, fmt.Errorf("summarize %q: entry value: %w", holding.Listing.ID(), err)
		}
		market, err := convert(holding.MarketValue)
		if err != nil {
			return nil, fmt.Errorf("summarize %q: market value: %w", holding.Listing.ID(), err)
		}
		if totalEntry, err = totalEntry.Add(entry); err != nil {
			return nil, err
		}
		if totalMarket, err = totalMarket.Add(market); err != nil {
			return nil, err
		}
	}

	pnl := totalMarket.Value - totalEntry.Value
	var pnlPercent float64
	if totalEntry.Value > 0 {
		pnlPercent = pnl / totalEntry.Value * 100
	}
	return &HoldingsSummary{
		Currency:    target,
		MarketValue: totalMarket.Value,
		PnL:         pnl,
		PnLPercent:  pnlPercent,
		Rates:       applied,
	}, nil
}
