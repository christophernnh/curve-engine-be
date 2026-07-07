package handlers

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/christophernnh/curve-engine/api/types"
)

// GetCurve returns the current bootstrapped curve: raw pillars,
// interpolated display rates, and the source date from Treasury.gov.
//
// GET /api/curve
func (h *Handler) GetCurve(w http.ResponseWriter, r *http.Request) {
	c, parYields, sourceDate, err := h.cache.Get()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable,
			"curve unavailable: "+err.Error())
		return
	}

	// ---- Raw bootstrapped pillars ----
	pillarTimes := c.PillarTimes()
	pillars := make([]types.PillarPoint, len(pillarTimes))
	for i, t := range pillarTimes {
		pillars[i] = types.PillarPoint{
			MaturityYears:  t,
			ZeroRate:       c.ZeroRate(t),
			DiscountFactor: c.DiscountFactor(t),
			ParYield:       parYields[t],
		}
	}

	// ---- Interpolated display curve ----
	// 50 evenly spaced points from 1M to 30Y for smooth chart rendering.
	displayTenors := make([]float64, 0, 50)
	for i := 1; i <= 50; i++ {
		displayTenors = append(displayTenors, float64(i)*30.0/50.0)
	}
	rates := make([]types.TenorRate, len(displayTenors))
	for i, t := range displayTenors {
		rates[i] = types.TenorRate{
			MaturityYears:  t,
			ZeroRate:       c.ZeroRate(t),
			DiscountFactor: c.DiscountFactor(t),
		}
	}

	writeJSON(w, http.StatusOK, types.CurveResponse{
		Pillars:       pillars,
		Rates:         rates,
		Interpolation: "log_linear_df",
		SourceDate:    sourceDate,
	})
}

// GetForwardRate returns the implied forward rate between two tenors.
//
// GET /api/curve/forward?t1=5&t2=10
func (h *Handler) GetForwardRate(w http.ResponseWriter, r *http.Request) {
	t1Str := r.URL.Query().Get("t1")
	t2Str := r.URL.Query().Get("t2")

	if t1Str == "" || t2Str == "" {
		writeError(w, http.StatusBadRequest, "t1 and t2 query params required (years)")
		return
	}

	t1, err := strconv.ParseFloat(t1Str, 64)
	if err != nil || t1 <= 0 {
		writeError(w, http.StatusBadRequest, "t1 must be a positive number")
		return
	}
	t2, err := strconv.ParseFloat(t2Str, 64)
	if err != nil || t2 <= t1 {
		writeError(w, http.StatusBadRequest, "t2 must be greater than t1")
		return
	}

	c := h.getCurve(w)
	if c == nil {
		return
	}

	writeJSON(w, http.StatusOK, types.ForwardRateResponse{
		T1:          t1,
		T2:          t2,
		ForwardRate: c.ForwardRate(t1, t2),
	})
}

// sortedKeys returns map keys in ascending order -- used to produce
// deterministic JSON output for bucketed DV01 arrays.
func sortedKeys(m map[float64]float64) []float64 {
	keys := make([]float64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Float64s(keys)
	return keys
}