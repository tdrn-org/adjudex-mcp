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
	"fmt"

	"github.com/tdrn-org/go-finance"
)

// ErrCurrencyMismatch signals an operation combining amounts of different
// currencies without an explicit conversion.
var ErrCurrencyMismatch = errors.New("currency mismatch")

// ErrRateNotAvailable signals that no exchange rate is available to convert
// between two currencies.
var ErrRateNotAvailable = errors.New("exchange rate not available")

// Amount is a monetary value bound to the currency it is denominated in.
//
// Unlike a bare float64 an Amount carries its currency, so a monetary value can
// never be silently combined with a value denominated in another currency.
// Amounts of different currencies are not comparable: convert them explicitly
// via [Amount.Convert] first. This is what keeps a portfolio containing several
// currencies from producing a meaningless total.
type Amount struct {
	// Currency is the currency this value is denominated in.
	Currency finance.Currency `json:"currency"`
	// Value is the monetary value in Currency.
	Value float64 `json:"value"`
}

// NewAmount creates an Amount denominated in the given currency.
func NewAmount(currency finance.Currency, value float64) Amount {
	return Amount{Currency: currency, Value: value}
}

// ZeroAmount creates a zero Amount denominated in the given currency.
func ZeroAmount(currency finance.Currency) Amount {
	return Amount{Currency: currency}
}

// Add adds other to a. Both amounts must be denominated in the same currency.
func (a Amount) Add(other Amount) (Amount, error) {
	if a.Currency != other.Currency {
		return Amount{}, fmt.Errorf("add %s to %s: %w", other.Currency, a.Currency, ErrCurrencyMismatch)
	}
	return Amount{Currency: a.Currency, Value: a.Value + other.Value}, nil
}

// Sub subtracts other from a. Both amounts must be denominated in the same currency.
func (a Amount) Sub(other Amount) (Amount, error) {
	if a.Currency != other.Currency {
		return Amount{}, fmt.Errorf("subtract %s from %s: %w", other.Currency, a.Currency, ErrCurrencyMismatch)
	}
	return Amount{Currency: a.Currency, Value: a.Value - other.Value}, nil
}

// Scale multiplies a by the given factor. The currency is kept.
func (a Amount) Scale(factor float64) Amount {
	return Amount{Currency: a.Currency, Value: a.Value * factor}
}

// Convert converts a into the given target currency using the given exchange
// rates. The conversion is explicit: if no rate is available
// [ErrRateNotAvailable] is returned instead of guessing a value. If a is already
// denominated in target, it is returned unchanged and no rate is needed.
func (a Amount) Convert(target finance.Currency, rates []finance.ExchangeRate) (Amount, error) {
	if a.Currency == target {
		return a, nil
	}
	rate, err := FindExchangeRate(rates, a.Currency, target)
	if err != nil {
		return Amount{}, err
	}
	return Amount{Currency: target, Value: a.Value * rate.Rate}, nil
}

// FindExchangeRate looks up the rate converting base into quote.
//
// A rate quoted the other way round is used inverted, so callers do not have to
// care which direction a provider happens to publish. The returned rate keeps
// the source and timestamps of the original quote: an inverted rate is derived
// data, and its provenance is the rate it was derived from.
func FindExchangeRate(rates []finance.ExchangeRate, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	for i := range rates {
		if rates[i].Base == base && rates[i].Quote == quote {
			return &rates[i], nil
		}
	}
	for i := range rates {
		if rates[i].Base == quote && rates[i].Quote == base && rates[i].Rate != 0 {
			inverted := rates[i]
			inverted.Base = base
			inverted.Quote = quote
			inverted.Rate = 1 / rates[i].Rate
			return &inverted, nil
		}
	}
	return nil, fmt.Errorf("%s->%s: %w", base, quote, ErrRateNotAvailable)
}
