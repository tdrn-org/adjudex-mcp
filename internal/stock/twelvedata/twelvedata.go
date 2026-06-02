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

// Package twelvedata provides a stock quote provider backed by the Twelve Data API.
// Free tier: 800 calls/day, 8 calls/min, 4-hour delayed data.
package twelvedata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/adjudex-mcp/internal/stock"
)

// compile-time interface check
var _ stock.Provider = (*Provider)(nil)

const baseURL = "https://api.twelvedata.com"

// Provider implements stock.Provider via Twelve Data REST API.
type Provider struct {
	apiKey     string
	httpClient *http.Client
}

// NewProvider creates a Twelve Data provider with the given API key.
func NewProvider(apiKey string) *Provider {
	return &Provider{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// FetchQuote returns the latest quote for a symbol.
func (p *Provider) FetchQuote(ctx context.Context, symbol string) (*domain.Quote, error) {
	url := fmt.Sprintf("%s/quote?symbol=%s&apikey=%s", baseURL, symbol, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("twelvedata: quote request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("twelvedata: quote call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("twelvedata: symbol %q not found", symbol)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("twelvedata: rate limit exceeded")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("twelvedata: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Symbol   string `json:"symbol"`
		Datetime string `json:"datetime"`
		Open     string `json:"open"`
		High     string `json:"high"`
		Low      string `json:"low"`
		Close    string `json:"close"`
		Volume   string `json:"volume"`
		Status   string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("twelvedata: decode quote: %w", err)
	}
	if result.Status == "error" {
		return nil, fmt.Errorf("twelvedata: quote error for %q", symbol)
	}

	ts, _ := time.Parse("2006-01-02 15:04:05", result.Datetime)
	if ts.IsZero() {
		ts, _ = time.Parse("2006-01-02", result.Datetime)
	}

	return &domain.Quote{
		Symbol:    symbol,
		Timestamp: ts,
		Open:      parseFloat(result.Open),
		High:      parseFloat(result.High),
		Low:       parseFloat(result.Low),
		Close:     parseFloat(result.Close),
		Volume:    parseInt64(result.Volume),
		Source:    "twelvedata",
	}, nil
}

// FetchHistory returns historical quotes for a symbol.
func (p *Provider) FetchHistory(ctx context.Context, symbol string, from, to time.Time) ([]domain.Quote, error) {
	url := fmt.Sprintf(
		"%s/time_series?symbol=%s&interval=1day&start_date=%s&end_date=%s&outputsize=5000&apikey=%s",
		baseURL, symbol,
		from.Format("2006-01-02"),
		to.Format("2006-01-02"),
		p.apiKey,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("twelvedata: history request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("twelvedata: history call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("twelvedata: symbol %q not found", symbol)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("twelvedata: rate limit exceeded")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("twelvedata: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Meta   map[string]any `json:"meta"`
		Values []struct {
			Datetime string `json:"datetime"`
			Open     string `json:"open"`
			High     string `json:"high"`
			Low      string `json:"low"`
			Close    string `json:"close"`
			Volume   string `json:"volume"`
		} `json:"values"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("twelvedata: decode history: %w", err)
	}
	if result.Status == "error" {
		return nil, fmt.Errorf("twelvedata: history error for %q", symbol)
	}

	// Twelve Data returns newest first — reverse for chronological order
	quotes := make([]domain.Quote, 0, len(result.Values))
	for i := len(result.Values) - 1; i >= 0; i-- {
		v := result.Values[i]
		ts, _ := time.Parse("2006-01-02", v.Datetime)
		if ts.IsZero() {
			ts, _ = time.Parse("2006-01-02 15:04:05", v.Datetime)
		}
		quotes = append(quotes, domain.Quote{
			Symbol:    symbol,
			Timestamp: ts,
			Open:      parseFloat(v.Open),
			High:      parseFloat(v.High),
			Low:       parseFloat(v.Low),
			Close:     parseFloat(v.Close),
			Volume:    parseInt64(v.Volume),
			Source:    "twelvedata",
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
