package handlers

import (
	"net/http"

	"github.com/christophernnh/curve-engine/api/types"
	"github.com/christophernnh/curve-engine/internal/curve"
	pnlpkg "github.com/christophernnh/curve-engine/internal/pnl"
)

// PnL returns P&L attribution for a simulated parallel rate shift.
// The shift simulates: "what if overnight, all rates moved by X bps?"
//
// POST /api/pnl
// Body: { "bond": {...}, "rate_shift_bps": 5 }
func (h *Handler) PnL(w http.ResponseWriter, r *http.Request) {
	var req types.PnLRequest
	if !decode(w, r, &req) {
		return
	}

	if req.Bond.Maturity <= 0 || req.Bond.FaceValue <= 0 {
		writeError(w, http.StatusBadRequest, "maturity and face_value must be positive")
		return
	}

	todayCurve := h.getCurve(w)
	if todayCurve == nil {
		return
	}

	// Build yesterday's curve: today's zero rates minus the shift.
	// This lets the caller ask "what was my P&L if rates moved X bps?"
	shiftDecimal := req.RateShiftBps / 10000.0
	pillarTimes := todayCurve.PillarTimes()
	yesterdayRates := make([]float64, len(pillarTimes))
	for i, t := range pillarTimes {
		yesterdayRates[i] = todayCurve.ZeroRate(t) - shiftDecimal
	}
	yesterdayCurve := curve.NewDiscountCurve(
		pillarTimes,
		yesterdayRates,
		curve.LogLinearDFInterpolator{},
	)

	bond := bondFromRequest(req.Bond)
	oneDay := 1.0 / 252.0
	attribution := pnlpkg.Attribute(bond, yesterdayCurve, todayCurve, oneDay)

	writeJSON(w, http.StatusOK, types.PnLResponse{
		RateShiftBps: req.RateShiftBps,
		ActualPnL:    attribution.ActualPnL,
		DV01PnL:      attribution.DV01PnL,
		ConvexityPnL: attribution.ConvexityPnL,
		ExplainedPnL: attribution.DV01PnL + attribution.ConvexityPnL,
		Residual:     attribution.Residual,
		CarryPnL:     attribution.CarryPnL,
	})
}
