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

package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/buildinfo"
	"github.com/tdrn-org/adjudex-mcp/internal/data"
	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/adjudex-mcp/internal/marketdata"
	"github.com/tdrn-org/go-finance"
	"github.com/tdrn-org/go-httpserver"
)

type Runtime interface {
	BaseURL() *url.URL
	Logger() *slog.Logger
	DataStore() *data.Store
	MarketDataProvider() *marketdata.Provider
	DisplayCurrency() finance.Currency
	Ping(ctx context.Context) error
}

//	@title			ADJUDEX MCP Server REST API
//	@version		1.0
//	@description	MCP server providing Agent access to virtual stock depot.

//	@contact.url	https://github.com/tdrn-org/adjudex-mcp

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:9123
//	@BasePath	/api/v1

type API struct {
	runtime Runtime
}

func NewAPI(runtime Runtime) *API {
	return &API{
		runtime: runtime,
	}
}

const basePath string = "/api/v1"
const PathPing string = basePath + "/ping"

func (api *API) Mount(server *httpserver.Instance) {
	server.HandleFunc("GET "+PathPing, api.PingGet)
	server.HandleFunc("GET "+basePath+"/info", api.InfoGet)

	server.HandleFunc("GET "+basePath+"/portfolios", api.PortfolioList)
	server.HandleFunc("POST "+basePath+"/portfolios", api.PortfolioCreate)
	server.HandleFunc("GET "+basePath+"/portfolios/summaries", api.PortfolioSummaries)
	server.HandleFunc("GET "+basePath+"/portfolios/{id}", api.PortfolioGet)
	server.HandleFunc("DELETE "+basePath+"/portfolios/{id}", api.PortfolioDelete)
	server.HandleFunc("GET "+basePath+"/portfolios/{id}/holdings", api.PortfolioHoldings)

	server.HandleFunc("POST "+basePath+"/portfolios/{id}/positions", api.PositionAdd)
	server.HandleFunc("PUT "+basePath+"/portfolios/{id}/positions/{posId}", api.PositionUpdate)
	server.HandleFunc("DELETE "+basePath+"/portfolios/{id}/positions/{posId}", api.PositionRemove)

	server.HandleFunc("GET "+basePath+"/quotes/{listing}", api.QuoteGet)
	server.HandleFunc("GET "+basePath+"/quotes/{listing}/history", api.QuoteHistory)

	server.HandleFunc("GET "+basePath+"/rates", api.RatesGet)
}

const responseOK string = "ok"
const responseServerError string = "server error"

// GET @BasePath/ping
//
//	@Summary		Ping the server
//	@Description	Ping the server to check general health
//	@Produce		text/plain
//	@Success		200	{string}	string	"ok"
//	@Failure		500	{string}	string	"server error"
//	@Router			/api/v1/ping [get]
func (api *API) PingGet(w http.ResponseWriter, r *http.Request) {
	err := api.runtime.Ping(r.Context())
	if err != nil {
		api.sendError(w, r, http.StatusInternalServerError, err)
		return
	}
	api.sendPlainTextResponse(w, r, http.StatusOK, responseOK)
}

// GET @BasePath/info
//
//	@Summary		Query server info
//	@Description	Get server software version details.
//	@Produce		json
//	@Success		200	{object}	ServerInfo
//	@Failure		500	{string}	string	"server error"
//	@Router			/api/v1/info [get]
func (api *API) InfoGet(w http.ResponseWriter, r *http.Request) {
	info := &serverInfo{
		Version:        buildinfo.Version(),
		FinanceVersion: buildinfo.FinanceVersion(),
	}
	api.sendJSON(w, r, http.StatusOK, info)
}

type serverInfo struct {
	Version        string `json:"version"`
	FinanceVersion string `json:"finance_version"`
}

// =============================================================================
// Portfolio Handlers
// =============================================================================

// GET @BasePath/portfolios
//
//	@Summary		List all portfolios
//	@Description	Returns all portfolios without positions for performance.
//	@Produce		json
//	@Success		200	{array}		domain.Portfolio
//	@Failure		500	{string}	string	"server error"
//	@Router			/api/v1/portfolios [get]
func (api *API) PortfolioList(w http.ResponseWriter, r *http.Request) {
	ps, err := api.runtime.DataStore().ListPortfolios(r.Context())
	if err != nil {
		api.sendError(w, r, http.StatusInternalServerError, fmt.Errorf("listing portfolios: %w", err))
		return
	}
	api.sendJSON(w, r, http.StatusOK, ps)
}

