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

package tracker_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/adjudex-mcp/internal/stock/tracker"
	"github.com/tdrn-org/adjudex-mcp/internal/stock/tracker/alphavantage"
	"github.com/tdrn-org/adjudex-mcp/internal/stock/tracker/demo"
	"github.com/tdrn-org/adjudex-mcp/internal/stock/tracker/twelvedata"
)

const defaultCurrency string = "EUR"
const defaultQuoteSymbol string = "CRWV"

func TestDemoProvider(t *testing.T) {
	provider := demo.NewProvider(defaultCurrency)
	testNamedProvider(t, provider, defaultQuoteSymbol, true)
	require.NoError(t, provider.Close())
}

func TestTwelveDataProvider(t *testing.T) {
	t.SkipNow()
	provider, err := twelvedata.NewProvider(defaultCurrency, "")
	require.NoError(t, err)
	testNamedProvider(t, provider, defaultQuoteSymbol, true)
	require.NoError(t, provider.Close())
}

func TestAlphaVantageProvider(t *testing.T) {
	t.SkipNow()
	provider, err := alphavantage.NewProvider(defaultCurrency, "")
	require.NoError(t, err)
	testNamedProvider(t, provider, defaultQuoteSymbol, true)
	require.NoError(t, provider.Close())
}

func testNamedProvider(t *testing.T, provider tracker.Provider, symbol string, query bool) {
	providerName := provider.Name()
	require.NotEmpty(t, providerName)
	t.Log(providerName)
	if !query {
		return
	}
	_, err := provider.ResolveSymbols(t.Context(), symbol)
	require.NoError(t, err)
	quote, err := provider.FetchQuote(t.Context(), symbol)
	require.NoError(t, err)
	t.Log(quote)
	require.NotNil(t, quote)
	require.Equal(t, providerName, tracker.ProviderName(quote.Source))
	now := time.Now()
	quotes, err := provider.FetchHistory(t.Context(), symbol, now.Add(-7*24*time.Hour), now)
	require.NoError(t, err)
	t.Log(quotes)
	require.NotNil(t, quotes)
}
