package handlers

import (
	"net/http"

	"github.com/christophernnh/curve-engine/api/types"
	hedgepkg "github.com/christophernnh/curve-engine/internal/hedge"
)

// Hedge returns the hedge ratio, notional, and residual DV01 buckets
// for hedging a position with a chosen instrument.
//
// POST /api/hedge
// Body: {
//   "position": { "maturity": 10, "coupon_rate": 0.04, "face_value": 100 },
//   "hedge_instrument": { "maturity": 10, "coupon_rate": 0.0448, "face_value": 100 },
//   "position_face": 10000000
// }
func (h *Handler) Hedge(w http.ResponseWriter, r *http.Request) {
	var req types.HedgeRequest
	if !decode(w, r, &req) {
		return
	}

	if req.Position.Maturity <= 0 || req.Position.FaceValue <= 0 {
		writeError(w, http.StatusBadRequest, "position maturity and face_value must be positive")
		return
	}
	if req.HedgeInstrument.Maturity <= 0 || req.HedgeInstrument.FaceValue <= 0 {
		writeError(w, http.StatusBadRequest, "hedge_instrument maturity and face_value must be positive")
		return
	}
	if req.PositionFace <= 0 {
		writeError(w, http.StatusBadRequest, "position_face must be positive")
		return
	}

	c := h.getCurve(w)
	if c == nil {
		return
	}

	position := bondFromRequest(req.Position)
	hedgeInstrument := bondFromRequest(req.HedgeInstrument)

	result, notional, err := hedgepkg.ComputeNotional(
		position, hedgeInstrument, c, req.PositionFace,
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Convert residual DV01 map to sorted slice.
	residuals := make([]types.BucketedDV01Point, 0, len(result.ResidualDV01))
	for _, tenor := range sortedKeys(result.ResidualDV01) {
		residuals = append(residuals, types.BucketedDV01Point{
			MaturityYears: tenor,
			DV01:          result.ResidualDV01[tenor],
		})
	}

	writeJSON(w, http.StatusOK, types.HedgeResponse{
		HedgeRatio:        result.HedgeRatio,
		HedgeNotional:     notional,
		PositionDV01:      result.PositionDV01,
		HedgeDV01PerUnit:  result.HedgeDV01PerUnit,
		ConvexityMismatch: result.ConvexityMismatch,
		ResidualDV01:      residuals,
		TotalResidualDV01: result.TotalResidualDV01,
	})
}