// GET @BasePath/portfolios/{id}
//
//	@Summary		Get portfolio by ID
//	@Description	Returns a single portfolio with all positions.
//	@Produce		json
//	@Param			id	path		string	true	"Portfolio ID"
//	@Success		200	{object}	domain.Portfolio
//	@Failure		404	{string}	string	"not found"
//	@Router			/api/v1/portfolios/{id} [get]
func (api *API) PortfolioGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := api.runtime.DataStore().GetPortfolio(r.Context(), id)
	if err != nil {
		api.sendError(w, r, http.StatusNotFound, fmt.Errorf("getting portfolio %q: %w", id, err))
		return
	}
	api.sendJSON(w, r, http.StatusOK, p)
}

// POST @BasePath/portfolios
//
//	@Summary		Create a new portfolio
//	@Description	Creates a portfolio and returns it with its ID.
//	@Accept			json
//	@Produce		json
//	@Param			portfolio	body		portfolioCreateRequest	true	"Portfolio data"
//	@Success		201			{object}	domain.Portfolio
//	@Failure		400			{string}	string	"bad request"
//	@Router			/api/v1/portfolios [post]
func (api *API) PortfolioCreate(w http.ResponseWriter, r *http.Request) {
	var req portfolioCreateRequest
	if err := api.decodeBody(r, &req); err != nil {
		api.sendError(w, r, http.StatusBadRequest, err)
		return
	}
	p := &domain.Portfolio{
		Name:        req.Name,
		Description: req.Description,
	}
	if err := api.runtime.DataStore().CreatePortfolio(r.Context(), p); err != nil {
		api.sendError(w, r, http.StatusInternalServerError, fmt.Errorf("creating portfolio: %w", err))
		return
	}
	api.sendJSON(w, r, http.StatusCreated, p)
}

type portfolioCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// DELETE @BasePath/portfolios/{id}
//
//	@Summary		Delete a portfolio
//	@Description	Deletes a portfolio and all its positions.
//	@Param			id	path		string	true	"Portfolio ID"
//	@Success		200	{object}	map[string]string
//	@Failure		404	{string}	string	"not found"
//	@Router			/api/v1/portfolios/{id} [delete]
func (api *API) PortfolioDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := api.runtime.DataStore().DeletePortfolio(r.Context(), id); err != nil {
		api.sendError(w, r, http.StatusNotFound, fmt.Errorf("deleting portfolio %q: %w", id, err))
		return
	}
	api.sendJSON(w, r, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

// =============================================================================
// Portfolio Summary Handler
// =============================================================================

type PortfolioSummary struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Currency      finance.Currency       `json:"currency"`
	MarketValue   float64                `json:"market_value"`
	PnL           float64                `json:"pnl"`
	PnLPercent    float64                `json:"pnl_percent"`
	PositionCount int                    `json:"position_count"`
	PricedCount   int                    `json:"priced_count"`
	Rates         []finance.ExchangeRate `json:"rates,omitempty"`
}

// displayCurrency returns the currency a request asks values to be presented in.
// An explicit ?currency= wins, otherwise the configured display currency is used.
func (api *API) displayCurrency(r *http.Request) finance.Currency {
	if currency := r.URL.Query().Get("currency"); currency != "" {
		return finance.Currency(strings.ToUpper(currency))
	}
	return api.runtime.DisplayCurrency()
}

// ErrInvalidListing indicates a listing that could not be read from the request.
var ErrInvalidListing error = errors.New("invalid listing")

