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
	"maps"
	"slices"
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

type Runtime interface {
	DataStore() *data.Store
	Logger() *slog.Logger
}

// QuoteService resolves stock quotes using store-first caching, provider affinity,
// and fallback across all configured sources. It also runs a periodic job
// to keep quotes fresh for all tracked symbols.
type QuoteService struct {
	cfg            *config.QuoteServiceConfig
	runtime        Runtime
	sources        map[tracker.ProviderName]*sourceProvider
	sourceAffinity map[string]tracker.ProviderName // symbol → preferred provider
	logger         *slog.Logger
	mutex          sync.Mutex
}

func NewQuoteService(runtime Runtime, cfg *config.QuoteServiceConfig) (*QuoteService, error) {
	qs := &QuoteService{
		cfg:            cfg,
		runtime:        runtime,
		sources:        make(map[tracker.ProviderName]*sourceProvider),
		sourceAffinity: make(map[string]tracker.ProviderName),
		logger:         runtime.Logger().With(slog.String("job", "quotesTracker")),
	}
	if cfg.Demo.Enabled {
		qs.logger.Info("enabling Demo quotes tracker")
		provider := demo.NewProvider(cfg.Currency)
		qs.addSource(provider, cfg.Demo.Online)
	}
	if cfg.AlphaVantage.Enabled {
		qs.logger.Info("enabling Alpha Vantage quotes tracker")
		provider, err := alphavantage.NewProvider(cfg.Currency, cfg.AlphaVantage.APIKey)
		if err != nil {
			return nil, err
		}
		qs.addSource(provider, cfg.AlphaVantage.Online)
	}
	if cfg.TwelveData.Enabled {
		qs.logger.Info("enabling Twelve Data quotes tracker")
		provider, err := twelvedata.NewProvider(cfg.Currency, cfg.TwelveData.APIKey)
		if err != nil {
			return nil, err
		}
		qs.addSource(provider, cfg.TwelveData.Online)
	}
	return qs, nil
}

type sourceProvider struct {
	provider tracker.Provider
	online   bool
}

func (qs *QuoteService) addSource(provider tracker.Provider, online bool) {
	qs.sources[provider.Name()] = &sourceProvider{
		provider: provider,
		online:   online,
	}
}

// ResolveSymbols resolves a search query (e.g., WKN) to a ticker symbol
// using all configured providers. Returns the symbol and the provider that resolved it.
func (qs *QuoteService) ResolveSymbols(ctx context.Context, query string) []domain.SymbolInfo {
	var result []domain.SymbolInfo
	for providerName, source := range qs.sources {
		symbolInfos, err := source.provider.ResolveSymbols(ctx, query)
		if err != nil {
			qs.logger.Warn("failed to resolve symbol", slog.String("provider", providerName.String()), slog.Any("err", err))
			continue
		}
		result = append(result, symbolInfos...)
	}
	return result
}

// ResolveQuote returns the latest quote for a symbol using store-first caching
// with a maxAge freshness check, provider affinity, and fallback across all providers.
func (qs *QuoteService) ResolveQuote(ctx context.Context, symbol string) (*domain.Quote, error) {
	store := qs.runtime.DataStore()
	quote, err := store.GetLatestQuote(ctx, symbol)
	if err != nil {
		qs.logger.Warn("failed to get latest quote", slog.String("symbol", symbol), slog.Any("err", err))
	}
	if quote != nil && time.Since(quote.Timestamp) < time.Duration(qs.cfg.MaxAge) {
		return quote, nil // ✅ cache hit, 0 API calls
	}

	qs.mutex.Lock()
	defer qs.mutex.Unlock()

	quote, err = qs.fetchQuoteLocked(ctx, symbol, qs.sourceAffinity[symbol])
	if err != nil {
		return nil, err
	}
	err = store.SaveQuote(ctx, quote)
	if err != nil {
		qs.logger.Warn("failed to save quote", slog.String("symbol", symbol), slog.Any("err", err))
	}
	qs.sourceAffinity[symbol] = tracker.ProviderName(quote.Source)
	return quote, nil
}

func (qs *QuoteService) fetchQuoteLocked(ctx context.Context, symbol string, preferred tracker.ProviderName) (*domain.Quote, error) {
	for providerName, source := range qs.sources {
		if providerName != preferred || !source.online {
			continue
		}
		quote, err := source.provider.FetchQuote(ctx, symbol)
		if err == nil {
			return quote, nil
		}
		qs.logger.Warn("failed to fetch quote", slog.String("source", providerName.String()), slog.String("symbol", symbol), slog.Any("err", err))
		break
	}
	for providerName, source := range qs.sources {
		if providerName == preferred || !source.online {
			continue
		}
		quote, err := source.provider.FetchQuote(ctx, symbol)
		if err == nil {
			return quote, nil
		}
		qs.logger.Warn("failed to fetch quote", slog.String("source", providerName.String()), slog.String("symbol", symbol), slog.Any("err", err))
	}
	return nil, domain.ErrNoQuote
}

