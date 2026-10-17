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
	"strings"

	"github.com/tdrn-org/go-finance"
)

// ListingInfo describes a listing in a form a client can work with: a stable,
// provider-independent set of fields plus the listing ID that has to be
// passed back to the quote tools.
//
// The provider-specific identifying attributes are deliberately NOT expanded into
// individual fields. Which attributes identify an instrument is up to the
// provider (Twelve Data supplies ticker and MIC, Consorsbank ISIN and exchange),
// so one field per attribute would grow with every provider and force every
// caller to know which of them happen to be populated. Identifier condenses
// whatever the provider supplied into a single string instead.
type ListingInfo struct {
	// Listing is the identity carrier to pass back to the quote tools unchanged.
	// A listing is stored as compressed, base64 encoded text and cannot be
	// reconstructed from the fields below.
	Listing finance.Listing `json:"listing"`
	// InstrumentName is the human readable instrument name.
	InstrumentName string `json:"instrument_name"`
	// InstrumentType is the instrument's type (equity, ETF, ...).
	InstrumentType finance.InstrumentType `json:"instrument_type"`
	// Provider names the provider that issued this listing.
	Provider string `json:"provider"`
	// Identifier holds the identifying attributes the provider supplied, in the
	// order defined by finance.ListingAttributeKeys, e.g.
	// "ISIN:US0378331005 MIC:XNGS". Which attributes appear is entirely up to the
	// provider; the format stays stable.
	Identifier string `json:"identifier"`
}

// NewListingInfo describes the given listing.
//
// The listing must be valid — see [finance.Listing.Validate]. Callers at an API
// boundary check that first: a listing without identity has nothing to describe.
func NewListingInfo(listing finance.Listing) ListingInfo {
	return ListingInfo{
		Listing:        listing,
		InstrumentName: listing.InstrumentName(),
		InstrumentType: listing.InstrumentType(),
		Provider:       listing.Provider(),
		Identifier:     listingIdentifier(listing),
	}
}

// NewListingInfos describes each of the given listings.
func NewListingInfos(listings []finance.Listing) []ListingInfo {
	infos := make([]ListingInfo, 0, len(listings))
	for i := range listings {
		infos = append(infos, NewListingInfo(listings[i]))
	}
	return infos
}

// listingIdentifier condenses the listing's attributes into a single string.
//
// The iteration order is the one of finance.ListingAttributeKeys, so the result
// is deterministic and comparable across calls — unlike iterating the attribute
// map, which would yield a different order on every run.
func listingIdentifier(listing finance.Listing) string {
	buffer := &strings.Builder{}
	for _, key := range finance.ListingAttributeKeys {
		value, ok := listing.Attribute(key)
		if !ok || value == "" {
			continue
		}
		if buffer.Len() > 0 {
			buffer.WriteRune(' ')
		}
		buffer.WriteString(strings.ToUpper(string(key)))
		buffer.WriteRune(':')
		buffer.WriteString(value)
	}
	return buffer.String()
}
