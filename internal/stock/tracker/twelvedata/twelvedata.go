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

package twelvedata

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/buildinfo"
	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/adjudex-mcp/internal/stock/tracker"
	"github.com/tdrn-org/go-cache"
	"github.com/tdrn-org/go-cache/memory"
	"github.com/twelvedata/twelvedata-go/twelvedata"
)

const Name tracker.ProviderName = "twelvedata"

func NewProvider(currency string, apiKey string) (tracker.NamedProvider, error) {
	cfg := twelvedata.NewConfiguration()
	cfg.DefaultHeader["Authorization"] = "apikey " + apiKey
	cfg.DefaultHeader["X-API-Version"] = "last"
	cfg.HTTPClient = &http.Client{Transport: twelvedata.WrapTransportWithSource(http.DefaultTransport, buildinfo.Cmd())}
	client := twelvedata.NewAPIClient(cfg)
	p := &twelvedataProvider{
		currency: currency,
		client:   client,
		logger:   slog.With(slog.String("provider", Name.String())),
	}
	exchangeRateCache, err := memory.NewKeyValue(0, -24*time.Hour, p.loadExchangeRate)
	if err != nil {
		return nil, err
	}
	p.exchangeRateCache = exchangeRateCache
	return p, nil
}

type twelvedataProvider struct {
	currency          string
	client            *twelvedata.APIClient
	exchangeRateCache cache.KeyValue[string, float64]
	logger            *slog.Logger
}

func (p *twelvedataProvider) Name() tracker.ProviderName {
	return Name
}

func (p *twelvedataProvider) FetchQuote(ctx context.Context, symbol string) (*domain.Quote, error) {
	timestamp := time.Now()
	rsp, _, err := p.client.MarketDataAPI.
		GetQuote(ctx).
		Symbol(symbol).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quote data via Twelve Data (cause: %w)", err)
	}
	quote, err := p.quoteFromGetQuoteResponse(symbol, rsp, timestamp)
	if err != nil {
		return nil, err
	}
	quote.ApplyCurrency(p.currencyExchangeRate(ctx, quote.Currency))
	return quote, nil
}

func (p *twelvedataProvider) FetchHistory(ctx context.Context, symbol string, from, to time.Time) (domain.Quotes, error) {
	timestamp := time.Now()
	rsp, _, err := p.client.MarketDataAPI.
		GetTimeSeries(ctx).
		Symbol(symbol).
		StartDate(from.Format(shortDateTimeLayout)).
		EndDate(to.Format((shortDateTimeLayout))).
		Interval(twelvedata.INTERVALENUM__1DAY).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch history data via Twelve Data (cause: %w)", err)
	}
	quotes, err := p.quotesFromGetTimeSeriesResponse(symbol, rsp, timestamp)
	if err != nil {
		return nil, err
	}
	quotes.ApplyCurrency(p.currencyExchangeRate(ctx, quotes.Currency()))
	return quotes, nil
}

func (p *twelvedataProvider) Close() error {
	return p.exchangeRateCache.Close()
}

func (p *twelvedataProvider) quoteFromGetQuoteResponse(symbol string, rsp *twelvedata.GetQuote200Response, timestamp time.Time) (*domain.Quote, error) {
	if rsp.Currency == nil || *rsp.Currency == "" {
		return nil, fmt.Errorf("missing currency in quote for symbol '%s'", symbol)
	}
	open, err := p.parseSymbolCurrency(symbol, "open", rsp.Open)
	if err != nil {
		return nil, err
	}
	high, err := p.parseSymbolCurrency(symbol, "high", rsp.High)
	if err != nil {
		return nil, err
	}
	low, err := p.parseSymbolCurrency(symbol, "low", rsp.Low)
	if err != nil {
		return nil, err
	}
	close, err := p.parseSymbolCurrency(symbol, "close", rsp.Close)
	if err != nil {
		return nil, err
	}
	var volume int64
	if rsp.Volume != nil {
		volume, err = p.parseSymbolAmount(symbol, "volume", *rsp.Volume)
		if err != nil {
			return nil, err
		}
	}
	sourceTimestamp, err := p.parseSymbolTimestamp(symbol, "datetime", rsp.Datetime)
	if err != nil {
		return nil, err
	}
	quote := &domain.Quote{
		Symbol:          symbol,
		Timestamp:       timestamp,
		Currency:        *rsp.Currency,
		Open:            open,
		High:            high,
		Low:             low,
		Close:           close,
		Price:           close,
		Volume:          volume,
		Source:          Name.String(),
		SourceTimestamp: sourceTimestamp,
	}
	return quote, nil
}