// listingFromPath reads the listing ID passed as a path segment.
//
// The ID identifies an instrument; it is resolved against the store, which is
// where the description (instrument name, type) comes from and which proves that
// the server issued this listing at all. A listing that only arrives as an ID is
// never enough on its own - see [finance.Listing.MarshalJSON].
func (api *API) listingFromPath(r *http.Request) (finance.Listing, error) {
	var unset finance.Listing
	raw := r.PathValue("listing")
	if raw == "" {
		return unset, fmt.Errorf("no listing in path, expected /quotes/{listing}")
	}
	listing, err := finance.ParseListingID(raw)
	if err != nil {
		// Both wraps: the sentinel makes the failure safe to report, the cause
		// carries the detail. Two %w keep the message free of duplication.
		return unset, fmt.Errorf("%w: %w", ErrInvalidListing, err)
	}
	resolved, err := api.runtime.DataStore().ResolveListing(r.Context(), listing)
	if err != nil {
		return unset, err
	}
	return resolved, nil
}

// sendListingError reports a listing that could not be read from the request:
// malformed is the caller's mistake, unknown is a stale or foreign ID.
func (api *API) sendListingError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, data.ErrUnknownListing) {
		status = http.StatusNotFound
	}
	api.sendError(w, r, status, err)
}

// GET @BasePath/rates
//
//	@Summary		Exchange rates
//	@Description	Returns the exchange rates converting the given base currencies into the target currency. Quotes are always delivered in the currency their provider reports; clients use these rates to present them in their own currency.
//	@Produce		json
//	@Param			target	query		string		false	"Target currency (e.g. EUR). Defaults to the configured display currency."
//	@Param			base	query		[]string	false	"Base currencies to convert from (repeatable, e.g. ?base=USD&base=CHF)."
//	@Success		200		{array}		finance.ExchangeRate
//	@Failure		503		{string}	string	"rate not available"
//	@Router			/api/v1/rates [get]
func (api *API) RatesGet(w http.ResponseWriter, r *http.Request) {
	target := finance.Currency(strings.ToUpper(r.URL.Query().Get("target")))
	if target == "" {
		target = api.runtime.DisplayCurrency()
	}
	rawBases := r.URL.Query()["base"]
	bases := make([]finance.Currency, 0, len(rawBases))
	for _, raw := range rawBases {
		bases = append(bases, finance.Currency(strings.ToUpper(raw)))
	}
	rates, err := api.runtime.MarketDataProvider().QueryExchangeRates(r.Context(), target, bases)
	if err != nil {
		api.sendError(w, r, http.StatusServiceUnavailable, err)
		return
	}
	api.sendJSON(w, r, http.StatusOK, rates)
}

// GET @BasePath/portfolios/summaries
func (api *API) PortfolioSummaries(w http.ResponseWriter, r *http.Request) {
	ps, err := api.runtime.DataStore().ListPortfolios(r.Context())
	if err != nil {
		api.sendError(w, r, http.StatusInternalServerError, fmt.Errorf("listing portfolios: %w", err))
		return
	}

	summaries := make([]PortfolioSummary, 0, len(ps))
	for _, p := range ps {
		full, err := api.runtime.DataStore().GetPortfolio(r.Context(), p.ID)
		if err != nil {
			api.runtime.Logger().Warn("failed to get portfolio for summary", "id", p.ID, "err", err)
			continue
		}

		target := api.displayCurrency(r)
		holdings := make([]domain.Holding, 0, len(full.Positions))
		bases := make([]finance.Currency, 0, 2*len(full.Positions))
		for _, pos := range full.Positions {
			quote, err := api.runtime.MarketDataProvider().QueryQuote(r.Context(), pos.Listing)
			if err != nil {
				api.runtime.Logger().Warn("failed to resolve quote for summary", "listing", pos.Listing.ID(), "err", err)
				continue
			}
			holdings = append(holdings, domain.NewHolding(pos, quote.ListingPrice))
			bases = append(bases, pos.Currency, quote.Currency)
		}

		rates, err := api.runtime.MarketDataProvider().QueryExchangeRates(r.Context(), target, bases)
		if err != nil {
			api.sendError(w, r, http.StatusServiceUnavailable, fmt.Errorf("portfolio %s: %w", p.ID, err))
			return
		}
		summary, err := domain.Holdings(holdings).Summarize(target, rates)
		if err != nil {
			api.sendError(w, r, http.StatusServiceUnavailable, fmt.Errorf("portfolio %s: %w", p.ID, err))
			return
		}
		summaries = append(summaries, PortfolioSummary{
			ID:            p.ID,
			Name:          p.Name,
			Currency:      summary.Currency,
			MarketValue:   summary.MarketValue,
			PnL:           summary.PnL,
			PnLPercent:    summary.PnLPercent,
			PositionCount: len(full.Positions),
			PricedCount:   len(holdings),
			Rates:         summary.Rates,
		})
	}

	api.sendJSON(w, r, http.StatusOK, summaries)
}

