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

package stock

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/tdrn-org/adjudex-mcp/config"
	"github.com/tdrn-org/adjudex-mcp/internal/data"
	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/adjudex-mcp/internal/stock/tracker"
	"github.com/tdrn-org/adjudex-mcp/internal/stock/tracker/alphavantage"
	"github.com/tdrn-org/adjudex-mcp/internal/stock/tracker/demo"
	"github.com/tdrn-org/adjudex-mcp/internal/stock/tracker/twelvedata"
)

const defaultMaxAge time.Duration = 15 * time.Minute

type Runtime interface {
	DataStore() *data.Store
	Logger() *slog.Logger
}

// QuoteService resolves stock quotes using store-first caching, provider affinity,
// and fallback across all configured providers. It also runs a periodic job
// to keep quotes fresh for all tracked symbols.
type QuoteService struct {
	runtime  Runtime
	entries  map[string]*quoteServiceEntry
	affinity map[string]tracker.ProviderName // symbol → preferred provider
	maxAge   time.Duration                   // max age for cached quotes
	logger   *slog.Logger
	mutex    sync.RWMutex
}

func NewQuoteService(runtime Runtime, cfg *config.QuoteTrackerConfig) (*QuoteService, error) {
	qs := &QuoteService{
		runtime:  runtime,
		entries:  make(map[string]*quoteServiceEntry),
		affinity: make(map[string]tracker.ProviderName),
		maxAge:   defaultMaxAge,
		logger:   runtime.Logger().With(slog.String("job", "quotesTracker")),
	}
	if cfg.Demo.Enabled {
		qs.logger.Info("enabling demo quotes tracker")
		provider := demo.NewProvider(cfg.Currency)
		qs.addProvider(provider, cfg.Demo.Online, cfg.Demo.Frequency)
	}
	if cfg.AlphaVantage.Enabled {
		qs.logger.Info("enabling Alpha Vantage quotes tracker")
		provider, err := alphavantage.NewProvider(cfg.Currency, cfg.AlphaVantage.APIKey)
		if err != nil {
			return nil, err
		}
		qs.addProvider(provider, cfg.AlphaVantage.Online, cfg.AlphaVantage.Frequency)
	}
	if cfg.TwelveData.Enabled {
		qs.logger.Info("enabling Twelve Data quotes tracker")
		provider, err := twelvedata.NewProvider(cfg.Currency, cfg.TwelveData.APIKey)
		if err != nil {
			return nil, err
		}
		qs.addProvider(provider, cfg.TwelveData.Online, cfg.TwelveData.Frequency)
	}
	return qs, nil
}

type quoteServiceEntry struct {
	provider  tracker.NamedProvider
	online    bool
	frequency time.Duration
	lastRun   time.Time
	lastErr   error
}

func (qs *QuoteService) addProvider(provider tracker.NamedProvider, online bool, frequency time.Duration) {
	qs.entries[provider.Name().String()] = &quoteServiceEntry{
		provider:  provider,
		online:    online,
		frequency: frequency,
	}
}

// ResolveQuote returns the latest quote for a symbol using store-first caching
// with a maxAge freshness check, provider affinity, and fallback across all providers.
func (qs *QuoteService) ResolveQuote(ctx context.Context, symbol string) (*domain.Quote, error) {
	qs.mutex.RLock()
	maxAge := qs.maxAge
	preferred, hasAffinity := qs.affinity[symbol]
	qs.mutex.RUnlock()

	// 1. Store-First with freshness check
	q, _ := qs.runtime.DataStore().GetLatestQuote(ctx, symbol)
	if q != nil && time.Since(q.Timestamp) < maxAge {
		return q, nil // ✅ cache hit, 0 API calls
	}

	// 2. Provider affinity: try the last successful provider first
	if hasAffinity {
		qs.logger.Info("resolving quote via affinity", "symbol", symbol, "provider", preferred.String())
		q, err := qs.fetchAndSaveQuoteFrom(ctx, symbol, preferred)
		if err == nil {
			return q, nil
		}
		qs.logger.Warn("affinity provider failed, falling back", "symbol", symbol, "provider", preferred.String(), "err", err)
	}

	// 3. Fallback: try all providers
	qs.logger.Info("resolving quote via fallback", "symbol", symbol)
	q, err := qs.fetchAndSaveQuote(ctx, symbol)
	if err != nil {
		return nil, err
	}

	// 4. Update provider affinity
	qs.mutex.Lock()
	qs.affinity[symbol] = tracker.ProviderName(q.Source)
	qs.mutex.Unlock()

	return q, nil
}

// ResolveSymbol resolves a search query (e.g., WKN) to a ticker symbol
// using all configured providers. Returns the symbol and the provider that resolved it.
func (qs *QuoteService) ResolveSymbol(ctx context.Context, query string) (symbol string, provider tracker.ProviderName, err error) {
	// TODO: implement WKN→ticker resolution via Alpha Vantage SYMBOL_SEARCH
	return "", "", errors.New("not implemented")
}

