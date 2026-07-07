// Package hedge computes the hedge ratio and residual risk for a bond
// position hedged with a benchmark instrument. The goal is DV01
// neutrality: after the hedge, a 1bp rate move should produce near-
// zero net P&L across the combined position.
package hedge

import (
	"fmt"

	"github.com/christophernnh/curve-engine/internal/curve"
	"github.com/christophernnh/curve-engine/internal/pricing"
	"github.com/christophernnh/curve-engine/internal/risk"
)

// Result holds the full hedge analysis for a position/hedge pair.
type Result struct {
	// HedgeRatio is the face-value multiplier of the hedge instrument
	// needed to neutralize the position's DV01. Negative = short the
	// hedge instrument (the common case when hedging a long position).
	// e.g. -1.822 means sell 1.822 units of face value per unit of
	// position face value.
	HedgeRatio float64

	// PositionDV01 is the total DV01 of the position before hedging.
	PositionDV01 float64

	// HedgeDV01PerUnit is the DV01 of one unit of the hedge instrument.
	HedgeDV01PerUnit float64

	// ResidualDV01 maps each curve pillar to the net DV01 remaining
	// after the hedge -- ideally near zero at every bucket.
	// Non-zero buckets reveal where basis risk remains.
	ResidualDV01 map[float64]float64

	// TotalResidualDV01 is the sum of all residual bucket DV01s --
	// the net rate exposure left unhedged across the whole curve.
	TotalResidualDV01 float64

	// ConvexityMismatch is the net convexity of the combined position
	// (position + hedge). Non-zero means the hedge drifts out of
	// alignment for large rate moves and needs rebalancing.
	ConvexityMismatch float64
}

// String returns a formatted hedge report for display.
func (r Result) String() string {
	sign := "SELL"
	ratio := -r.HedgeRatio
	if r.HedgeRatio > 0 {
		sign = "BUY"
		ratio = r.HedgeRatio
	}

	residualFlag := "✓ well hedged"
	if abs(r.TotalResidualDV01) > 0.001*abs(r.PositionDV01) {
		residualFlag = fmt.Sprintf("△ basis risk remains (%.4f unhedged DV01)", r.TotalResidualDV01)
	}

	report := fmt.Sprintf(
		"=== Hedge Report ===\n"+
			"Position DV01:      %+.6f\n"+
			"Hedge DV01/unit:    %+.6f\n"+
			"Hedge ratio:        %s %.4f units of hedge instrument\n"+
			"Convexity mismatch: %+.4f\n"+
			"Residual DV01:      %s\n"+
			"Residual by bucket:\n",
		r.PositionDV01,
		r.HedgeDV01PerUnit,
		sign, ratio,
		r.ConvexityMismatch,
		residualFlag,
	)

	for tenor, dv01 := range r.ResidualDV01 {
		if abs(dv01) > 1e-8 {
			report += fmt.Sprintf("  %5.1fY: %+.6f\n", tenor, dv01)
		}
	}
	return report
}

// Compute calculates the hedge ratio and residual risk for hedging
// a position bond with a hedge instrument, both priced off the same
// curve. positionFace and hedgeFace define the face value of each --
// the hedge ratio is expressed per unit of hedge face value.
func Compute(
	position pricing.Bond,
	hedgeInstrument pricing.Bond,
	c *curve.DiscountCurve,
) (Result, error) {
	// ---- Risk measures for both instruments ----
	positionRisk := risk.Analyze(position, c)
	hedgeRisk := risk.Analyze(hedgeInstrument, c)

	if abs(hedgeRisk.DV01) < 1e-10 {
		return Result{}, fmt.Errorf("hedge: hedge instrument has near-zero DV01, cannot compute ratio")
	}

	// ---- Hedge ratio ----
	// We want: positionDV01 + ratio × hedgeDV01 = 0
	// Solving: ratio = -positionDV01 / hedgeDV01
	ratio := -positionRisk.DV01 / hedgeRisk.DV01

	// ---- Residual DV01 by bucket ----
	// For each curve pillar: net = position bucket + ratio × hedge bucket.
	// A perfect parallel hedge would leave zero everywhere.
	// A cross-maturity hedge (e.g. hedging 10Y with 5Y) leaves clear
	// residuals at the buckets where the two bonds differ.
	residual := make(map[float64]float64)
	totalResidual := 0.0
	for tenor, posDV01 := range positionRisk.BucketedDV01 {
		hedgeBucket := hedgeRisk.BucketedDV01[tenor]
		net := posDV01 + ratio*hedgeBucket
		residual[tenor] = net
		totalResidual += net
	}

	// ---- Convexity mismatch ----
	// Net convexity of the combined position + hedge.
	// Zero = perfectly convexity-matched (rare in practice).
	// Positive = position has more convexity than hedge (you benefit
	//   from large moves, but need to rebalance the hedge over time).
	// Negative = hedge has more convexity than position.
	convexityMismatch := positionRisk.Convexity + ratio*hedgeRisk.Convexity

	return Result{
		HedgeRatio:        ratio,
		PositionDV01:      positionRisk.DV01,
		HedgeDV01PerUnit:  hedgeRisk.DV01,
		ResidualDV01:      residual,
		TotalResidualDV01: totalResidual,
		ConvexityMismatch: convexityMismatch,
	}, nil
}

// ComputeNotional is a convenience wrapper that converts the unit-based
// hedge ratio into an actual face-value notional amount, given the
// position's face value.
//
// e.g. if HedgeRatio = -1.822 and positionFace = $10M,
// then HedgeNotional = -$18.22M (sell $18.22M of the hedge instrument).
func ComputeNotional(
	position pricing.Bond,
	hedgeInstrument pricing.Bond,
	c *curve.DiscountCurve,
	positionFaceValue float64,
) (Result, float64, error) {
	result, err := Compute(position, hedgeInstrument, c)
	if err != nil {
		return Result{}, 0, err
	}
	hedgeNotional := result.HedgeRatio * positionFaceValue
	return result, hedgeNotional, nil
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}