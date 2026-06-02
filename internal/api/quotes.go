package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/tdrn-org/adjudex-mcp/internal/domain"
	"github.com/tdrn-org/adjudex-mcp/internal/mcp/tools"
)

// --- GET /api/v1/quotes/{symbol} ---

func (h *handler) quoteGet(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	q, err := tools.QuoteGet(r.Context(), h.stores.Quote, tools.QuoteGetArgs{Symbol: symbol})
	if err != nil {
		// Fallback: fetch from provider and save
		q2, err2 := h.provider.FetchQuote(r.Context(), symbol)
		if err2 != nil {
			writeError(w, http.StatusNotFound, "no quote found for "+symbol)
			return
		}
		h.stores.Quote.SaveQuotes(r.Context(), []domain.Quote{*q2})
		writeJSON(w, http.StatusOK, q2)
		return
	}
	writeJSON(w, http.StatusOK, q)
}

// --- GET /api/v1/quotes/{symbol}/history ---

func (h *handler) quoteHistory(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" {
		from = time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
	}
	if to == "" {
		to = time.Now().Format(time.RFC3339)
	}
	args := tools.QuoteHistoryArgs{
		Symbol: symbol,
		From:   from,
		To:     to,
	}
	quotes, err := tools.QuoteHistory(r.Context(), h.stores.Quote, args)
	if err != nil || len(quotes) == 0 {
		// Fallback: fetch from provider and save
		fromTime, _ := time.Parse(time.RFC3339, from)
		toTime, _ := time.Parse(time.RFC3339, to)
		q2, err2 := h.provider.FetchHistory(r.Context(), symbol, fromTime, toTime)
		if err2 != nil {
			writeError(w, http.StatusBadRequest, err2.Error())
			return
		}
		if len(q2) > 0 {
			h.stores.Quote.SaveQuotes(r.Context(), q2)
		}
		writeJSON(w, http.StatusOK, domain.PriceHistory{Symbol: symbol, Quotes: q2})
		return
	}
	writeJSON(w, http.StatusOK, domain.PriceHistory{Symbol: symbol, Quotes: quotes})
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
