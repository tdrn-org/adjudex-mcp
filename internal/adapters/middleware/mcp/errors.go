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

	"github.com/tdrn-org/go-finance"
)

// quoteUnavailable reports whether err means "the provider cannot deliver a
// live quote right now" rather than "something is broken".
//
// A closed exchange, a quote the provider is still collecting and a listing
// without any quote all belong to this group: in each case the last quote the
// store knows is a better answer than an error. Whether that quote is *good*
// enough is not for the server to decide - the caller sees its SourceTimestamp
// and judges the age itself.
func quoteUnavailable(err error) bool {
	return errors.Is(err, finance.ErrQuoteExchangeClosed) ||
		errors.Is(err, finance.ErrRequestPending) ||
		errors.Is(err, finance.ErrQuoteNotAvailable)
}

// translateProviderError restates a finance provider error in terms a caller
// can act on.
//
// A caller that only sees "quote exchange closed" cannot tell whether the
// server is broken or the market is simply shut. The original error stays
// wrapped, so errors.Is keeps working for callers that need the machine
// readable classification rather than the message.
func translateProviderError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, finance.ErrQuoteExchangeClosed):
		return fmt.Errorf("no live quote, the exchange is closed: %w", err)
	case errors.Is(err, finance.ErrRequestPending):
		return fmt.Errorf("quote is still being collected by the provider, try again shortly: %w", err)
	case errors.Is(err, finance.ErrQuoteNotAvailable):
		return fmt.Errorf("provider has no quote for this listing: %w", err)
	case errors.Is(err, finance.ErrHistoryNotAvailable):
		return fmt.Errorf("provider has no history for this listing: %w", err)
	case errors.Is(err, finance.ErrEmptyListing):
		return fmt.Errorf("a listing is required: %w", err)
	case errors.Is(err, finance.ErrListingProviderMismatch):
		return fmt.Errorf("listing belongs to a different provider: %w", err)
	case errors.Is(err, finance.ErrListingSearchRestricted):
		return fmt.Errorf("listing search is restricted: %w", err)
	default:
		return err
	}
}
