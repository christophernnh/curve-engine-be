package carry

import (
	"math"

	"github.com/christophernnh/curve-engine/internal/curve"
	"github.com/christophernnh/curve-engine/internal/pricing"
)

// Results holds carry and rolldown outputs for a single bond position
// over a specified horizon.
type Results struct {
	// Carry: net income from coupon income received minus funding cost paid.
	// Expressed in dollars per unit of face value.
	Carry float64

	// Rolldown is the price change from the bond aging down the curve
	// over the horizon, assuming the curve stays completely unchanged.
	// Positive on an upward-sloping curve (bond rolls to lower yield).
	// Expressed in dollars per unit of face value.
	Rolldown float64

	// Total is Carry + Rolldown
	// Full P&L from holding the bond, ignoring any changes in the curve.
	Total float64

	// BreakevenBps is how many basis points rates must move AGAINST
	// the position to wipe out the carry+rolldown benefit.
	// Computed as Total / DV01, giving the rate move needed to break even.
	BreakevenBps float64
}

// Computes carry and rolldown for a bond over the given
// horizon (in years, e.g. 0.25 for 3 months), using the provided
// curve for both discounting and funding rate estimation.
func Analyze(b pricing.Bond, c *curve.DiscountCurve, horizonYears float64, dv01 float64) Results {
	carryAmount := carry(b, c, horizonYears)
	rolldownAmount := rolldown(b, c, horizonYears)
	total := carryAmount + rolldownAmount

	breakeven := 0.0
	if dv01 != 0 {
		breakeven = (total / dv01)
	}

	return Results{
		Carry:        carryAmount,
		Rolldown:     rolldownAmount,
		Total:        total,
		BreakevenBps: breakeven,
	}
}

// carry computes the net income from holding the bond over the horizon:
// coupon accrued minus funding cost.
//
// Coupon income: the bond accrues coupon linearly over time.
// For a bond paying CouponRate annually with FaceValue:
//   coupon income = CouponRate × FaceValue × horizonYears
//
// Funding cost: we approximate the overnight funding rate using the
// shortest available point on the curve (1M discount factor)

//   funding cost = fundingRate × Price × horizonYears
func carry(b pricing.Bond, c *curve.DiscountCurve, horizonYears float64) float64 {
	// Coupon income accrued over the horizon.
	couponIncome := b.CouponRate * b.FaceValue * horizonYears

	// Funding cost: we approximate the overnight funding rate
	fundingPillar := 1.0
	fundingRate := c.ZeroRate(fundingPillar)

	// Current bond price (what you paid for the bond in question).)
	currentPrice := pricing.NPV(b, c)

	// Funding cost: interest on the borrowed amount over the horizon.
	fundingCost := fundingRate * currentPrice * horizonYears

	return couponIncome - fundingCost
}

// rolldown computes the price change from the bond aging down the
// curve over the horizon, with the curve held completely fixed.
//
// Price the bond with its current maturity, 
// then price the same bond with maturity reduced by the horizon (it has aged).
// using the same unchanged curve for both. The difference is rolldown.

func rolldown(b pricing.Bond, c *curve.DiscountCurve, horizonYears float64) float64 {
	currentPrice := pricing.NPV(b, c)

	// Create an aged version of the bond: same coupon, same face value.
	agedMaturity := b.Maturity - horizonYears
	if agedMaturity <= 0 {
		// Bond matures within the horizon -- no rolldown to compute.
		return 0
	}

	agedBond := pricing.Bond{
		Maturity:     agedMaturity,
		CouponRate:   b.CouponRate,
		FaceValue:    b.FaceValue,
		Frequency:    b.Frequency,
		CreditSpread: b.CreditSpread,
		DayCount:     b.DayCount,
	}

	agedPrice := pricing.NPV(agedBond, c)
	return agedPrice - currentPrice
}

// BreakevenRateMove returns how many basis points the curve would need
// to shift in the adverse direction to offset a given total P&L.
func BreakevenRateMove(totalPnL, dv01 float64) float64 {
	if dv01 == 0 {
		return math.Inf(1)
	}
	return totalPnL / dv01
}