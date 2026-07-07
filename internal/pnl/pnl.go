// Uses DV01 (linear rate move), convexity (curvature), carry (time) to break down P&L attributions
package pnl

import (
	"fmt"
	"math"

	"github.com/christophernnh/curve-engine/internal/carry"
	"github.com/christophernnh/curve-engine/internal/curve"
	"github.com/christophernnh/curve-engine/internal/pricing"
	"github.com/christophernnh/curve-engine/internal/risk"
)

// Attribution holds the full P&L breakdown for one bond position
// between two curve snapshots (e.g. yesterday's close vs today's close).
type Attribution struct {
	// ActualPnL is the true total P&L: NPV(today) - NPV(yesterday).
	ActualPnL float64

	// DV01PnL is the P&L explained by the first-order (linear) rate
	// move: DV01 × rate change.
	DV01PnL float64

	// ConvexityPnL is the P&L explained by the curvature of the price/rate relationship
	// Always positive for plain bonds -- convexity helps you regardless
	// of direction. Small for tiny moves, meaningful for large ones.
	ConvexityPnL float64

	// CarryPnL is the P&L from pure time passage: one day's worth of coupons minus funding + rolldown
	CarryPnL float64

	// Residual is ActualPnL minus all explained components.
	// Ideally near zero. 
	// A large residual triggers investigation: model error, missing risk factor, or position change.
	Residual float64

	// RateChangeBps records the parallel rate shift between the two
	// curves (in basis points)
	RateChangeBps float64
}

// String returns a formatted P&L attribution report, matching the
// style of a real desk morning report.
func (a Attribution) String() string {
	return fmt.Sprintf(
		"=== P&L Attribution ===\n"+
			"Rate move:      %+.2f bps\n"+
			"Actual P&L:     %+.4f\n"+
			"  DV01 P&L:     %+.4f\n"+
			"  Convexity:    %+.4f\n"+
			"  Carry:        %+.4f\n"+
			"  --------------------\n"+
			"  Explained:    %+.4f\n"+
			"  Residual:     %+.4f  %s\n",
		a.RateChangeBps,
		a.ActualPnL,
		a.DV01PnL,
		a.ConvexityPnL,
		a.CarryPnL,
		a.DV01PnL+a.ConvexityPnL+a.CarryPnL,
		a.Residual,
		residualFlag(a.Residual, a.ActualPnL),
	)
}

// residualFlag returns a warning string 
func residualFlag(residual, actualPnL float64) string {
	if actualPnL == 0 {
		return ""
	}
	pct := math.Abs(residual/actualPnL) * 100
	switch {
	case pct > 10:
		return fmt.Sprintf("⚠ LARGE (%.1f%% of P&L -- investigate)", pct)
	case pct > 5:
		return fmt.Sprintf("△ elevated (%.1f%% of P&L)", pct)
	default:
		return "✓ within tolerance"
	}
}

// Attribute computes the full P&L attribution for a bond position
// between two curve snapshots.
//	1/252 for one business day)
func Attribute(
	b pricing.Bond,
	yesterdayCurve *curve.DiscountCurve,
	todayCurve *curve.DiscountCurve,
	horizonYears float64,
) Attribution {
	// ---- Actual P&L: reprice on both curves, take the difference ----
	priceYesterday := pricing.NPV(b, yesterdayCurve)
	priceToday := pricing.NPV(b, todayCurve)
	actualPnL := priceToday - priceYesterday

	// ---- Rate change: parallel shift approximation ----
	// Compute the average zero rate change across all shared pillar
	// tenors, as a simple summary of "how much did the curve move."
	// In bucketed attribution you'd do this per-pillar; here we use
	// the parallel approximation for clarity.
	deltaY := parallelRateChange(yesterdayCurve, todayCurve)
	deltaYBps := deltaY * 10000

	// Risk measures off yesterday's curve
	riskResults := risk.Analyze(b, yesterdayCurve)

	// DV01 P&L: linear (first-order) rate move effect
	dv01PnL := -riskResults.DV01 * deltaYBps

	// Convexity P&L: second-order curvature correction
	convexityPnL := 0.5 * riskResults.Convexity * priceYesterday * deltaY * deltaY

	// Carry P&L: time passage effect
	carryResults := carry.Analyze(b, yesterdayCurve, horizonYears, riskResults.DV01)
	carryPnL := carryResults.Total

	// Residual: what's left unexplained
	residual := actualPnL - dv01PnL - convexityPnL

	return Attribution{
		ActualPnL:     actualPnL,
		DV01PnL:       dv01PnL,
		ConvexityPnL:  convexityPnL,
		CarryPnL:      carryPnL,
		Residual:      residual,
		RateChangeBps: deltaYBps,
	}
}

// parallelRateChange computes the average zero rate change across
// all pillar tenors shared between two curves.
func parallelRateChange(yesterday, today *curve.DiscountCurve) float64 {
	pillars := yesterday.PillarTimes()
	if len(pillars) == 0 {
		return 0
	}

	totalChange := 0.0
	for _, t := range pillars {
		totalChange += today.ZeroRate(t) - yesterday.ZeroRate(t)
	}
	return totalChange / float64(len(pillars))
}