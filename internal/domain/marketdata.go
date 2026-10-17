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
	"context"

	"github.com/tdrn-org/go-finance"
)

// MarketData gives access to market data: instrument lookup, quotes and the
// exchange rates needed to present those quotes in another currency.
//
// Quotes are always returned in the currency their provider reports — never
// converted. Conversion is the caller's decision and must be explicit, so the
// rate applied is known and can be reported alongside the result.
type MarketData interface {
	// SearchListings searches for listings matching the given query.
	SearchListings(ctx context.Context, query string) ([]finance.Listing, error)
	// QueryQuote returns the latest quote for the given listing, denominated in
	// the currency reported by the provider.
	QueryQuote(ctx context.Context, listing finance.Listing) (*Quote, error)
	// QueryExchangeRate returns the rate converting base into quote.
	QueryExchangeRate(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error)
	// QueryExchangeRates returns the rates converting each of the given base
	// currencies into target. Currencies already denominated in target are
	// skipped. Rates are collected in one call so a warming-up provider is
	// waited for once instead of once per currency.
	QueryExchangeRates(ctx context.Context, target finance.Currency, bases []finance.Currency) ([]finance.ExchangeRate, error)
}
