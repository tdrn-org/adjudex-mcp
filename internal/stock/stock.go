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

type Runtime interface {
	DataStore() *data.Store
	Logger() *slog.Logger
}

type TrackerPool struct {
	runtime Runtime
	entries map[string]*trackerPoolEntry
	logger  *slog.Logger
	mutex   sync.RWMutex
}

func NewTrackerPool(runtime Runtime, cfg *config.QuoteTrackerConfig) (*TrackerPool, error) {
	tp := &TrackerPool{
		runtime: runtime,
		entries: make(map[string]*trackerPoolEntry),
		logger:  runtime.Logger().With(slog.String("job", "quotesTracker")),
	}
	if cfg.Demo.Enabled {
		tp.logger.Info("enabling demo quotes tracker")
		provider := demo.NewProvider(cfg.Currency)
		tp.addProvider(provider, cfg.Demo.Online, cfg.Demo.Frequency)
	}
	if cfg.AlphaVantage.Enabled {
		tp.logger.Info("enabling Alpha Vantage quotes tracker")
		provider, err := alphavantage.NewProvider(cfg.Currency, cfg.AlphaVantage.APIKey)
		if err != nil {
			return nil, err
		}
		tp.addProvider(provider, cfg.AlphaVantage.Online, cfg.AlphaVantage.Frequency)
	}
	if cfg.TwelveData.Enabled {
		tp.logger.Info("enabling Twelve Data quotes tracker")
		provider, err := twelvedata.NewProvider(cfg.Currency, cfg.TwelveData.APIKey)
		if err != nil {
			return nil, err
		}
		tp.addProvider(provider, cfg.TwelveData.Online, cfg.TwelveData.Frequency)
	}
	return tp, nil
}

type trackerPoolEntry struct {
	provider  tracker.NamedProvider
	online    bool
	frequency time.Duration
	lastRun   time.Time
	lastErr   error
}

func (tp *TrackerPool) addProvider(provider tracker.NamedProvider, online bool, frequency time.Duration) {
	tp.entries[provider.Name().String()] = &trackerPoolEntry{
		provider:  provider,
		online:    online,
		frequency: frequency,
	}
}

func (tp *TrackerPool) FetchQuote(ctx context.Context, symbol string) (*domain.Quote, error) {
	tp.mutex.RLock()
	defer tp.mutex.RUnlock()

	for service, entry := range tp.entries {
		if !entry.online {
			continue
		}
		quote, err := entry.provider.FetchQuote(ctx, symbol)
		entry.lastRun = time.Now()
		entry.lastErr = err
		if err != nil {
			tp.runtime.Logger().Warn("failed to fetch quote", slog.String("service", service), slog.String("symbol", symbol), slog.Any("err", err))
			continue
		}
		return quote, nil
	}
	return nil, domain.ErrNoQuote
}

func (tp *TrackerPool) FetchHistory(ctx context.Context, symbol string, from, to time.Time) ([]domain.Quote, error) {
	tp.mutex.RLock()
	defer tp.mutex.RUnlock()

	for service, entry := range tp.entries {
		if !entry.online {
			continue
		}
		quotes, err := entry.provider.FetchHistory(ctx, symbol, from, to)
		entry.lastRun = time.Now()
		entry.lastErr = err
		if err != nil {
			tp.runtime.Logger().Warn("failed to fetch history", slog.String("service", service), slog.String("symbol", symbol), slog.Any("err", err))
			continue
		}
		return quotes, nil
	}
	return nil, domain.ErrNoQuote
}

func (tp *TrackerPool) Run(ctx context.Context) {
	tp.logger.Info("fetching quotes...")

	tp.mutex.Lock()
	defer tp.mutex.Unlock()

	store := tp.runtime.DataStore()
	symbols, err := store.ListSymbols(ctx)
	if err != nil {
		tp.logger.Error("failed to list symbols", slog.Any("err", err))
	}
	for _, symbol := range symbols {
		quote, err := tp.FetchQuote(ctx, symbol)
		if err != nil {
			tp.logger.Warn("failed to fetch quote", slog.String("symbol", symbol), slog.Any("err", err))
			continue
		}
		err = store.SaveQuote(ctx, quote)
		if err != nil {
			tp.logger.Warn("failed to store quote", slog.String("symbol", symbol), slog.Any("err", err))
			continue
		}
	}
}

func (tp *TrackerPool) Close() error {
	tp.mutex.RLock()
	defer tp.mutex.RUnlock()

	closeErrs := make([]error, 0, len(tp.entries))
	for _, entry := range tp.entries {
		closeErr := entry.provider.Close()
		if closeErr != nil {
			closeErrs = append(closeErrs, closeErr)
		}
	}
	return errors.Join(closeErrs...)
}
