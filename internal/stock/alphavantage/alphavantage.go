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

// Package alphavantage provides a stock quote provider backed by the Alpha Vantage API.
// Free tier: 25 calls/day, 5 calls/min. Best for European/international stocks.
package alphavantage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/adjudex-mcp/internal/stock"
)

// compile-time interface check
var _ stock.Provider = (*Provider)(nil)

const baseURL = "https://www.alphavantage.co/query"

// Provider implements stock.Provider via Alpha Vantage REST API.
type Provider struct {
	apiKey     string
	httpClient *http.Client
}

// NewProvider creates an Alpha Vantage provider with the given API key.
func NewProvider(apiKey string) *Provider {
	return &Provider{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// FetchQuote returns the latest quote for a symbol.
func (p *Provider) FetchQuote(ctx context.Context, symbol string) (*domain.Quote, error) {
	url := fmt.Sprintf("%s?function=GLOBAL_QUOTE&symbol=%s&apikey=%s", baseURL, symbol, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("alphavantage: quote request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("alphavantage: quote call: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		GlobalQuote map[string]string `json:"Global Quote"`
		Information string            `json:"Information"`
		Note        string            `json:"Note"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("alphavantage: decode quote: %w", err)
	}
	if result.Note != "" {
		return nil, fmt.Errorf("alphavantage: rate limit — %s", result.Note)
	}
	if result.GlobalQuote == nil || len(result.GlobalQuote) == 0 {
		return nil, fmt.Errorf("alphavantage: symbol %q not found", symbol)
	}

	q := result.GlobalQuote
	ts, _ := time.Parse("2006-01-02", q["07. latest trading day"])
	if ts.IsZero() {
		ts = time.Now()
	}

	return &domain.Quote{
		Symbol:    symbol,
		Timestamp: ts,
		Open:      parseFloat(q["02. open"]),
		High:      parseFloat(q["03. high"]),
		Low:       parseFloat(q["04. low"]),
		Close:     parseFloat(q["05. price"]),
		Volume:    parseInt64(q["06. volume"]),
		Source:    "alphavantage",
	}, nil
}

// FetchHistory returns historical quotes for a symbol.
func (p *Provider) FetchHistory(ctx context.Context, symbol string, from, to time.Time) ([]domain.Quote, error) {
	url := fmt.Sprintf("%s?function=TIME_SERIES_DAILY&symbol=%s&outputsize=full&apikey=%s",
		baseURL, symbol, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("alphavantage: history request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("alphavantage: history call: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		TimeSeriesDaily map[string]map[string]string `json:"Time Series (Daily)"`
		Note            string                       `json:"Note"`
		Information     string                       `json:"Information"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("alphavantage: decode history: %w", err)
	}
	if result.Note != "" {
		return nil, fmt.Errorf("alphavantage: rate limit — %s", result.Note)
	}
	if result.TimeSeriesDaily == nil || len(result.TimeSeriesDaily) == 0 {
		return nil, fmt.Errorf("alphavantage: no history for %q", symbol)
	}

	// Collect and sort dates (Alpha Vantage returns newest first)
	dates := make([]string, 0, len(result.TimeSeriesDaily))
	for d := range result.TimeSeriesDaily {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	quotes := make([]domain.Quote, 0, len(dates))
	for _, d := range dates {
		ts, _ := time.Parse("2006-01-02", d)
		if ts.Before(from) || ts.After(to) {
			continue
		}
		v := result.TimeSeriesDaily[d]
		quotes = append(quotes, domain.Quote{
			Symbol:    symbol,
			Timestamp: ts,
			Open:      parseFloat(v["1. open"]),
			High:      parseFloat(v["2. high"]),
			Low:       parseFloat(v["3. low"]),
			Close:     parseFloat(v["4. close"]),
			Volume:    parseInt64(v["5. volume"]),
			Source:    "alphavantage",
		})
	}
	return quotes, nil
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func parseInt64(s string) int64 {
	var i int64
	fmt.Sscanf(s, "%d", &i)
	return i
}
