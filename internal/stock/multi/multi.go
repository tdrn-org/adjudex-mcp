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

// Package multi provides a provider that chains multiple backends with fallback.
package multi

import (
	"context"
	"fmt"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/adjudex-mcp/internal/stock"
)

// compile-time interface check
var _ stock.Provider = (*Provider)(nil)

// Provider chains multiple stock.Provider backends in fallback order.
type Provider struct {
	backends []stock.Provider
}

// New creates a multi-provider that tries each backend in order.
func New(backends ...stock.Provider) *Provider {
	return &Provider{backends: backends}
}

// FetchQuote tries each backend until one succeeds.
func (p *Provider) FetchQuote(ctx context.Context, symbol string) (*domain.Quote, error) {
	for i, b := range p.backends {
		q, err := b.FetchQuote(ctx, symbol)
		if err == nil {
			return q, nil
		}
		if i == len(p.backends)-1 {
			return nil, fmt.Errorf("multi: quote %q: all providers failed", symbol)
		}
	}
	return nil, fmt.Errorf("multi: quote %q: no backends configured", symbol)
}

// FetchHistory tries each backend until one succeeds.
func (p *Provider) FetchHistory(ctx context.Context, symbol string, from, to time.Time) ([]domain.Quote, error) {
	for i, b := range p.backends {
		quotes, err := b.FetchHistory(ctx, symbol, from, to)
		if err == nil {
			return quotes, nil
		}
		if i == len(p.backends)-1 {
			return nil, fmt.Errorf("multi: history %q: all providers failed", symbol)
		}
	}
	return nil, fmt.Errorf("multi: history %q: no backends configured", symbol)
}
