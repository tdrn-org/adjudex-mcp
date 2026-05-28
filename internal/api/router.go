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

// Package api provides the REST API for the adjudex web frontend.
// All endpoints mirror the MCP tools defined in internal/mcp/tools.
package api

import (
	"encoding/json"
	"net/http"

	"github.com/tdrn-org/adjudex-mcp/internal/mcp"
	"github.com/tdrn-org/go-httpserver"
)

// Router registers all API routes on the provided httpserver.Instance.
// Routes are prefixed with /api/v1/ and mirror the 21 MCP tools.
func Router(srv *httpserver.Instance, stores mcp.StoreSet) {
	h := &handler{stores: stores}

	srv.HandleFunc("POST /api/v1/portfolios", h.portfolioCreate)
	srv.HandleFunc("GET /api/v1/portfolios", h.portfolioList)
	srv.HandleFunc("GET /api/v1/portfolios/{id}", h.portfolioGet)
	srv.HandleFunc("DELETE /api/v1/portfolios/{id}", h.portfolioDelete)
	srv.HandleFunc("POST /api/v1/portfolios/{id}/positions", h.portfolioAddPosition)
	srv.HandleFunc("DELETE /api/v1/portfolios/{pid}/positions/{posid}", h.portfolioRemovePosition)
	srv.HandleFunc("GET /api/v1/portfolios/{id}/holdings", h.portfolioGetHoldings)

	srv.HandleFunc("GET /api/v1/quotes/{symbol}", h.quoteGet)
	srv.HandleFunc("GET /api/v1/quotes/{symbol}/history", h.quoteHistory)
	srv.HandleFunc("GET /api/v1/quotes/{symbol}/indicator", h.quoteIndicator)

	srv.HandleFunc("POST /api/v1/alerts", h.alertCreate)
	srv.HandleFunc("GET /api/v1/alerts", h.alertList)
	srv.HandleFunc("PUT /api/v1/alerts/{id}/acknowledge", h.alertAcknowledge)
	srv.HandleFunc("DELETE /api/v1/alerts/{id}", h.alertDelete)

	srv.HandleFunc("POST /api/v1/strategies", h.strategyCreate)
	srv.HandleFunc("GET /api/v1/strategies", h.strategyList)
	srv.HandleFunc("GET /api/v1/strategies/{id}", h.strategyGet)
	srv.HandleFunc("DELETE /api/v1/strategies/{id}", h.strategyDelete)
	srv.HandleFunc("POST /api/v1/strategies/{id}/backtest", h.strategyBacktest)

	srv.HandleFunc("GET /api/v1/trades", h.tradeList)
	srv.HandleFunc("GET /api/v1/trades/{id}", h.tradeGet)
}

// handler holds the store references injected into all endpoints.
type handler struct {
	stores mcp.StoreSet
}

// --- JSON helpers ---

// writeJSON marshals v as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, `{"error":"json encode failed"}`, http.StatusInternalServerError)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// decodeBody decodes the request body JSON into v.
func decodeBody(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
