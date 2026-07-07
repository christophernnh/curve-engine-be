// Calcualates DV01, Modified Duration, bucketed DV01, and convexity. 
// All measures are computed by bumping the curve and repricing.
package risk

import (
	"github.com/christophernnh/curve-engine/internal/curve"
	"github.com/christophernnh/curve-engine/internal/pricing"
)

const bumpBps = 0.5 // basis points used for finite difference bumps

// Results holds all risk measures for a single bond position.
type Results struct {
	DV01 float64

	// ModifiedDuration is DV01 expressed as a percentage sensitivity:
	// approximately what % the price changes per 100bp of rate move.
	ModifiedDuration float64

	// BucketedDV01 maps each pillar maturity (in years) to the dollar
	// price change caused by a 1bp bump of ONLY that pillar.
	// The sum of all buckets approximates the total DV01.
	BucketedDV01 map[float64]float64

	// Convexity measures the curvature of the price/rate relationship.
	// A bond with higher convexity benefits more from rate falls and
	// loses less from rate rises than duration alone would predict.
	Convexity float64
}

// Analyze computes all risk measures for the given bond and curve.
func Analyze(b pricing.Bond, c *curve.DiscountCurve) Results {
	basePrice := pricing.NPV(b, c)

	return Results{
		DV01:             totalDV01(b, c, basePrice),
		ModifiedDuration: modifiedDuration(b, c, basePrice),
		BucketedDV01:     bucketedDV01(b, c),
		Convexity:        convexity(b, c, basePrice),
	}
}

// totalDV01 computes the price change for a 1bp PARALLEL shift.
//
// difference is 0.5 bps, so no need to divide by 2.0 to get DV01
func totalDV01(b pricing.Bond, c *curve.DiscountCurve, basePrice float64) float64 {
	upCurve := parallelBump(c, +bumpBps)
	downCurve := parallelBump(c, -bumpBps)

	priceUp := pricing.NPV(b, upCurve)
	priceDown := pricing.NPV(b, downCurve)

	return priceDown - priceUp
}

// modifiedDuration converts DV01 into a percentage-of-price measure.
// A duration of 8.1 means a 100bp move changes price by ~8.1%.
func modifiedDuration(b pricing.Bond, c *curve.DiscountCurve, basePrice float64) float64 {
	dv01 := totalDV01(b, c, basePrice)
	return dv01 / (basePrice * 0.0001)
}

// bucketedDV01 computes the price sensitivity to a 1bp bump of each
// individual pillar, holding all other pillars fixed.
func bucketedDV01(b pricing.Bond, c *curve.DiscountCurve) map[float64]float64 {
	pillars := c.PillarTimes()
	buckets := make(map[float64]float64, len(pillars))

	for i, t := range pillars {
		upCurve := c.Bump(i, +bumpBps)
		downCurve := c.Bump(i, -bumpBps)

		priceUp := pricing.NPV(b, upCurve)
		priceDown := pricing.NPV(b, downCurve)

		// Same fix as totalDV01: h=0.5bp, 2h=1bp, so no /2 needed.
		buckets[t] = priceDown - priceUp
	}
	return buckets
}

// convexity measures the curvature of the price/yield relationship.
// Formula (second-order finite difference):
//	Convexity = (Price_up + Price_down - 2×Price_base) / (Price_base × bump²)
func convexity(b pricing.Bond, c *curve.DiscountCurve, basePrice float64) float64 {
	upCurve := parallelBump(c, +bumpBps)
	downCurve := parallelBump(c, -bumpBps)

	priceUp := pricing.NPV(b, upCurve)
	priceDown := pricing.NPV(b, downCurve)

	bumpDecimal := bumpBps / 10000.0
	return (priceUp + priceDown - 2*basePrice) / (basePrice * bumpDecimal * bumpDecimal)
}

// parallelBump returns a new curve with ALL pillars shifted by bumpBps.
func parallelBump(c *curve.DiscountCurve, bumpBps float64) *curve.DiscountCurve {
	pillars := c.PillarTimes()
	bumped := c
	for i := range pillars {
		bumped = bumped.Bump(i, bumpBps)
	}
	return bumped
}
