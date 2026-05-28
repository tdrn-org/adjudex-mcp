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

// --- POST /api/v1/strategies ---

func (h *handler) strategyCreate(w http.ResponseWriter, r *http.Request) {
	var args tools.StrategyCreateArgs
	if err := decodeBody(r, &args); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	s, err := tools.StrategyCreate(r.Context(), h.stores.Strategy, args)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

// --- GET /api/v1/strategies ---

func (h *handler) strategyList(w http.ResponseWriter, r *http.Request) {
	strategies, err := tools.StrategyList(r.Context(), h.stores.Strategy, tools.StrategyListArgs{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, strategies)
}

// --- GET /api/v1/strategies/{id} ---

func (h *handler) strategyGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s, err := tools.StrategyGet(r.Context(), h.stores.Strategy, tools.StrategyGetArgs{ID: id})
	if err != nil {
		handleNotFound(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// --- DELETE /api/v1/strategies/{id} ---

func (h *handler) strategyDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := tools.StrategyDelete(r.Context(), h.stores.Strategy, tools.StrategyDeleteArgs{ID: id}); err != nil {
		handleNotFound(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

// --- POST /api/v1/strategies/{id}/backtest ---

func (h *handler) strategyBacktest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var args tools.StrategyBacktestArgs
	if err := decodeBody(r, &args); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	args.StrategyID = id
	result, err := tools.StrategyBacktest(r.Context(), h.stores.Strategy, h.stores.Quote, h.stores.Trade, args)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
