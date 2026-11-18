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

package mcp

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-finance"
)

// TestTranslateProviderErrorStatesWhatHappened makes sure every provider
// sentinel reaches the caller as a sentence that explains the situation -
// not as a bare "quote exchange closed" that reads like a defect.
func TestTranslateProviderErrorStatesWhatHappened(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"exchange closed", finance.ErrQuoteExchangeClosed, "the exchange is closed"},
		{"request pending", finance.ErrRequestPending, "still being collected"},
		{"quote not available", finance.ErrQuoteNotAvailable, "no quote for this listing"},
		{"history not available", finance.ErrHistoryNotAvailable, "no history for this listing"},
		{"empty listing", finance.ErrEmptyListing, "a listing is required"},
		{"provider mismatch", finance.ErrListingProviderMismatch, "belongs to a different provider"},
		{"search restricted", finance.ErrListingSearchRestricted, "search is restricted"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			translated := translateProviderError(test.err)
			require.Error(t, translated)
			require.Contains(t, translated.Error(), test.expected)
			require.ErrorIs(t, translated, test.err, "the sentinel must stay wrapped")
		})
	}
}

// TestTranslateProviderErrorKeepsUnknownErrors makes sure an unclassified
// error is not swallowed and not rewritten: hiding an unknown failure behind a
// friendly sentence would cost us the diagnosis.
func TestTranslateProviderErrorKeepsUnknownErrors(t *testing.T) {
	unknown := errors.New("disk on fire")
	require.Equal(t, unknown, translateProviderError(unknown))
}

// TestTranslateProviderErrorHandlesNil keeps the helper safe to call without a
// guard, so call sites do not need their own nil check.
func TestTranslateProviderErrorHandlesNil(t *testing.T) {
	require.NoError(t, translateProviderError(nil))
}

// TestTranslateProviderErrorWrapsThroughContext documents the behaviour the
// quote handler relies on: a sentinel wrapped by the provider still translates.
func TestTranslateProviderErrorWrapsThroughContext(t *testing.T) {
	wrapped := fmt.Errorf("streaming market data: %w", finance.ErrQuoteExchangeClosed)
	translated := translateProviderError(wrapped)
	require.ErrorIs(t, translated, finance.ErrQuoteExchangeClosed)
	require.Contains(t, translated.Error(), "the exchange is closed")
}

// TestQuoteUnavailableClassifiesOnlyMarketConditions pins down which errors
// fall back to the stored quote. A closed exchange, a pending request and a
// missing quote do; a broken provider does not - that one must surface.
func TestQuoteUnavailableClassifiesOnlyMarketConditions(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"exchange closed", finance.ErrQuoteExchangeClosed, true},
		{"request pending", finance.ErrRequestPending, true},
		{"quote not available", finance.ErrQuoteNotAvailable, true},
		{"wrapped exchange closed", fmt.Errorf("query: %w", finance.ErrQuoteExchangeClosed), true},
		{"history not available", finance.ErrHistoryNotAvailable, false},
		{"empty listing", finance.ErrEmptyListing, false},
		{"unknown", errors.New("connection refused"), false},
		{"nil", nil, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, quoteUnavailable(test.err))
		})
	}
}
