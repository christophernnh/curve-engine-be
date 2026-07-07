package handlers

import (
	"net/http"

	"github.com/christophernnh/curve-engine/api/types"
	"github.com/christophernnh/curve-engine/internal/pricing"
)

// Price returns NPV, price percentage, YTM, and the full cashflow
// schedule with present values for a given bond.
//
// POST /api/price
// Body: { "bond": { "maturity": 10, "coupon_rate": 0.04, "face_value": 100 } }
func (h *Handler) Price(w http.ResponseWriter, r *http.Request) {
	var req types.PriceRequest
	if !decode(w, r, &req) {
		return
	}

	if req.Bond.Maturity <= 0 || req.Bond.FaceValue <= 0 {
		writeError(w, http.StatusBadRequest, "maturity and face_value must be positive")
		return
	}

	c := h.getCurve(w)
	if c == nil {
		return
	}

	bond := bondFromRequest(req.Bond)
	npv := pricing.NPV(bond, c)
	pricePct := pricing.Price(bond, c)

	ytm, err := pricing.YTM(bond, c)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "YTM failed to converge: "+err.Error())
		return
	}

	// Build cashflow schedule with present values.
	schedule := bond.Schedule()
	cashflows := make([]types.CashflowPoint, len(schedule))
	for i, cf := range schedule {
		cashflows[i] = types.CashflowPoint{
			TimeYears:    cf.Time,
			Amount:       cf.Amount,
			PresentValue: cf.Amount * c.DiscountFactor(cf.Time),
		}
	}

	writeJSON(w, http.StatusOK, types.PriceResponse{
		NPV:       npv,
		PricePct:  pricePct,
		YTM:       ytm,
		Cashflows: cashflows,
	})
}