// GET @BasePath/portfolios/{id}/holdings
//
//	@Summary		Get portfolio holdings with live PnL
//	@Description	Returns the portfolio with all positions enriched with current market data.
//	@Produce		json
//	@Param			id	path		string	true	"Portfolio ID"
//	@Success		200	{array}		domain.Holding
//	@Failure		404	{string}	string	"not found"
//	@Router			/api/v1/portfolios/{id}/holdings [get]
func (api *API) PortfolioHoldings(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := api.runtime.DataStore().GetPortfolio(r.Context(), id)
	if err != nil {
		api.sendError(w, r, http.StatusNotFound, fmt.Errorf("getting portfolio %q: %w", id, err))
		return
	}

	target := api.displayCurrency(r)
	holdings := make([]domain.Holding, 0, len(p.Positions))
	bases := make([]finance.Currency, 0, 2*len(p.Positions))
	for _, pos := range p.Positions {
		quote, err := api.runtime.MarketDataProvider().QueryQuote(r.Context(), pos.Listing)
		if err != nil {
			api.runtime.Logger().Warn("failed to resolve quote for holding", "listing", pos.Listing.ID(), "err", err)
			continue
		}
		holdings = append(holdings, domain.NewHolding(pos, quote.ListingPrice))
		bases = append(bases, pos.Currency, quote.Currency)
	}
	rates, err := api.runtime.MarketDataProvider().QueryExchangeRates(r.Context(), target, bases)
	if err != nil {
		api.sendError(w, r, http.StatusServiceUnavailable, fmt.Errorf("portfolio %s: %w", id, err))
		return
	}
	converted, err := domain.Holdings(holdings).ConvertedTo(target, rates)
	if err != nil {
		api.sendError(w, r, http.StatusServiceUnavailable, fmt.Errorf("portfolio %s: %w", id, err))
		return
	}
	api.sendJSON(w, r, http.StatusOK, converted)
}

// POST @BasePath/portfolios/{id}/positions
//
//	@Summary		Add a position to a portfolio
//	@Description	Adds a new position and returns it with its ID.
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string				true	"Portfolio ID"
//	@Param			position	body		positionAddRequest	true	"Position data"
//	@Success		201			{object}	domain.Position
//	@Failure		400			{string}	string	"bad request"
//	@Router			/api/v1/portfolios/{id}/positions [post]
func (api *API) PositionAdd(w http.ResponseWriter, r *http.Request) {
	portfolioID := r.PathValue("id")
	var req positionAddRequest
	if err := api.decodeBody(r, &req); err != nil {
		api.sendError(w, r, http.StatusBadRequest, err)
		return
	}
	var listing finance.Listing
	err := json.Unmarshal([]byte(req.Listing), &listing)
	if err != nil {
		api.sendError(w, r, http.StatusBadRequest, err)
		return
	}

	entryDate, err := time.Parse(time.RFC3339, req.EntryDate)
	if err != nil {
		api.sendError(w, r, http.StatusBadRequest, fmt.Errorf("parsing entry_date: %w", err))
		return
	}

	// The entry price is only meaningful together with its currency, and that
	// currency cannot be derived: a German broker books a US stock in EUR. There is
	// deliberately no default here — a silent USD default would corrupt the PnL.
	if req.Currency == "" {
		api.sendError(w, r, http.StatusBadRequest, fmt.Errorf("currency is required: the entry price currency cannot be derived"))
		return
	}
	currency := req.Currency

	pos := &domain.Position{
		Listing:    listing,
		Currency:   currency,
		Quantity:   req.Quantity,
		EntryPrice: req.EntryPrice,
		EntryDate:  entryDate,
		Notes:      req.Notes,
	}
	if err := api.runtime.DataStore().AddPosition(r.Context(), portfolioID, pos); err != nil {
		api.sendError(w, r, http.StatusInternalServerError, fmt.Errorf("adding position: %w", err))
		return
	}
	api.sendJSON(w, r, http.StatusCreated, pos)
}

