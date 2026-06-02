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

// Package quote provides background quote polling for adjudex.
package quote

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/adjudex-mcp/internal/stock"
)

// Poller periodically fetches quotes for all tracked symbols.
type Poller struct {
	provider    stock.Provider
	portfolio   domain.PortfolioStore
	quotes      domain.QuoteStore
	interval    time.Duration
	minInterval time.Duration
	lastPolled  map[string]time.Time
	mu          sync.Mutex
}

// NewPoller creates a new background quote poller.
func NewPoller(provider stock.Provider, portfolio domain.PortfolioStore, quotes domain.QuoteStore, interval, minInterval time.Duration) *Poller {
	return &Poller{
		provider:    provider,
		portfolio:   portfolio,
		quotes:      quotes,
		interval:    interval,
		minInterval: minInterval,
		lastPolled:  make(map[string]time.Time),
	}
}

// Run starts the polling loop. Blocks until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	log.Printf("quote/poller: started (interval=%v, minInterval=%v)", p.interval, p.minInterval)

	// Run once immediately
	p.poll(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Printf("quote/poller: stopped")
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

// poll collects symbols and fetches quotes respecting rate limits.
func (p *Poller) poll(ctx context.Context) {
	symbols, err := p.portfolio.ListSymbols(ctx)
	if err != nil {
		log.Printf("quote/poller: list symbols: %v", err)
		return
	}
	if len(symbols) == 0 {
		return
	}

	log.Printf("quote/poller: polling %d symbol(s)", len(symbols))
	for _, sym := range symbols {
		if !p.shouldPoll(sym) {
			continue
		}
		p.fetchAndSave(ctx, sym)
	}
}

// shouldPoll checks rate-limiting for a symbol.
func (p *Poller) shouldPoll(symbol string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if last, ok := p.lastPolled[symbol]; ok {
		return time.Since(last) >= p.minInterval
	}
	return true
}

// fetchAndSave fetches a quote and persists it, updating lastPolled.
func (p *Poller) fetchAndSave(ctx context.Context, symbol string) {
	q, err := p.provider.FetchQuote(ctx, symbol)
	if err != nil {
		log.Printf("quote/poller: fetch %q: %v", symbol, err)
		return
	}
	if err := p.quotes.SaveQuote(ctx, q); err != nil {
		log.Printf("quote/poller: save %q: %v", symbol, err)
		return
	}

	p.mu.Lock()
	p.lastPolled[symbol] = time.Now()
	p.mu.Unlock()

	log.Printf("quote/poller: %s = %.2f (source=%s)", symbol, q.Close, q.Source)
}

// PollNow forces an immediate poll for a specific symbol.
func (p *Poller) PollNow(ctx context.Context, symbol string) (*domain.Quote, error) {
	q, err := p.provider.FetchQuote(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("poller: poll %q: %w", symbol, err)
	}
	if err := p.quotes.SaveQuote(ctx, q); err != nil {
		return nil, fmt.Errorf("poller: save %q: %w", symbol, err)
	}
	p.mu.Lock()
	p.lastPolled[symbol] = time.Now()
	p.mu.Unlock()
	return q, nil
}
