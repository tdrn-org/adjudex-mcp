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

package alphavantage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/adjudex-mcp/internal/stock/tracker"
	"github.com/tdrn-org/go-cache"
	"github.com/tdrn-org/go-cache/memory"
)

const Name tracker.ProviderName = "alphavantage"

const defaultBaseURL string = "https://www.alphavantage.co/query"

func NewProvider(currency, apiKey string) (tracker.Provider, error) {
	p := &alphavantageProvider{
		currency:   currency,
		baseURL:    defaultBaseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{},
		logger:     slog.With(slog.String("provider", Name.String())),
	}
	symbolInfoCache, err := memory.NewKeyValue(0, -24*time.Hour, p.loadSymbolInfo)
	if err != nil {
		return nil, err
	}
	p.symbolInfoCache = symbolInfoCache
	return p, nil
}

type alphavantageProvider struct {
	currency        string
	baseURL         string
	apiKey          string
	httpClient      *http.Client
	symbolInfoCache cache.KeyValue[string, *symbolInfo]
	logger          *slog.Logger
}

type symbolInfo struct {
	SourceCurrency string
	ExchangeRate   float64
}

func (p *alphavantageProvider) Name() tracker.ProviderName {
	return Name
}

func (p *alphavantageProvider) ResolveSymbols(ctx context.Context, query string) ([]domain.SymbolInfo, error) {
	symbolSearch, err := p.getSymbolSearch(ctx, query)
	if err != nil {
		return nil, err
	}
	symbolInfos := make([]domain.SymbolInfo, 0, len(symbolSearch.BestMatches))
	for _, bestMatch := range symbolSearch.BestMatches {
		symbolInfo := domain.SymbolInfo{
			Symbol: bestMatch.Symbol,
			Source: Name.String(),
			Info:   fmt.Sprintf("%s/%s", bestMatch.Region, bestMatch.Currency),
		}
		symbolInfos = append(symbolInfos, symbolInfo)
	}
	return symbolInfos, nil
}

func (p *alphavantageProvider) FetchQuote(ctx context.Context, symbol string) (*domain.Quote, error) {
	timestamp := time.Now()
	info, err := p.symbolInfo(ctx, symbol)
	if err != nil {
		return nil, err
	}
	globalQuote, err := p.getGlobalQuote(ctx, symbol)
	if err != nil {
		return nil, nil
	}
	quote, err := p.quoteFromGlobalQuoteResponse(globalQuote, info, timestamp)
	if err != nil {
		return nil, err
	}
	quote.ApplyCurrency(p.currency, info.ExchangeRate)
	return quote, nil
}

func (p *alphavantageProvider) FetchHistory(ctx context.Context, symbol string, from, to time.Time) (domain.Quotes, error) {
	timestamp := time.Now()
	info, err := p.symbolInfo(ctx, symbol)
	if err != nil {
		return nil, err
	}
	timeSeries, err := p.getTimeSeriesDaily(ctx, symbol)
	if err != nil {
		return nil, nil
	}
	quotes, err := p.quotesFromTimeSeriesDailyResponse(timeSeries, info, timestamp)
	if err != nil {
		return nil, err
	}
	quotes.ApplyCurrency(p.currency, info.ExchangeRate)
	return quotes, nil
}

func (p *alphavantageProvider) Close() error {
	return p.symbolInfoCache.Close()
}

func (p *alphavantageProvider) symbolInfo(ctx context.Context, symbol string) (*symbolInfo, error) {
	symbolInfo, err := p.symbolInfoCache.Get(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("unable to get symbol info for symbol '%s' via Alpha Vantage API (cause: %w)", symbol, err)
	}
	return symbolInfo, nil
}

func (p *alphavantageProvider) loadSymbolInfo(ctx context.Context, symbol string) (*symbolInfo, error) {
	symbolSearch, err := p.getSymbolSearch(ctx, symbol)
	if err != nil {
		return nil, err
	}
	var bestMatch *BestMatchResponse
	for _, match := range symbolSearch.BestMatches {
		if match.Symbol == symbol {
			bestMatch = &match
			break
		}
	}
	if bestMatch == nil {
		return nil, cache.ErrNotFound
	}
	if p.currency == bestMatch.Currency {
		return &symbolInfo{SourceCurrency: p.currency, ExchangeRate: 1.0}, nil
	}
	exchangeRateResponse, err := p.getCurrencyExchangeRate(ctx, bestMatch.Currency, p.currency)
	if err != nil {
		return nil, err
	}
	exchangeRate, err := p.parseSymbolCurrency(symbol, "ExchangeRate", exchangeRateResponse.RealtimeRate.ExchangeRate)
	if err != nil {
		return nil, err
	}
	return &symbolInfo{SourceCurrency: bestMatch.Currency, ExchangeRate: exchangeRate}, nil
}

func (p *alphavantageProvider) getSymbolSearch(ctx context.Context, symbol string) (*SymbolSearchResponse, error) {
	url := fmt.Sprintf("%s?function=SYMBOL_SEARCH&keywords=%s&apikey=%s", p.baseURL, symbol, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create symbol search request (cause: %w)", err)
	}
	rsp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send symbol search request (cause: %w)", err)
	}
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("symbol search failure (status: %s)", rsp.Status)
	}
	defer rsp.Body.Close()
	symbolSearch := &SymbolSearchResponse{}
	err = json.NewDecoder(rsp.Body).Decode(symbolSearch)
	if err != nil {
		return nil, fmt.Errorf("failed to decode symbol search response (cause: %w)", err)
	}
	err = symbolSearch.Validate()
	if err != nil {
		return nil, err
	}
	return symbolSearch, nil
}

