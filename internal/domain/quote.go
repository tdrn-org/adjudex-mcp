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
	"time"

	"github.com/tdrn-org/go-finance"
)

// Quote is a full price data point for a listing.
//
// A Quote is a fact reported by a provider and is therefore immutable: its
// prices are denominated in the currency carried by the embedded
// [finance.ListingPrice] and are never rescaled afterwards. Currency conversion
// is a presentation concern of the caller — converting here would leave the
// quote claiming a provenance it does not have (provider X, currency Y, while
// the number is actually provider X multiplied by an FX rate from provider Z).
type Quote finance.Quote

type Quotes []Quote

// IsUniform reports whether all quotes share the same currency, returning that
// currency. A set of quotes spanning several currencies has no single currency:
// summing it without conversion would produce a meaningless number.
func (qs Quotes) IsUniform() (finance.Currency, bool) {
	if len(qs) == 0 {
		return "", true
	}
	currency := qs[0].Currency
	for i := 1; i < len(qs); i++ {
		if qs[i].Currency != currency {
			return "", false
		}
	}
	return currency, true
}

// PriceHistory is a chronological series of quotes for a single symbol.
type PriceHistory struct {
	Listing finance.Listing `json:"listing"`
	Quotes  []Quote         `json:"quotes"`
}

// IndicatorType names the technical indicator.
type IndicatorType string

const (
	IndicatorRSI  IndicatorType = "rsi"
	IndicatorSMA  IndicatorType = "sma"
	IndicatorEMA  IndicatorType = "ema"
	IndicatorMACD IndicatorType = "macd"
)

// IndicatorValue holds a computed indicator value at a point in time.
type IndicatorValue struct {
	Listing   finance.Listing  `json:"listing"`
	Currency  finance.Currency `json:"currency"`
	Type      IndicatorType    `json:"type"`
	Period    int              `json:"period"`
	Value     float64          `json:"value"`
	Timestamp time.Time        `json:"timestamp"`
}
