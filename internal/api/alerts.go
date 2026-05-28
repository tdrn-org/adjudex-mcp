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

// --- POST /api/v1/alerts ---

func (h *handler) alertCreate(w http.ResponseWriter, r *http.Request) {
	var args tools.AlertCreateArgs
	if err := decodeBody(r, &args); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	a, err := tools.AlertCreate(r.Context(), h.stores.Alert, args)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

// --- GET /api/v1/alerts ---

func (h *handler) alertList(w http.ResponseWriter, r *http.Request) {
	args := tools.AlertListArgs{
		Symbol: r.URL.Query().Get("symbol"),
	}
	alerts, err := tools.AlertList(r.Context(), h.stores.Alert, args)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, alerts)
}

// --- PUT /api/v1/alerts/{id}/acknowledge ---

func (h *handler) alertAcknowledge(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a, err := tools.AlertAcknowledge(r.Context(), h.stores.Alert, tools.AlertAcknowledgeArgs{ID: id})
	if err != nil {
		handleNotFound(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// --- DELETE /api/v1/alerts/{id} ---

func (h *handler) alertDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := tools.AlertDelete(r.Context(), h.stores.Alert, tools.AlertDeleteArgs{ID: id}); err != nil {
		handleNotFound(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}
