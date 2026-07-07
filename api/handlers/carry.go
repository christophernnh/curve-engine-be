package handlers

import (
	"net/http"

	"github.com/christophernnh/curve-engine/api/types"
	"github.com/christophernnh/curve-engine/internal/carry"
	"github.com/christophernnh/curve-engine/internal/risk"
)

// Carry returns carry, rolldown, total, and breakeven basis points
// for a bond over the given horizon.
//
// POST /api/carry
// Body: { "bond": {...}, "horizon_months": 3 }
func (h *Handler) Carry(w http.ResponseWriter, r *http.Request) {
	var req types.CarryRequest
	if !decode(w, r, &req) {
		return
	}

	if req.Bond.Maturity <= 0 || req.Bond.FaceValue <= 0 {
		writeError(w, http.StatusBadRequest, "maturity and face_value must be positive")
		return
	}
	if req.HorizonMonths <= 0 {
		writeError(w, http.StatusBadRequest, "horizon_months must be positive")
		return
	}

	c := h.getCurve(w)
	if c == nil {
		return
	}

	bond := bondFromRequest(req.Bond)
	horizonYears := req.HorizonMonths / 12.0

	// DV01 is needed for the breakeven calculation inside carry.Analyze.
	riskResults := risk.Analyze(bond, c)
	carryResults := carry.Analyze(bond, c, horizonYears, riskResults.DV01)

	writeJSON(w, http.StatusOK, types.CarryResponse{
		HorizonMonths: req.HorizonMonths,
		Carry:         carryResults.Carry,
		Rolldown:      carryResults.Rolldown,
		Total:         carryResults.Total,
		BreakevenBps:  carryResults.BreakevenBps,
	})
}