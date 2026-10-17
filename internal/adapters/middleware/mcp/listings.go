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
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/tdrn-org/adjudex-mcp/internal/domain"
)

func addListingTools(server *mcp.Server, runtime Runtime) {
	addListingSearchTool(server, runtime)
}

func addListingSearchTool(server *mcp.Server, runtime Runtime) {
	server.AddTool(&mcp.Tool{
		Name:        "listing_search",
		Description: "Searches for a listing by ticker symbol, WKN, ISIN, or company name. Returns matching candidates with their provider source and the listing ID that the other tools take.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"The search query — WKN (e.g. A413X6), ISIN, or company name (e.g. CoreWeave)."}},"required":["query"]}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return nil, fmt.Errorf("parsing arguments: %w", err)
		}

		results, err := runtime.MarketDataProvider().SearchListings(ctx, args.Query)
		if err != nil {
			return nil, fmt.Errorf("search failure: %w", err)
		}
		// Register what is handed out: the listing ID is all a client gets back
		// later, and only a registered ID can be resolved into the instrument it
		// identifies. An unregistered result would be a candidate that cannot be
		// used, so a registration failure fails the search.
		for _, listing := range results {
			err := runtime.DataStore().AddListing(ctx, listing)
			if err != nil {
				return nil, fmt.Errorf("registering listing %q: %w", listing.ID(), err)
			}
		}
		return newToolResult(domain.NewListingInfos(results))
	})
}
