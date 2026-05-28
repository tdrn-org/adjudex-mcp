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

// Package yahoo provides a quote provider that generates plausible mock data
// for development and testing. Real API integration will replace this once
// a stable API is chosen (expected: Phase 4 follow-up).
package yahoo

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/adjudex-mcp/internal/stock"
)

// compile-time interface check
var _ stock.Provider = (*Provider)(nil)

// Provider implements stock.Provider with synthetic historical data.
type Provider struct {
	// BasePrice is the reference price from which mock data is derived.
	BasePrice float64
}

// NewProvider creates a Provider with a default base price of 100.0.
func NewProvider() *Provider {
	return &Provider{BasePrice: 100.0}
}

// FetchQuote returns a single quote as of "now".
func (p *Provider) FetchQuote(ctx context.Context, symbol string) (*domain.Quote, error) {
	now := time.Now().Truncate(time.Minute)
	prices := p.generatePrices(now, now, 1)
	q := prices[0]
	q.Symbol = symbol
	q.Source = "mock-yahoo"
	return &q, nil
}

// FetchHistory returns quotes for the symbol from from to to (inclusive).
func (p *Provider) FetchHistory(ctx context.Context, symbol string, from, to time.Time) ([]domain.Quote, error) {
	if from.After(to) {
		return nil, fmt.Errorf("yahoo: FetchHistory for %q: from %s is after to %s", symbol, from, to)
	}
	prices := p.generatePrices(from, to, 0)
	for i := range prices {
		prices[i].Symbol = symbol
		prices[i].Source = "mock-yahoo"
	}
	return prices, nil
}

// generatePrices creates a synthetic price series between from and to.
// If count > 0, generates exactly count entries (for FetchQuote with count=1).
// If count == 0, generates daily entries from from to to.
func (p *Provider) generatePrices(from, to time.Time, count int) []domain.Quote {
	if count > 0 {
		return p.generateCount(count, from)
	}
	return p.generateDaily(from, to)
}

// generateDaily generates one quote per business day between from and to.
func (p *Provider) generateDaily(from, to time.Time) []domain.Quote {
	var quotes []domain.Quote
	// Start from a stable seed so results are deterministic per day
	rng := rand.New(rand.NewPCG(uint64(from.Unix()), uint64(from.UnixNano())))
	base := p.BasePrice
	current := from
	for !current.After(to) {
		if isBusinessDay(current) {
			// Random walk: drift + daily volatility
			drift := rng.NormFloat64() * base * 0.0001 // tiny drift
			vol := rng.NormFloat64() * base * 0.015    // ~1.5% daily vol
			change := drift + vol
			o := base
			c := o + change
			h := math.Max(o, c) + math.Abs(rng.Float64())*math.Abs(change)*0.5
			l := math.Min(o, c) - math.Abs(rng.Float64())*math.Abs(change)*0.5
			v := int64(rng.IntN(5_000_000)) + 500_000

			quotes = append(quotes, domain.Quote{
				Timestamp: current,
				Open:      round2(o),
				High:      round2(h),
				Low:       round2(l),
				Close:     round2(c),
				Volume:    v,
				Source:    "mock-yahoo",
			})
			base = c // next day starts from this close
		}
		current = current.AddDate(0, 0, 1)
	}
	return quotes
}

// generateCount generates exactly count quotes with timestamps evenly spaced.
func (p *Provider) generateCount(count int, anchor time.Time) []domain.Quote {
	rng := rand.New(rand.NewPCG(uint64(anchor.Unix()), uint64(anchor.UnixNano())))
	base := p.BasePrice
	quotes := make([]domain.Quote, count)
	for i := range count {
		drift := rng.NormFloat64() * base * 0.0001
		vol := rng.NormFloat64() * base * 0.015
		change := drift + vol
		o := base
		c := o + change
		h := math.Max(o, c) + math.Abs(rng.Float64())*math.Abs(change)*0.5
		l := math.Min(o, c) - math.Abs(rng.Float64())*math.Abs(change)*0.5
		v := int64(rng.IntN(5_000_000)) + 500_000

		quotes[i] = domain.Quote{
			Timestamp: anchor.Add(-time.Duration(count-1-i) * time.Minute * 5),
			Open:      round2(o),
			High:      round2(h),
			Low:       round2(l),
			Close:     round2(c),
			Volume:    v,
			Source:    "mock-yahoo",
		}
		base = c
	}
	return quotes
}

// isBusinessDay returns true if t is Monday–Friday.
func isBusinessDay(t time.Time) bool {
	d := t.Weekday()
	return d != time.Saturday && d != time.Sunday
}

// round2 rounds to two decimal places.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
