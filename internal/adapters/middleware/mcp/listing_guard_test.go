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
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-finance"
)

func validTestListing(t *testing.T) finance.Listing {
	t.Helper()
	return *finance.NewListing("test", finance.ListingAttributeKeySymbol).
		InstrumentName("Test Instrument").
		InstrumentType(finance.InstrumentTypeEquity).
		Attribute(finance.ListingAttributeKeySymbol, "TEST").
		Build().MustValidate()
}

// TestValidateOptionalListingAllowsUnset makes sure a filter argument stays
// optional: an unset listing means "no filter" and must not be rejected.
//
// A malformed listing ID cannot even be constructed: it would have to be parsed
// first, and parsing rejects it - see finance.ParseListingID.
func TestValidateOptionalListingAllowsUnset(t *testing.T) {
	require.NoError(t, validateOptionalListing(finance.Listing{}))
	require.NoError(t, validateOptionalListing(validTestListing(t)))
}