func (p *alphavantageProvider) getCurrencyExchangeRate(ctx context.Context, from, to string) (*CurrencyExchangeRateResponse, error) {
	url := fmt.Sprintf("%s?function=CURRENCY_EXCHANGE_RATE&from_currency=%s&to_currency=%s&apikey=%s", p.baseURL, from, to, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create currency exchange rate request (cause: %w)", err)
	}
	rsp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send currency exchange rate request (cause: %w)", err)
	}
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("currency exchange rate failure (status: %s)", rsp.Status)
	}
	defer rsp.Body.Close()
	exchangeRate := &CurrencyExchangeRateResponse{}
	err = json.NewDecoder(rsp.Body).Decode(exchangeRate)
	if err != nil {
		return nil, fmt.Errorf("failed to decode currency exchange rate response (cause: %w)", err)
	}
	err = exchangeRate.Validate()
	if err != nil {
		return nil, err
	}
	return exchangeRate, nil
}

func (p *alphavantageProvider) getGlobalQuote(ctx context.Context, symbol string) (*GlobalQuoteResponse, error) {
	url := fmt.Sprintf("%s?function=GLOBAL_QUOTE&symbol=%s&apikey=%s", p.baseURL, symbol, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create global quote request (cause: %w)", err)
	}
	rsp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send global quote request (cause: %w)", err)
	}
	defer rsp.Body.Close()
	globalQuote := &GlobalQuoteResponse{}
	err = json.NewDecoder(rsp.Body).Decode(globalQuote)
	if err != nil {
		return nil, fmt.Errorf("failed to decode global quote response (cause: %w)", err)
	}
	return globalQuote, nil
}

func (p *alphavantageProvider) quoteFromGlobalQuoteResponse(rsp *GlobalQuoteResponse, info *symbolInfo, timestamp time.Time) (*domain.Quote, error) {
	open, err := p.parseSymbolCurrency(rsp.GlobalQuote.Symbol, "02. open", rsp.GlobalQuote.Open)
	if err != nil {
		return nil, err
	}
	high, err := p.parseSymbolCurrency(rsp.GlobalQuote.Symbol, "03. high", rsp.GlobalQuote.High)
	if err != nil {
		return nil, err
	}
	low, err := p.parseSymbolCurrency(rsp.GlobalQuote.Symbol, "04. low", rsp.GlobalQuote.Low)
	if err != nil {
		return nil, err
	}
	price, err := p.parseSymbolCurrency(rsp.GlobalQuote.Symbol, "05. price", rsp.GlobalQuote.Price)
	if err != nil {
		return nil, err
	}
	volume, err := p.parseSymbolAmount(rsp.GlobalQuote.Symbol, "06. volume", rsp.GlobalQuote.Volume)
	if err != nil {
		return nil, err
	}
	sourceTimestamp, err := p.parseSymbolTimestamp(rsp.GlobalQuote.Symbol, "07. latest trading day", rsp.GlobalQuote.LatestTradingDay)
	if err != nil {
		return nil, err
	}
	quote := &domain.Quote{
		Symbol:          rsp.GlobalQuote.Symbol,
		Timestamp:       timestamp,
		Currency:        info.SourceCurrency,
		Open:            open,
		High:            high,
		Low:             low,
		Close:           price,
		Price:           price,
		Volume:          volume,
		Source:          Name.String(),
		SourceTimestamp: sourceTimestamp,
	}
	return quote, nil
}

func (p *alphavantageProvider) getTimeSeriesDaily(ctx context.Context, symbol string) (*TimeSeriesDailyResponse, error) {
	url := fmt.Sprintf("%s?function=TIME_SERIES_DAILY&symbol=%s&apikey=%s", p.baseURL, symbol, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create time series daily request (cause: %w)", err)
	}
	rsp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send time series daily request (cause: %w)", err)
	}
	defer rsp.Body.Close()
	timeSeries := &TimeSeriesDailyResponse{}
	err = json.NewDecoder(rsp.Body).Decode(timeSeries)
	if err != nil {
		return nil, fmt.Errorf("failed to decode time series daily response (cause: %w)", err)
	}
	return timeSeries, nil
}

func (p *alphavantageProvider) quotesFromTimeSeriesDailyResponse(rsp *TimeSeriesDailyResponse, info *symbolInfo, timestamp time.Time) (domain.Quotes, error) {
	quotes := make(domain.Quotes, 0, len(rsp.TimeSeries))
	for date, daily := range rsp.TimeSeries {
		sourceTimestamp, err := p.parseSymbolTimestamp(rsp.MetaData.Symbol, "TimeSeries", date)
		if err != nil {
			return nil, err
		}
		quote, err := p.quoteFromDailyData(&daily, rsp.MetaData.Symbol, info, timestamp, sourceTimestamp)
		if err != nil {
			return nil, err
		}
		quotes = append(quotes, *quote)
	}
	return quotes, nil
}

func (p *alphavantageProvider) quoteFromDailyData(rsp *DailyData, symbol string, info *symbolInfo, timestamp, sourceTimestamp time.Time) (*domain.Quote, error) {
	open, err := p.parseSymbolCurrency(symbol, "1. open", rsp.Open)
	if err != nil {
		return nil, err
	}
	high, err := p.parseSymbolCurrency(symbol, "2. high", rsp.High)
	if err != nil {
		return nil, err
	}
	low, err := p.parseSymbolCurrency(symbol, "3. low", rsp.Low)
	if err != nil {
		return nil, err
	}
	close, err := p.parseSymbolCurrency(symbol, "4. price", rsp.Close)
	if err != nil {
		return nil, err
	}
	volume, err := p.parseSymbolAmount(symbol, "5. volume", rsp.Volume)
	if err != nil {
		return nil, err
	}
	quote := &domain.Quote{
		Symbol:          symbol,
		Timestamp:       timestamp,
		Currency:        info.SourceCurrency,
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
