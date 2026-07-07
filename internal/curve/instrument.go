package curve

type BootstrapInstrument interface {
	// Maturity returns the instrument's maturity in years.
	Maturity() float64

	// Residual will be driven to zero to find the implied zero rate at the instrument's maturity.
	Residual(trialCurve *DiscountCurve) float64
}

// ParBond represents a hypothetical Treasury bond, freshly issued at par
type ParBond struct {
	maturity float64 // years
	yield    float64 // quoted par yield, e.g. 0.0413 for 4.13%
}

// NewParBond constructs a ParBond from a Treasury par yield quote.
func NewParBond(maturity, yield float64) ParBond {
	return ParBond{maturity: maturity, yield: yield}
}

func (b ParBond) Maturity() float64 { return b.maturity }

func (b ParBond) Yield() float64 { return b.yield }

// couponTimes returns the semi-annual coupon schedule
func (b ParBond) couponTimes() []float64 {
	const step = 0.5 // semi-annual, standard US Treasury convention
	var times []float64
	for t := b.maturity; t > 1e-9; t -= step {
		times = append([]float64{t}, times...)
	}
	return times
}

// Residual implements the par-bond pricing identity
//	price = couponRate/freq * sum(D(t_i)) + 1.0 * D(maturity)
func (b ParBond) Residual(trialCurve *DiscountCurve) float64 {
	const frequency = 2.0
	couponPerPeriod := b.yield / frequency

	pv := 0.0
	for _, t := range b.couponTimes() {
		pv += couponPerPeriod * trialCurve.DiscountFactor(t)
	}
	pv += 1.0 * trialCurve.DiscountFactor(b.maturity) // principal redemption

	const par = 1.0
	// gets the difference between the present value of the bond's cash flows and par (1.0)
	// effectively the residual that we want to be zero when the curve correctly prices the bond at par
	return pv - par
}