type positionAddRequest struct {
	Listing    string           `json:"listing"`
	Currency   finance.Currency `json:"currency"`
	Quantity   float64          `json:"quantity"`
	EntryPrice float64          `json:"entry_price"`
	EntryDate  string           `json:"entry_date"`
	Notes      string           `json:"notes"`
}

// PUT @BasePath/portfolios/{id}/positions/{posId}
//
//	@Summary		Update a position
//	@Description	Updates an existing position. Only provided fields are changed (PATCH semantics).
//	@Accept			json
//	@Produce		json
//	@Param			id			path		string					true	"Portfolio ID"
//	@Param			posId		path		string					true	"Position ID"
//	@Param			position	body		positionUpdateRequest	true	"Position updates"
//	@Success		200			{object}	domain.Position
//	@Failure		400			{string}	string	"bad request"
//	@Router			/api/v1/portfolios/{id}/positions/{posId} [put]
func (api *API) PositionUpdate(w http.ResponseWriter, r *http.Request) {
	portfolioID := r.PathValue("id")
	positionID := r.PathValue("posId")

	p, err := api.runtime.DataStore().GetPortfolio(r.Context(), portfolioID)
	if err != nil {
		api.sendError(w, r, http.StatusNotFound, fmt.Errorf("getting portfolio %q: %w", portfolioID, err))
		return
	}
	var pos *domain.Position
	for i := range p.Positions {
		if p.Positions[i].ID == positionID {
			pos = &p.Positions[i]
			break
		}
	}
	if pos == nil {
		api.sendError(w, r, http.StatusNotFound, fmt.Errorf("position %q not found", positionID))
		return
	}

	var req positionUpdateRequest
	if err := api.decodeBody(r, &req); err != nil {
		api.sendError(w, r, http.StatusBadRequest, err)
		return
	}
	if req.Quantity != nil {
		pos.Quantity = *req.Quantity
	}
	// The entry price currency is a booked fact: changing it later would silently
	// rewrite the position's performance history.
	if req.Currency != nil {
		api.sendError(w, r, http.StatusBadRequest, fmt.Errorf("position currency cannot be changed after booking (current %s)", pos.Currency))
		return
	}
	if req.EntryPrice != nil {
		pos.EntryPrice = *req.EntryPrice
	}
	if req.Notes != nil {
		pos.Notes = *req.Notes
	}

	if err := api.runtime.DataStore().UpdatePosition(r.Context(), portfolioID, pos); err != nil {
		api.sendError(w, r, http.StatusInternalServerError, fmt.Errorf("updating position: %w", err))
		return
	}
	api.sendJSON(w, r, http.StatusOK, pos)
}

type positionUpdateRequest struct {
	Currency   *finance.Currency `json:"currency"`
	Quantity   *float64          `json:"quantity"`
	EntryPrice *float64          `json:"entry_price"`
	Notes      *string           `json:"notes"`
}

// DELETE @BasePath/portfolios/{id}/positions/{posId}
//
//	@Summary		Remove a position
//	@Description	Removes a position from a portfolio.
//	@Param			id		path		string	true	"Portfolio ID"
//	@Param			posId	path		string	true	"Position ID"
//	@Success		200		{object}	map[string]string
//	@Failure		404		{string}	string	"not found"
//	@Router			/api/v1/portfolios/{id}/positions/{posId} [delete]
func (api *API) PositionRemove(w http.ResponseWriter, r *http.Request) {
	portfolioID := r.PathValue("id")
	positionID := r.PathValue("posId")
	if err := api.runtime.DataStore().RemovePosition(r.Context(), portfolioID, positionID); err != nil {
		api.sendError(w, r, http.StatusNotFound, fmt.Errorf("removing position %q: %w", positionID, err))
		return
	}
	api.sendJSON(w, r, http.StatusOK, map[string]string{"status": "removed", "id": positionID})
}

// =============================================================================
// Quote Handlers
// =============================================================================

