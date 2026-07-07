package curve

import (
	"fmt"
	"math"
	"sort"
)

// Bootstrap builds a DiscountCurve by interpolating zero rates.
//
// Short maturities with no coupons before maturity (e.g. 3M, 6M
// T-Bills) are solved in closed form. Longer maturities with
// intermediate semi-annual coupons require root-finding.
func Bootstrap(bonds []BootstrapInstrument, interp Interpolator) (*DiscountCurve, error) {
	if len(bonds) == 0 {
		return nil, fmt.Errorf("curve: cannot bootstrap from zero instruments")
	}

	sorted := append([]BootstrapInstrument(nil), bonds...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Maturity() < sorted[j].Maturity()
	})

	times := make([]float64, 0, len(sorted))
	rates := make([]float64, 0, len(sorted))

	for _, inst := range sorted {
		bond, ok := inst.(ParBond)
		if !ok {
			return nil, fmt.Errorf("curve: unsupported instrument type %T", inst)
		}

		var zeroRate float64

		if len(bond.couponTimes()) <= 1 {
			// No coupons before maturity (e.g. 3M/6M T-Bill) -- solve directly
			price := 1.0 // par
			payoff := 1.0 + bond.yield*bond.maturity
			df := price / payoff
			zeroRate = -math.Log(df) / bond.maturity
		} else {
			// Coupon case: root-find the zero rate at this new
			// pillar such that Residual() == 0 using newton raphson, given all previously
			// solved pillars plus this trial pillar.
			seed := seedGuess(rates)

			objective := func(z float64) float64 {
				trialTimes := append(append([]float64(nil), times...), bond.Maturity())
				trialRates := append(append([]float64(nil), rates...), z)
				trialCurve := NewDiscountCurve(trialTimes, trialRates, interp)
				return bond.Residual(trialCurve)
			}

			solved, err := solveNewton(objective, seed)
			if err != nil {
				return nil, fmt.Errorf("curve: failed to bootstrap pillar at maturity %v: %w", bond.Maturity(), err)
			}
			zeroRate = solved
		}

		times = append(times, bond.Maturity())
		rates = append(rates, zeroRate)
	}

	return NewDiscountCurve(times, rates, interp), nil
}

// seedGuess provides a starting guess for the Newton-Raphson solver.
func seedGuess(previousRates []float64) float64 {
	if len(previousRates) == 0 {
		return 0.03 // reasonable starting point (~3%)
	}
	return previousRates[len(previousRates)-1]
}