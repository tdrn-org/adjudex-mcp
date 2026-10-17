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

package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tdrn-org/go-finance"
)

func testPositionListing(t *testing.T) finance.Listing {
	t.Helper()
	return *finance.NewListing("test", finance.ListingAttributeKeySymbol).
		InstrumentName("Test Instrument").
		InstrumentType(finance.InstrumentTypeEquity).
		Attribute(finance.ListingAttributeKeySymbol, "TEST").
		Build().MustValidate()
}

// TestPositionJSONIncludesListingInfo pins the display contract: a client cannot
// decode the opaque listing value, so every position has to carry the instrument
// name — otherwise a UI can only show the blob.
func TestPositionJSONIncludesListingInfo(t *testing.T) {
	position := Position{
		ID:         "position-1",
		Listing:    testPositionListing(t),
		Currency:   finance.CurrencyEUR,
		Quantity:   1,
		EntryPrice: 100,
		EntryDate:  time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC),
	}

	encoded, err := json.Marshal(position)
	require.NoError(t, err)

	doc := map[string]any{}
	require.NoError(t, json.Unmarshal(encoded, &doc))
	require.Contains(t, doc, "listing", "the listing value must stay usable for follow-up requests")
	require.Contains(t, doc, "listing_info")
	// The value matters, not just its presence: a listing embedded in a struct
	// marshalled by value is not addressable, and a pointer-only Marshaler would
	// silently render as an empty object here.
	require.Equal(t, position.Listing.ID(), doc["listing"], "the listing must be the ID: %s", encoded)

	info, ok := doc["listing_info"].(map[string]any)
	require.True(t, ok, "listing_info must be an object: %s", encoded)
	require.Equal(t, "Test Instrument", info["instrument_name"])
	require.Equal(t, "equity", info["instrument_type"])
	require.Equal(t, "test", info["provider"])
}

// TestHoldingJSONStaysFlat guards the effect of Position.MarshalJSON on Holding,
// which embeds Position: the payload must stay flat. A nested "Position" object
// would silently break every consumer of the holdings endpoint.
func TestHoldingJSONStaysFlat(t *testing.T) {
	position := Position{
		ID:         "position-1",
		Listing:    testPositionListing(t),
		Currency:   finance.CurrencyEUR,
		Quantity:   2,
		EntryPrice: 100,
		EntryDate:  time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC),
	}
	holding := NewHolding(position, finance.ListingPrice{
		Listing:  position.Listing,
		Currency: finance.CurrencyEUR,
		Price:    110,
	})

	encoded, err := json.Marshal(holding)
	require.NoError(t, err)

	doc := map[string]any{}
	require.NoError(t, json.Unmarshal(encoded, &doc))
	require.NotContains(t, doc, "Position", "Holding JSON must stay flat: %s", encoded)
	require.Contains(t, doc, "id")
	require.Contains(t, doc, "entry_price")
	require.Contains(t, doc, "current_price")
	require.Contains(t, doc, "listing")
	require.Equal(t, position.Listing.ID(), doc["listing"], "the listing must be the ID: %s", encoded)
	require.Equal(t, "Test Instrument", doc["listing_info"].(map[string]any)["instrument_name"])
}

// TestPositionJSONToleratesInvalidListing makes sure marshalling never panics or
// fails because of a broken listing — an invalid value yields an empty description.
func TestPositionJSONToleratesInvalidListing(t *testing.T) {
	position := Position{ID: "position-1", Currency: finance.CurrencyEUR}

	encoded, err := json.Marshal(position)
	require.NoError(t, err)

	doc := map[string]any{}
	require.NoError(t, json.Unmarshal(encoded, &doc))
	info, ok := doc["listing_info"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "", info["instrument_name"])
}
