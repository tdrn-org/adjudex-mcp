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
	"errors"
	"net/http"
	"strconv"

	"github.com/tdrn-org/adjudex-mcp/internal/mcp/tools"
)

// --- POST /api/v1/portfolios ---

func (h *handler) portfolioCreate(w http.ResponseWriter, r *http.Request) {
	var args tools.PortfolioCreateArgs
	if err := decodeBody(r, &args); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	p, err := tools.PortfolioCreate(r.Context(), h.stores.Portfolio, args)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// --- GET /api/v1/portfolios ---

func (h *handler) portfolioList(w http.ResponseWriter, r *http.Request) {
	portfolios, err := tools.PortfolioList(r.Context(), h.stores.Portfolio, tools.PortfolioListArgs{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, portfolios)
}

// --- GET /api/v1/portfolios/{id} ---

func (h *handler) portfolioGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := tools.PortfolioGet(r.Context(), h.stores.Portfolio, tools.PortfolioGetArgs{ID: id})
	if err != nil {
		handleNotFound(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// --- DELETE /api/v1/portfolios/{id} ---

func (h *handler) portfolioDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := tools.PortfolioDelete(r.Context(), h.stores.Portfolio, tools.PortfolioDeleteArgs{ID: id}); err != nil {
		handleNotFound(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

// --- POST /api/v1/portfolios/{id}/positions ---

func (h *handler) portfolioAddPosition(w http.ResponseWriter, r *http.Request) {
	portfolioID := r.PathValue("id")
	var args tools.PortfolioAddPositionArgs
	if err := decodeBody(r, &args); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	args.PortfolioID = portfolioID
	pos, err := tools.PortfolioAddPosition(r.Context(), h.stores.Portfolio, args)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, pos)
}

// --- DELETE /api/v1/portfolios/{pid}/positions/{posid} ---

func (h *handler) portfolioRemovePosition(w http.ResponseWriter, r *http.Request) {
	args := tools.PortfolioRemovePositionArgs{
		PortfolioID: r.PathValue("pid"),
		PositionID:  r.PathValue("posid"),
	}
	if err := tools.PortfolioRemovePosition(r.Context(), h.stores.Portfolio, args); err != nil {
		handleNotFound(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// --- GET /api/v1/portfolios/{id}/holdings ---

func (h *handler) portfolioGetHoldings(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	holdings, err := tools.PortfolioGetHoldings(r.Context(), h.stores.Portfolio, h.stores.Quote, tools.PortfolioGetHoldingsArgs{PortfolioID: id})
	if err != nil {
		handleNotFound(w, err)
		return
	}
	writeJSON(w, http.StatusOK, holdings)
}

// --- shared helpers ---

// handleNotFound checks for not-found errors and returns 404 or 500.
func handleNotFound(w http.ResponseWriter, err error) {
	if errors.Is(err, strconv.ErrSyntax) || err.Error() == "" {
		writeError(w, http.StatusNotFound, "resource not found")
		return
	}
	writeError(w, http.StatusNotFound, err.Error())
}