// fetchAndSaveQuote fetches a quote from all online providers and saves it to the store.
func (qs *QuoteService) fetchAndSaveQuote(ctx context.Context, symbol string) (*domain.Quote, error) {
	qs.mutex.RLock()
	defer qs.mutex.RUnlock()

	return qs.fetchAndSaveQuoteLocked(ctx, symbol)
}

func (qs *QuoteService) fetchAndSaveQuoteFrom(ctx context.Context, symbol string, providerName tracker.ProviderName) (*domain.Quote, error) {
	qs.mutex.RLock()
	defer qs.mutex.RUnlock()

	entry, ok := qs.entries[providerName.String()]
	if !ok || !entry.online {
		return nil, errors.New("provider not available")
	}
	return qs.fetchAndSaveQuoteFromEntry(ctx, symbol, providerName, entry)
}

func (qs *QuoteService) fetchAndSaveQuoteLocked(ctx context.Context, symbol string) (*domain.Quote, error) {
	for service, entry := range qs.entries {
		if !entry.online {
			continue
		}
		quote, err := qs.fetchAndSaveQuoteFromEntry(ctx, symbol, tracker.ProviderName(service), entry)
		if err != nil {
			qs.logger.Warn("failed to fetch quote", slog.String("service", service), slog.String("symbol", symbol), slog.Any("err", err))
			continue
		}
		return quote, nil
	}
	return nil, domain.ErrNoQuote
}

func (qs *QuoteService) fetchAndSaveQuoteFromEntry(ctx context.Context, symbol string, providerName tracker.ProviderName, entry *quoteServiceEntry) (*domain.Quote, error) {
	qs.logger.Info("fetching quote...", slog.String("symbol", symbol), slog.String("provider", providerName.String()))
	quote, err := entry.provider.FetchQuote(ctx, symbol)
	entry.lastRun = time.Now()
	entry.lastErr = err
	if err != nil {
		return nil, err
	}
	qs.logger.Debug("quote fetched", slog.String("symbol", symbol), slog.Float64("price", quote.Price), slog.String("currency", quote.Currency))
	err = qs.runtime.DataStore().SaveQuote(ctx, quote)
	if err != nil {
		qs.runtime.Logger().Warn("failed to save quote", slog.String("symbol", symbol), slog.Any("err", err))
	}
	return quote, nil
}

// FetchQuote is the legacy API for direct provider access (used by MCP Runtime compatibility).
// Prefer ResolveQuote for new code.
func (qs *QuoteService) FetchQuote(ctx context.Context, symbol string) (*domain.Quote, error) {
	return qs.ResolveQuote(ctx, symbol)
}

// FetchHistory returns historical quotes. Store-first, falls back to live providers.
func (qs *QuoteService) FetchHistory(ctx context.Context, symbol string, from, to time.Time) (domain.Quotes, error) {
	return qs.fetchAndSaveHistory(ctx, symbol, from, to)
}

func (qs *QuoteService) fetchAndSaveHistory(ctx context.Context, symbol string, from, to time.Time) (domain.Quotes, error) {
	for service, entry := range qs.entries {
		if !entry.online {
			continue
		}
		quotes, err := entry.provider.FetchHistory(ctx, symbol, from, to)
		entry.lastRun = time.Now()
		entry.lastErr = err
		if err != nil {
			qs.runtime.Logger().Warn("failed to fetch history", slog.String("service", service), slog.String("symbol", symbol), slog.Any("err", err))
			continue
		}
		err = qs.runtime.DataStore().SaveQuotes(ctx, quotes)
		if err != nil {
			qs.runtime.Logger().Warn("failed to save quotes", slog.String("symbol", symbol), slog.Any("err", err))
		}
		return quotes, nil
	}
	return nil, domain.ErrNoQuote
}

func (qs *QuoteService) Run(ctx context.Context) {
	qs.logger.Info("fetching quotes...")

	qs.mutex.Lock()
	defer qs.mutex.Unlock()

	store := qs.runtime.DataStore()
	symbols, err := store.ListSymbols(ctx)
	if err != nil {
		qs.logger.Error("failed to list symbols", slog.Any("err", err))
	}
	for _, symbol := range symbols {
		qs.logger.Debug("fetching quote...", slog.String("symbol", symbol))
		_, err := qs.fetchAndSaveQuoteLocked(ctx, symbol)
		if err != nil {
			qs.logger.Warn("failed to fetch quote", slog.String("symbol", symbol), slog.Any("err", err))
			continue
		}
	}
}

func (qs *QuoteService) Close() error {
	qs.mutex.RLock()
	defer qs.mutex.RUnlock()

	closeErrs := make([]error, 0, len(qs.entries))
	for _, entry := range qs.entries {
		closeErr := entry.provider.Close()
		if closeErr != nil {
			closeErrs = append(closeErrs, closeErr)
		}
	}
	return errors.Join(closeErrs...)
}
