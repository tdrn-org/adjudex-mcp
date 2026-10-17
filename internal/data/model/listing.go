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

package model

import (
	"context"
	_ "embed"

	"github.com/tdrn-org/go-database"
	"github.com/tdrn-org/go-finance"
)

type Listing struct {
	ID             string                  `db:"id"`
	Listing        finance.ProviderListing `db:"listing"`
	Provider       string                  `db:"provider"`
	InstrumentName string                  `db:"instrument_name"`
	InstrumentType string                  `db:"instrument_type"`
}

//go:embed listing.insert.sql
var insertListingSQL string

//go:embed listing_attribute.insert.sql
var insertListingAttributeSQL string

func InsertListing(ctx context.Context, tx *database.Tx, listing finance.ProviderListing) (*Listing, error) {
	l := &Listing{
		ID:             listing.ID(),
		Listing:        listing,
		Provider:       listing.Provider(),
		InstrumentName: listing.InstrumentName(),
		InstrumentType: string(listing.InstrumentType()),
	}
	err := tx.ExecTx(ctx, insertListingSQL,
		l.ID,
		&l.Listing,
		l.Provider,
		l.InstrumentName,
		l.InstrumentType)
	if err != nil {
		return nil, err
	}
	for _, key := range finance.ListingAttributeKeys {
		value, ok := listing.Attribute(key)
		if !ok {
			continue
		}
		err := tx.ExecTx(ctx, insertListingAttributeSQL,
			l.ID,
			key,
			value)
		if err != nil {
			return nil, err
		}
	}
	return l, nil
}

//go:embed listing.select.sql
var selectListingSQL string

func SelectListing(ctx context.Context, tx *database.Tx, id string) (*Listing, error) {
	l := &Listing{
		ID: id,
	}
	row, err := tx.QueryRowTx(ctx, selectListingSQL, l.ID)
	if err != nil {
		return nil, err
	}
	err = database.ScanRow(row, l,
		"listing",
		"provider",
		"instrument_name",
		"instrument_type")
	if database.NoRows(err) {
		l = nil
	} else if err != nil {
		return nil, err
	}
	return l, nil
}

//go:embed listing.select_by_positions.sql
var selectListingByPositionsSQL string

func SelectListingsByPositions(ctx context.Context, tx *database.Tx) ([]Listing, error) {
	rows, err := tx.QueryTx(ctx, selectListingByPositionsSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ls := make([]Listing, 0)
	for rows.Next() {
		l := &Listing{}
		err = database.Scan(rows, l)
		if err != nil {
			return nil, err
		}
		ls = append(ls, *l)
	}
	return ls, nil
}
