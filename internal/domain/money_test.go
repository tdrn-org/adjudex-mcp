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
	"testing"
	"time"

	"github.com/tdrn-org/go-finance"
)

func testRate(base, quote finance.Currency, rate float64) finance.ExchangeRate {
	ts := time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC)
	return finance.ExchangeRate{
		Timestamp:       ts,
		Base:            base,
		Quote:           quote,
		Rate:            rate,
		Source:          "test",
		SourceTimestamp: ts,
	}
}

func TestAmountAddSameCurrency(t *testing.T) {
	sum, err := NewAmount(finance.CurrencyEUR, 10).Add(NewAmount(finance.CurrencyEUR, 5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum.Currency != finance.CurrencyEUR || sum.Value != 15 {
		t.Fatalf("got %v %v, want EUR 15", sum.Currency, sum.Value)
	}
}

func TestAmountAddDifferentCurrency(t *testing.T) {
	_, err := NewAmount(finance.CurrencyEUR, 10).Add(NewAmount(finance.CurrencyUSD, 5))
	if !errors.Is(err, ErrCurrencyMismatch) {
		t.Fatalf("got %v, want ErrCurrencyMismatch", err)
	}
}

func TestAmountSubDifferentCurrency(t *testing.T) {
	_, err := NewAmount(finance.CurrencyEUR, 10).Sub(NewAmount(finance.CurrencyUSD, 5))
	if !errors.Is(err, ErrCurrencyMismatch) {
		t.Fatalf("got %v, want ErrCurrencyMismatch", err)
	}
}

func TestAmountConvertSameCurrencyNeedsNoRate(t *testing.T) {
	converted, err := NewAmount(finance.CurrencyEUR, 10).Convert(finance.CurrencyEUR, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if converted.Value != 10 || converted.Currency != finance.CurrencyEUR {
		t.Fatalf("got %v, want EUR 10", converted)
	}
}

func TestAmountConvertAppliesRate(t *testing.T) {
	rates := []finance.ExchangeRate{testRate(finance.CurrencyUSD, finance.CurrencyEUR, 0.9)}
	converted, err := NewAmount(finance.CurrencyUSD, 100).Convert(finance.CurrencyEUR, rates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if converted.Currency != finance.CurrencyEUR || converted.Value != 90 {
		t.Fatalf("got %v, want EUR 90", converted)
	}
}

func TestAmountConvertInvertsRate(t *testing.T) {
	// Only the reverse direction is published: EUR->USD at 1.25 means USD->EUR at 0.8.
	rates := []finance.ExchangeRate{testRate(finance.CurrencyEUR, finance.CurrencyUSD, 1.25)}
	converted, err := NewAmount(finance.CurrencyUSD, 100).Convert(finance.CurrencyEUR, rates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if converted.Value != 80 {
		t.Fatalf("got %v, want 80", converted.Value)
	}
}

func TestAmountConvertMissingRate(t *testing.T) {
	_, err := NewAmount(finance.CurrencyUSD, 100).Convert(finance.CurrencyEUR, nil)
	if !errors.Is(err, ErrRateNotAvailable) {
		t.Fatalf("got %v, want ErrRateNotAvailable", err)
	}
}

func TestFindExchangeRateIgnoresZeroRate(t *testing.T) {
	// A zero rate cannot be inverted and must not produce Inf.
	rates := []finance.ExchangeRate{testRate(finance.CurrencyEUR, finance.CurrencyUSD, 0)}
	_, err := FindExchangeRate(rates, finance.CurrencyUSD, finance.CurrencyEUR)
	if !errors.Is(err, ErrRateNotAvailable) {
		t.Fatalf("got %v, want ErrRateNotAvailable", err)
	}
}

func TestQuotesIsUniform(t *testing.T) {
	currency, uniform := Quotes{{Currency: finance.CurrencyEUR}, {Currency: finance.CurrencyEUR}}.IsUniform()
	if !uniform || currency != finance.CurrencyEUR {
		t.Fatalf("got %v %v, want EUR true", currency, uniform)
	}
	if _, uniform := (Quotes{{Currency: finance.CurrencyEUR}, {Currency: finance.CurrencyUSD}}).IsUniform(); uniform {
		t.Fatal("mixed currencies reported as uniform")
	}
	if _, uniform := (Quotes{}).IsUniform(); !uniform {
		t.Fatal("empty set reported as non-uniform")
	}
}
