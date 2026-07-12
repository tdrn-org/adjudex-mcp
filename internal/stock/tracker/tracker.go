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

package tracker

import (
	"context"
	"io"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
)

const DefaultHistorySampleRate int = 12

type ProviderName string

func (name ProviderName) String() string {
	return string(name)
}

// Provider defines the interface for fetching stock quotes from an external data source.
// Implementations should handle rate limiting, error wrapping, and source attribution.
type Provider interface {
	// Name returns the name of provider.
	Name() ProviderName

	// ResolveSymbols tries to resolve a given query string to a symbol. If a nil or empty array
	// is returned, no symbol info could be resolved,
	ResolveSymbols(ctx context.Context, query string) ([]domain.SymbolInfo, error)

	// FetchQuote returns the most recent quote for a symbol.
	FetchQuote(ctx context.Context, symbol string) (*domain.Quote, error)

	// FetchHistory returns quotes for a symbol within the given date range.
	FetchHistory(ctx context.Context, symbol string, from, to time.Time) (domain.Quotes, error)

	io.Closer
}
