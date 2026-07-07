package handlers

import (
	"net/http"

	"github.com/christophernnh/curve-engine/api/types"
	"github.com/christophernnh/curve-engine/internal/risk"
)

// Risk returns DV01, modified duration, convexity, and bucketed DV01
// for a given bond priced off the current bootstrapped curve.
//
// POST /api/risk
// Body: { "bond": { "maturity": 10, "coupon_rate": 0.04, "face_value": 100 } }
func (h *Handler) Risk(w http.ResponseWriter, r *http.Request) {
	var req types.RiskRequest
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
	results := risk.Analyze(bond, c)

	// Convert bucketed DV01 map to sorted slice for deterministic JSON.
	buckets := make([]types.BucketedDV01Point, 0, len(results.BucketedDV01))
	for _, tenor := range sortedKeys(results.BucketedDV01) {
		buckets = append(buckets, types.BucketedDV01Point{
			MaturityYears: tenor,
			DV01:          results.BucketedDV01[tenor],
		})
	}

	writeJSON(w, http.StatusOK, types.RiskResponse{
		DV01:             results.DV01,
		ModifiedDuration: results.ModifiedDuration,
		Convexity:        results.Convexity,
		BucketedDV01:     buckets,
	})
}