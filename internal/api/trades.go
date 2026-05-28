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

	"github.com/tdrn-org/adjudex-mcp/internal/mcp/tools"
)

// --- GET /api/v1/trades ---

func (h *handler) tradeList(w http.ResponseWriter, r *http.Request) {
	args := tools.TradeListArgs{
		Symbol:     r.URL.Query().Get("symbol"),
		StrategyID: r.URL.Query().Get("strategy_id"),
	}
	trades, err := tools.TradeList(r.Context(), h.stores.Trade, args)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, trades)
}

// --- GET /api/v1/trades/{id} ---

func (h *handler) tradeGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := tools.TradeGet(r.Context(), h.stores.Trade, tools.TradeGetArgs{ID: id})
	if err != nil {
		handleNotFound(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}
