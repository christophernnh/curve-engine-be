// Package curve implements discount curve construction and interpolation.
package curve

import "math"

// Strategy for interpolating between pillar points on a discount curve.
type Interpolator interface {
	DiscountFactor(t float64, pillarTimes, pillarRates []float64) float64
}

// NOT USED: LinearZeroInterpolator linearly interpolates the continuously
// compounded zero rate z(t) between pillars, then converts to a
// discount factor via D(t) = exp(-z(t) * t).

type LinearZeroInterpolator struct{}

func (LinearZeroInterpolator) DiscountFactor(t float64, times, rates []float64) float64 {
	z := interpolateLinear(t, times, rates)
	return math.Exp(-z * t)
}

// LogLinearDFInterpolator linearly interpolates ln(D(t)) between
// pillars, then exponentiates back to a discount factor.
type LogLinearDFInterpolator struct{}

func (LogLinearDFInterpolator) DiscountFactor(t float64, times, rates []float64) float64 {
	// Convert pillar zero rates to pillar log-discount-factors first.
	logDFs := make([]float64, len(times))
	for i, ti := range times {
		logDFs[i] = -rates[i] * ti
	}
	logDF := interpolateLinear(t, times, logDFs)
	return math.Exp(logDF)
}

// Shared interpolation helper: interpolateLinear performs linear interpolation of y as a function of
// x at point t, given sorted x-values xs and corresponding y-values ys.
func interpolateLinear(t float64, xs, ys []float64) float64 {
	n := len(xs)
	if n == 0 {
		panic("curve: cannot interpolate with zero pillars")
	}
	if n == 1 || t <= xs[0] {
		return ys[0]
	}
	if t >= xs[n-1] {
		return ys[n-1]
	}

	// Find the bracketing interval [xs[i], xs[i+1]] containing t.
	i := 0
	for i < n-2 && xs[i+1] < t {
		i++
	}

	x0, x1 := xs[i], xs[i+1]
	y0, y1 := ys[i], ys[i+1]
	weight := (t - x0) / (x1 - x0)
	return y0 + weight*(y1-y0)
}