// GET @BasePath/quotes/{listing}
//
//	@Summary		Get latest quote
//	@Description	Returns the latest quote for a symbol (cached or live).
//	@Produce		json
//	@Param			listing	path		string	true	"Listing"
//	@Success		200		{object}	domain.Quote
//	@Failure		404		{string}	string	"not found"
//	@Router			/api/v1/quotes/{listing} [get]
func (api *API) QuoteGet(w http.ResponseWriter, r *http.Request) {
	listing, err := api.listingFromPath(r)
	if err != nil {
		api.sendListingError(w, r, err)
		return
	}

	q, err := api.runtime.DataStore().GetLatestQuote(r.Context(), listing)
	if err != nil {
		api.sendError(w, r, http.StatusNotFound, fmt.Errorf("resolving quote for %q: %w", listing.String(), err))
		return
	}
	api.sendJSON(w, r, http.StatusOK, q)
}

// GET @BasePath/quotes/{listing}/history
//
//	@Summary		Get quote history
//	@Description	Returns historical quotes for a symbol within a date range.
//	@Produce		json
//	@Param			listing	path		string	true	"Listing"
//	@Param			from	query		string	false	"Start date (RFC3339)"
//	@Param			to		query		string	false	"End date (RFC3339)"
//	@Success		200		{object}	domain.PriceHistory
//	@Failure		404		{string}	string	"not found"
//	@Router			/api/v1/quotes/{symbol}/history [get]
func (api *API) QuoteHistory(w http.ResponseWriter, r *http.Request) {
	listing, err := api.listingFromPath(r)
	if err != nil {
		api.sendListingError(w, r, err)
		return
	}
	now := time.Now()
	to := now
	from := now.AddDate(0, -1, 0)

	if f := r.URL.Query().Get("from"); f != "" {
		var err error
		from, err = time.Parse(time.RFC3339, f)
		if err != nil {
			api.sendError(w, r, http.StatusBadRequest, fmt.Errorf("parsing from: %w", err))
			return
		}
	}
	if t := r.URL.Query().Get("to"); t != "" {
		var err error
		to, err = time.Parse(time.RFC3339, t)
		if err != nil {
			api.sendError(w, r, http.StatusBadRequest, fmt.Errorf("parsing to: %w", err))
			return
		}
	}

	quotes, err := api.runtime.DataStore().GetQuotes(r.Context(), listing, from, to)
	if err != nil || len(quotes) == 0 {
		api.sendError(w, r, http.StatusNotFound, fmt.Errorf("fetching history for %q: %w", listing.String(), err))
	}

	api.sendJSON(w, r, http.StatusOK, domain.PriceHistory{Listing: listing, Quotes: quotes})
}

// =============================================================================
// Helpers
// =============================================================================

func (api *API) sendJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		api.runtime.Logger().Error("failed to encode JSON response", slog.String("path", r.URL.Path), slog.Any("err", err))
	}
}

func (api *API) decodeBody(r *http.Request, v any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("reading body: %w", err)
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}
	return nil
}

func (api *API) sendError(w http.ResponseWriter, r *http.Request, status int, cause error) {
	if cause != nil {
		api.runtime.Logger().Error("http handler failure", slog.String("path", r.URL.Path), slog.String("method", r.Method), slog.Any("err", cause))
	}
	// A caller that only reads "server error" cannot tell a typo in a listing
	// from a stale one, so failures that describe the request itself are passed
	// through. Everything else stays opaque: driver errors, provider responses
	// and stack details never leave the server.
	if message, ok := safeErrorMessage(cause); ok {
		http.Error(w, message, status)
		return
	}
	http.Error(w, responseServerError, status)
}

// safeErrorMessage returns the cause's message if it is safe to hand to a client.
//
// The whitelist is deliberate: a new error type has to be added here to become
// visible, so nothing leaks by accident.
func safeErrorMessage(cause error) (string, bool) {
	switch {
	case cause == nil:
		return "", false
	case errors.Is(cause, ErrInvalidListing), errors.Is(cause, data.ErrUnknownListing):
		return cause.Error(), true
	default:
		return "", false
	}
}

func (api *API) sendPlainTextResponse(w http.ResponseWriter, r *http.Request, status int, content string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	_, err := w.Write([]byte(content))
	if err != nil {
		api.runtime.Logger().Error("failed to send 'text/plain' response", slog.String("path", r.URL.Path), slog.String("method", r.Method), slog.Any("err", err))
	}
}
