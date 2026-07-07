package curve

import "math"

// Dicount factor curve D(t)
type DiscountCurve struct {
	pillarTimes []float64
	pillarRates []float64
	interp      Interpolator
}

// Discount Curve D(t) constructor.
// Times, rates and interpolator must be provided. times and rates must be the same length and strictly ascending.
func NewDiscountCurve(times, rates []float64, interp Interpolator) *DiscountCurve {
	if len(times) != len(rates) {
		panic("curve: pillarTimes and pillarRates must be the same length")
	}
	for i := 1; i < len(times); i++ {
		if times[i] <= times[i-1] {
			panic("curve: pillarTimes must be strictly ascending")
		}
	}

	t := append([]float64(nil), times...)
	r := append([]float64(nil), rates...)
	return &DiscountCurve{pillarTimes: t, pillarRates: r, interp: interp}
}

// DiscountFactor returns D(t), the present value of $1 received at time t from the interpolated discount curve.
func (c *DiscountCurve) DiscountFactor(t float64) float64 {
	if t <= 0 {
		return 1.0
	}
	return c.interp.DiscountFactor(t, c.pillarTimes, c.pillarRates)
}

// ZeroRate returns the continuously compounded zero rate z(t) implied
// by the curve at time t, derived from D(t) = exp(-z(t)*t).
// Converts discount factors to zero rates. (example: 0.95 at 1 year implies a zero rate of -ln(0.95)/1 = 5.13%).
func (c *DiscountCurve) ZeroRate(t float64) float64 {
	if t <= 0 {
		return c.pillarRates[0]
	}
	df := c.DiscountFactor(t)
	return -math.Log(df) / t
}

// ForwardRate returns the simple (non-compounded) forward rate implied
// by the curve between t1 and t2, t1 < t2:
//	F(t1, t2) = (D(t1)/D(t2) - 1) / (t2 - t1)
func (c *DiscountCurve) ForwardRate(t1, t2 float64) float64 {
	if t2 <= t1 {
		panic("curve: ForwardRate requires t2 > t1")
	}
	d1 := c.DiscountFactor(t1)
	d2 := c.DiscountFactor(t2)
	return (d1/d2 - 1.0) / (t2 - t1)
}

// Returns a new discount curve with pillars bumped by bumpBps (in basis points)
func (c *DiscountCurve) Bump(pillarIndex int, bumpBps float64) *DiscountCurve {
	if pillarIndex < 0 || pillarIndex >= len(c.pillarRates) {
		panic("curve: pillarIndex out of range")
	}
	newRates := append([]float64(nil), c.pillarRates...)
	newRates[pillarIndex] += bumpBps / 10000.0 // converting bps to decimal
	return NewDiscountCurve(c.pillarTimes, newRates, c.interp)
}

// PillarTimes returns a copy of the pillar times (read-only access).
func (c *DiscountCurve) PillarTimes() []float64 {
	return append([]float64(nil), c.pillarTimes...)
}