// FetchHistory returns historical quotes. Store-first, falls back to live providers.
func (qs *QuoteService) FetchHistory(ctx context.Context, symbol string, from, to time.Time) (domain.Quotes, error) {
	quotes, err := qs.fetchSavedHistory(ctx, symbol, from, to)
	if err != nil {
		qs.logger.Warn("failed to get quote history", slog.String("symbol", symbol), slog.Any("err", err))
	}
	if len(quotes) >= (int(to.Sub(from).Hours())+23)/24 {
		return quotes, nil
	}

	qs.mutex.Lock()
	defer qs.mutex.Unlock()

	quotes, err = qs.fetchHistoryLocked(ctx, symbol, from, to, qs.sourceAffinity[symbol])
	if err != nil {
		return nil, err
	}
	err = qs.runtime.DataStore().SaveQuotes(ctx, quotes)
	if err != nil {
		qs.logger.Warn("failed to save quotes", slog.String("symbol", symbol), slog.Any("err", err))
	}
	if len(quotes) > 0 {
		qs.sourceAffinity[symbol] = tracker.ProviderName(quotes[0].Source)
	}
	return quotes, nil
}

func (qs *QuoteService) fetchSavedHistory(ctx context.Context, symbol string, from, to time.Time) (domain.Quotes, error) {
	quotes, err := qs.runtime.DataStore().GetQuotes(ctx, symbol, from, to)
	if err != nil {
		return nil, err
	}
	dateQuotes := make(map[time.Time]int, len(quotes))
	for quoteIndex, quote := range quotes {
		date := time.Date(quote.SourceTimestamp.Year(), quote.SourceTimestamp.Month(), quote.SourceTimestamp.Day(), 0, 0, 0, 0, quote.SourceTimestamp.Location())
		dateQuoteIndex, ok := dateQuotes[date]
		if !ok || quotes[dateQuoteIndex].SourceTimestamp.Before(quote.SourceTimestamp) {
			dateQuotes[date] = quoteIndex
		}
	}
	history := make(domain.Quotes, 0, len(dateQuotes))
	for _, date := range slices.SortedFunc(maps.Keys(dateQuotes), func(t1, t2 time.Time) int { return t1.Compare(t2) }) {
		history = append(history, quotes[dateQuotes[date]])
	}
	return history, nil
}

func (qs *QuoteService) fetchHistoryLocked(ctx context.Context, symbol string, from, to time.Time, preferred tracker.ProviderName) (domain.Quotes, error) {
	for providerName, source := range qs.sources {
		if providerName != preferred || !source.online {
			continue
		}
		quotes, err := source.provider.FetchHistory(ctx, symbol, from, to)
		if err == nil {
			return quotes, nil
		}
		qs.logger.Warn("failed to fetch quotes", slog.String("source", providerName.String()), slog.String("symbol", symbol), slog.Any("err", err))
		break
	}
	for providerName, source := range qs.sources {
		if providerName == preferred || !source.online {
			continue
		}
		quotes, err := source.provider.FetchHistory(ctx, symbol, from, to)
		if err == nil {
			return quotes, nil
		}
		qs.logger.Warn("failed to fetch quotes", slog.String("source", providerName.String()), slog.String("symbol", symbol), slog.Any("err", err))
	}
	return nil, domain.ErrNoQuote
}

func (qs *QuoteService) Run(ctx context.Context) {
	qs.logger.Info("fetching quotes...")

	qs.mutex.Lock()
	defer qs.mutex.Unlock()

	store := qs.runtime.DataStore()
	symbolMap, err := store.ListSymbols(ctx)
	if err != nil {
		qs.logger.Error("failed to list symbols", slog.Any("err", err))
	}
	for symbol, lastQuote := range symbolMap {
		if time.Since(lastQuote) < time.Duration(qs.cfg.MaxAge) {
			continue
		}
		qs.logger.Debug("fetching quote...", slog.String("symbol", symbol))
		quote, err := qs.fetchQuoteLocked(ctx, symbol, "")
		if err != nil {
			// fetchQuoteLocked logs failures already
			continue
		}
		err = store.SaveQuote(ctx, quote)
		if err != nil {
			qs.logger.Warn("failed to save quote", slog.String("symbol", symbol), slog.Any("err", err))
		}
	}
}

func (qs *QuoteService) Close() error {
	qs.mutex.Lock()
	defer qs.mutex.Unlock()

	closeErrs := make([]error, 0, len(qs.sources))
	for _, source := range qs.sources {
		closeErr := source.provider.Close()
		if closeErr != nil {
			closeErrs = append(closeErrs, closeErr)
		}
	}
	return errors.Join(closeErrs...)
}
