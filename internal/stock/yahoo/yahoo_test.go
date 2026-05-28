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

package yahoo

import (
	"context"
	"testing"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/stock"
)

// compile-time check: Provider satisfies the interface
var _ stock.Provider = (*Provider)(nil)

func TestFetchQuote(t *testing.T) {
	p := NewProvider()
	ctx := context.Background()

	q, err := p.FetchQuote(ctx, "AAPL")
	if err != nil {
		t.Fatalf("FetchQuote: unexpected error: %v", err)
	}
	if q.Symbol != "AAPL" {
		t.Errorf("FetchQuote: symbol = %q, want AAPL", q.Symbol)
	}
	if q.Source != "mock-yahoo" {
		t.Errorf("FetchQuote: source = %q, want mock-yahoo", q.Source)
	}
	if q.Open == 0 {
		t.Errorf("FetchQuote: open is zero")
	}
	if q.Close == 0 {
		t.Errorf("FetchQuote: close is zero")
	}
}

func TestFetchHistory(t *testing.T) {
	p := NewProvider()
	ctx := context.Background()

	from := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC) // Mon–Fri

	quotes, err := p.FetchHistory(ctx, "MSFT", from, to)
	if err != nil {
		t.Fatalf("FetchHistory: unexpected error: %v", err)
	}

	// 5 business days (Mon–Fri)
	if len(quotes) != 5 {
		t.Errorf("FetchHistory: got %d quotes, want 5", len(quotes))
	}

	for i, q := range quotes {
		if q.Symbol != "MSFT" {
			t.Errorf("FetchHistory[%d]: symbol = %q, want MSFT", i, q.Symbol)
		}
		if q.Source != "mock-yahoo" {
			t.Errorf("FetchHistory[%d]: source = %q, want mock-yahoo", i, q.Source)
		}
		if !q.Timestamp.After(from.Add(-time.Minute)) || !q.Timestamp.Before(to.AddDate(0, 0, 1)) {
			t.Errorf("FetchHistory[%d]: timestamp %s out of range [%s, %s]", i, q.Timestamp, from, to)
		}
		if q.Open == q.Close && q.High == q.Low {
			t.Errorf("FetchHistory[%d]: prices are all equal — expected some volatility", i)
		}
	}
}

func TestFetchHistoryWeekend(t *testing.T) {
	p := NewProvider()
	ctx := context.Background()

	// Test a weekend-only range (Saturday to Sunday)
	sat := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	sun := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)

	quotes, err := p.FetchHistory(ctx, "TEST", sat, sun)
	if err != nil {
		t.Fatalf("FetchHistory weekend: unexpected error: %v", err)
	}

	if len(quotes) != 0 {
		t.Errorf("FetchHistory weekend: got %d quotes, want 0 (no business days)", len(quotes))
	}
}

func TestFetchHistoryInvalidRange(t *testing.T) {
	p := NewProvider()
	ctx := context.Background()

	from := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC) // before from

	_, err := p.FetchHistory(ctx, "ERR", from, to)
	if err == nil {
		t.Error("FetchHistory with from > to: expected error, got nil")
	}
}

func TestDeterministic(t *testing.T) {
	ctx := context.Background()
	from := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)

	// Two providers with same seed should produce identical results
	p1 := NewProvider()
	p2 := NewProvider()

	q1, _ := p1.FetchHistory(ctx, "DET", from, to)
	q2, _ := p2.FetchHistory(ctx, "DET", from, to)

	if len(q1) != len(q2) {
		t.Fatalf("Deterministic: lengths differ: %d vs %d", len(q1), len(q2))
	}

	for i := range q1 {
		if q1[i].Close != q2[i].Close {
			t.Errorf("Deterministic[%d]: close differs: %v vs %v", i, q1[i].Close, q2[i].Close)
		}
	}
}

func TestQuotesSorted(t *testing.T) {
	p := NewProvider()
	ctx := context.Background()

	from := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC)

	quotes, err := p.FetchHistory(ctx, "SORT", from, to)
	if err != nil {
		t.Fatalf("FetchHistory: unexpected error: %v", err)
	}

	if len(quotes) < 2 {
		t.Skip("not enough quotes to check sort order")
	}

	for i := 1; i < len(quotes); i++ {
		if !quotes[i].Timestamp.After(quotes[i-1].Timestamp) {
			t.Errorf("Quotes not sorted: quotes[%d].Timestamp %s <= quotes[%d].Timestamp %s",
				i, quotes[i].Timestamp, i-1, quotes[i-1].Timestamp)
		}
	}
}

// Test domain types are properly set on every quote
func TestQuoteFields(t *testing.T) {
	p := NewProvider()
	ctx := context.Background()

	q, err := p.FetchQuote(ctx, "NVDA")
	if err != nil {
		t.Fatalf("FetchQuote: %v", err)
	}

	// All price fields should be > 0
	if q.Open <= 0 || q.High <= 0 || q.Low <= 0 || q.Close <= 0 {
		t.Errorf("Price fields not positive: O=%.2f H=%.2f L=%.2f C=%.2f", q.Open, q.High, q.Low, q.Close)
	}

	// High >= Low
	if q.High < q.Low {
		t.Errorf("High (%.2f) < Low (%.2f)", q.High, q.Low)
	}

	// Volume > 0
	if q.Volume <= 0 {
		t.Errorf("Volume is zero")
	}

	// Timestamp is not zero
	if q.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}
}

// Verify the Provider implements the full stock.Provider interface.
func TestInterfaceCompliance(t *testing.T) {
	var p stock.Provider = NewProvider()
	_ = p // use it

	ctx := context.Background()

	// Both methods must work without panic
	q, err := p.FetchQuote(ctx, "IFACE")
	if err != nil || q == nil {
		t.Error("FetchQuote via interface failed")
	}

	now := time.Now()
	history, err := p.FetchHistory(ctx, "IFACE", now.AddDate(0, -1, 0), now)
	if err != nil || history == nil {
		t.Error("FetchHistory via interface failed")
	}
}
