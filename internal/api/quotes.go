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

package api

import (
	"net/http"
	"strconv"

	"github.com/tdrn-org/adjudex-mcp/internal/mcp/tools"
)

// --- GET /api/v1/quotes/{symbol} ---

func (h *handler) quoteGet(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	q, err := tools.QuoteGet(r.Context(), h.stores.Quote, tools.QuoteGetArgs{Symbol: symbol})
	if err != nil {
		handleNotFound(w, err)
		return
	}
	writeJSON(w, http.StatusOK, q)
}

// --- GET /api/v1/quotes/{symbol}/history ---

func (h *handler) quoteHistory(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	args := tools.QuoteHistoryArgs{
		Symbol: symbol,
		From:   r.URL.Query().Get("from"),
		To:     r.URL.Query().Get("to"),
	}
	quotes, err := tools.QuoteHistory(r.Context(), h.stores.Quote, args)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, quotes)
}

// --- GET /api/v1/quotes/{symbol}/indicator ---

func (h *handler) quoteIndicator(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	period, _ := strconv.Atoi(r.URL.Query().Get("period"))
	if period <= 0 {
		period = 14
	}
	args := tools.QuoteIndicatorArgs{
		Symbol:        symbol,
		IndicatorType: r.URL.Query().Get("type"),
		Period:        period,
	}
	iv, err := tools.QuoteIndicator(r.Context(), h.stores.Quote, args)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, iv)
}
