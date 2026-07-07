// HTTP handlers
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/christophernnh/curve-engine/api/types"
	"github.com/christophernnh/curve-engine/internal/curve"
	"github.com/christophernnh/curve-engine/internal/pricing"
)

// Cache is the interface the handler needs from the curve cache --
// keeps the handler decoupled from the concrete CurveCache type.
type Cache interface {
	Get() (*curve.DiscountCurve, map[float64]float64, string, error)
}

// Handler holds the shared dependencies for all HTTP handlers.
type Handler struct {
	cache Cache
}

// New constructs a Handler with the provided curve cache.
func New(cache Cache) *Handler {
	return &Handler{cache: cache}
}

// writeJSON encodes v as JSON and writes it to w with the given status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError writes a structured JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, types.ErrorResponse{Error: msg})
}

// decode reads and decodes a JSON request body into v.
// Returns false and writes a 400 response if decoding fails.
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return false
	}
	return true
}

// bondFromRequest converts a BondRequest JSON struct into a pricing.Bond
// domain object. Credit spread is converted from bps to decimal here.
func bondFromRequest(req types.BondRequest) pricing.Bond {
	if req.CreditSpreadBps > 0 {
		return pricing.NewCorporateBond(
			req.Maturity,
			req.CouponRate,
			req.FaceValue,
			req.CreditSpreadBps,
		)
	}
	return pricing.NewTreasuryBond(req.Maturity, req.CouponRate, req.FaceValue)
}

// getCurve fetches the current curve from the cache, writing an error
// response and returning nil on failure.
func (h *Handler) getCurve(w http.ResponseWriter) *curve.DiscountCurve {
	c, _, _, err := h.cache.Get()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable,
			"curve unavailable: "+err.Error())
		return nil
	}
	return c
}