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

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-finance/config"
)

type Provider struct {
	fxProvider      finance.FX
	listingProvider finance.ListingProvider
	defaultCurrency finance.Currency
}

func NewProvider(cfg *config.Config, defaultCurrency finance.Currency) (*Provider, error) {
	fxProvider, err := cfg.NewFXProvider()
	if err != nil {
		return nil, err
	}
	listingProvider, err := cfg.NewListingProvider()
	if err != nil {
		return nil, err
	}
	provider := &Provider{
		fxProvider:      fxProvider,
		listingProvider: listingProvider,
		defaultCurrency: defaultCurrency,
	}
	return provider, nil
}

func (p *Provider) Close() error {
	return nil
}

func (p *Provider) SearchListings(ctx context.Context, query string) ([]finance.ProviderListing, error) {
	return p.listingProvider.SearchListings(ctx, query)
}

func (p *Provider) QueryQuote(ctx context.Context, listing finance.ProviderListing, currency finance.Currency) (*domain.Quote, error) {
	rawQuote, err := p.listingProvider.QueryQuote(ctx, listing)
	if err != nil {
		return nil, err
	}
	quote := &[]domain.Quote{domain.Quote(*rawQuote)}[0]
	quoteCurrency := currency
	if quoteCurrency == "" {
		quoteCurrency = p.defaultCurrency
	}
	if quote.Currency != currency {
		rate, err := p.fxProvider.QueryExchangeRate(ctx, quote.Currency, quoteCurrency)
		if err != nil {
			return nil, err
		}
		quote.ApplyCurrency(quoteCurrency, rate.Rate)
	}
	return quote, nil
}
