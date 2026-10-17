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

package marketdata

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/config"
)

// Rate lookup retry policy.
//
// A streaming provider may report finance.ErrRequestPending while its
// subscription is still warming up. When every configured FX provider is in that
// state, the fallback composite in go-finance collapses the individual
// ErrRequestPending errors into finance.ErrExchangeRateNotAvailable, so from
// here a warming provider and a genuinely unsupported currency look identical.
// Retrying a few times tells the two apart without turning an unsupported
// currency into a long stall.
const (
	rateLookupAttempts = 3
	rateLookupDelay    = 400 * time.Millisecond
)

// Provider implements [domain.MarketData] on top of go-finance.
type Provider struct {
	fxProvider      finance.FX
	listingProvider finance.ListingProvider
}

// NewProvider creates the market data provider from the given go-finance
// configuration.
//
// There is deliberately no default currency here: a quote is always returned in
// the currency its provider reports, and a caller that wants another currency
// asks for the exchange rate explicitly.
func NewProvider(cfg *config.Config) (*Provider, error) {
	fxProvider, err := cfg.NewFXProvider()
	if err != nil {
		return nil, err
	}
	listingProvider, err := cfg.NewListingProvider()
	if err != nil {
		return nil, err
	}
	return &Provider{
		fxProvider:      fxProvider,
		listingProvider: listingProvider,
	}, nil
}

func (p *Provider) Close() error {
	return nil
}

func (p *Provider) SearchListings(ctx context.Context, query string) ([]finance.Listing, error) {
	return p.listingProvider.SearchListings(ctx, query)
}

// QueryQuote returns the quote as reported by the provider, without any currency
// conversion. The quote carries its currency and its provenance; rescaling it
// here would invalidate both.
func (p *Provider) QueryQuote(ctx context.Context, listing finance.Listing) (*domain.Quote, error) {
	rawQuote, err := p.listingProvider.QueryQuote(ctx, listing)
	if err != nil {
		return nil, err
	}
	quote := domain.Quote(*rawQuote)
	return &quote, nil
}

// QueryExchangeRate returns the rate converting base into quote.
func (p *Provider) QueryExchangeRate(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	if base == quote {
		return identityRate(base), nil
	}
	return p.queryExchangeRateWithRetry(ctx, base, quote)
}

// QueryExchangeRates returns the rates converting each of the given base
// currencies into target. Bases already denominated in target are skipped, so a
// single-currency portfolio never triggers an FX lookup at all.
func (p *Provider) QueryExchangeRates(ctx context.Context, target finance.Currency, bases []finance.Currency) ([]finance.ExchangeRate, error) {
	rates := make([]finance.ExchangeRate, 0, len(bases))
	seen := make(map[finance.Currency]bool, len(bases))
	for _, base := range bases {
		if base == target || seen[base] {
			continue
		}
		seen[base] = true
		rate, err := p.queryExchangeRateWithRetry(ctx, base, target)
		if err != nil {
			return nil, fmt.Errorf("querying rate %s->%s: %w", base, target, err)
		}
		rates = append(rates, *rate)
	}
	return rates, nil
}

func (p *Provider) queryExchangeRateWithRetry(ctx context.Context, base, quote finance.Currency) (*finance.ExchangeRate, error) {
	var lastErr error
	for attempt := 0; attempt < rateLookupAttempts; attempt++ {
		rate, err := p.fxProvider.QueryExchangeRate(ctx, base, quote)
		if err == nil {
			return rate, nil
		}
		lastErr = err
		if !isTransientRateError(err) {
			return nil, err
		}
		if attempt < rateLookupAttempts-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(rateLookupDelay):
			}
		}
	}
	return nil, lastErr
}

// isTransientRateError reports whether a rate lookup may succeed on a retry: the
// provider stream is still warming up, or every provider was pending and the
// fallback composite collapsed that into "not available".
func isTransientRateError(err error) bool {
	return errors.Is(err, finance.ErrRequestPending) || errors.Is(err, finance.ErrExchangeRateNotAvailable)
}

// identityRate is the rate of a currency to itself.
func identityRate(currency finance.Currency) *finance.ExchangeRate {
	now := time.Now().UTC()
	return &finance.ExchangeRate{
		Timestamp:       now,
		Base:            currency,
		Quote:           currency,
		Rate:            1,
		Source:          "identity",
		SourceTimestamp: now,
	}
}
