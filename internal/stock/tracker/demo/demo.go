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

package demo

import (
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/adjudex-mcp/internal/stock/tracker"
)

const Name tracker.ProviderName = "demo"

func NewProvider(currency string) tracker.NamedProvider {
	return &demoProvider{
		currency: currency,
	}
}

type demoProvider struct {
	currency string
}

func (p *demoProvider) Name() tracker.ProviderName {
	return Name
}

func (p *demoProvider) FetchQuote(_ context.Context, symbol string) (*domain.Quote, error) {
	quote := p.generateQuote(symbol, time.Now())
	return quote, nil
}

func (p *demoProvider) FetchHistory(_ context.Context, symbol string, from, to time.Time) (domain.Quotes, error) {
	if !from.Before(to) {
		return []domain.Quote{}, fmt.Errorf("invalid time range (from: %s to: %s)", from, to)
	}
	quotes := make([]domain.Quote, 0, tracker.DefaultHistorySampleRate)
	for current := from; !current.After(to); current = current.Add(24 * time.Hour) {
		quote := p.generateQuote(symbol, current)
		quotes = append(quotes, *quote)
	}
	return quotes, nil
}

func (p *demoProvider) Close() error {
	return nil
}

func (p *demoProvider) generateQuote(symbol string, timestamp time.Time) *domain.Quote {
	hash := fnv.New32a()
	hash.Write([]byte(symbol))
	base := float64((1 + (hash.Sum32() % 100)) * 10)
	return &domain.Quote{
		Symbol:          symbol,
		Timestamp:       timestamp,
		Currency:        p.currency,
		Open:            base,
		High:            base * 1.1,
		Low:             base * 0.9,
		Close:           base,
		Price:           base,
		Volume:          42,
		Source:          string(p.Name()),
		SourceTimestamp: timestamp,
	}
}