func (p *twelvedataProvider) quotesFromGetTimeSeriesResponse(symbol string, rsp *twelvedata.GetTimeSeries200Response, timestamp time.Time) (domain.Quotes, error) {
	if rsp.Meta.Currency == nil || *rsp.Meta.Currency == "" {
		return nil, fmt.Errorf("missing currency in time series for symbol '%s'", symbol)
	}
	values := rsp.GetValues()
	quotes := make(domain.Quotes, 0, len(values))
	for _, value := range values {
		quote, err := p.quoteFromTimeSeriesItem(symbol, *rsp.Meta.Currency, &value, timestamp)
		if err != nil {
			return nil, err
		}
		quotes = append(quotes, *quote)
	}
	return quotes, nil
}

func (p *twelvedataProvider) quoteFromTimeSeriesItem(symbol, currency string, item *twelvedata.TimeSeriesItem, timestamp time.Time) (*domain.Quote, error) {
	open, err := p.parseSymbolCurrency(symbol, "open", item.Open)
	if err != nil {
		return nil, err
	}
	high, err := p.parseSymbolCurrency(symbol, "high", item.High)
	if err != nil {
		return nil, err
	}
	low, err := p.parseSymbolCurrency(symbol, "low", item.Low)
	if err != nil {
		return nil, err
	}
	close, err := p.parseSymbolCurrency(symbol, "close", item.Close)
	if err != nil {
		return nil, err
	}
	var volume int64
	if item.Volume != nil {
		volume, err = p.parseSymbolAmount(symbol, "volume", *item.Volume)
		if err != nil {
			return nil, err
		}
	}
	sourceTimestamp, err := p.parseSymbolTimestamp(symbol, "datetime", item.Datetime)
	if err != nil {
		return nil, err
	}
	quote := &domain.Quote{
		Symbol:          symbol,
		Timestamp:       timestamp,
		Currency:        currency,
		Open:            open,
		High:            high,
		Low:             low,
		Close:           close,
		Price:           close,
		Volume:          volume,
		Source:          Name.String(),
		SourceTimestamp: sourceTimestamp,
	}
	return quote, nil
}

func (p *twelvedataProvider) currencyExchangeRate(ctx context.Context, currency string) (string, float64) {
	if p.currency == currency {
		return currency, 1.0
	}
	symbol := fmt.Sprintf("%s/%s", currency, p.currency)
	exchangeRate, err := p.exchangeRateCache.Get(ctx, symbol)
	if err != nil {
		p.logger.Warn("failed to get currency exchange rate", slog.String("symbol", symbol), slog.Any("err", err))
		return currency, 1.0
	}
	return p.currency, exchangeRate
}

func (p *twelvedataProvider) loadExchangeRate(ctx context.Context, symbol string) (float64, error) {
	split := strings.IndexRune(symbol, '/')
	if split <= 0 {
		return math.NaN(), fmt.Errorf("invalid symbol '%s'", symbol)
	}
	fromCurrency := symbol[:split-1]
	toCurrency := symbol[split:]
	if fromCurrency == toCurrency {
		return 1.0, nil
	}
	conversion, _, err := p.client.CurrenciesAPI.
		GetExchangeRate(ctx).
		Symbol(symbol).
		Execute()
	if err != nil {
		return math.NaN(), err
	}
	return conversion.Rate, nil
}
