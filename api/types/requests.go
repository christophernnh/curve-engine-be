// Requests for the API endpoints.
package types

// Defines bonds
type BondRequest struct {
	Maturity        float64 `json:"maturity"`          // years, e.g. 10.0
	CouponRate      float64 `json:"coupon_rate"`       // decimal, e.g. 0.04
	FaceValue       float64 `json:"face_value"`        // e.g. 100.0
	CreditSpreadBps float64 `json:"credit_spread_bps"` // bps, e.g. 50. 0 for Treasury
}

// PriceRequest asks for NPV, price, and YTM of a bond.
type PriceRequest struct {
	Bond BondRequest `json:"bond"`
}

// RiskRequest asks for DV01, duration, convexity, and bucketed DV01.
type RiskRequest struct {
	Bond BondRequest `json:"bond"`
}

// CarryRequest asks for carry, rolldown, and breakeven for a horizon.
type CarryRequest struct {
	Bond          BondRequest `json:"bond"`
	HorizonMonths float64     `json:"horizon_months"` // e.g. 3 for 3-month carry
}

// PnLRequest asks for P&L attribution between two rate scenarios.
// RateShiftBps simulates an overnight parallel curve move --
// positive = rates rose, negative = rates fell.
type PnLRequest struct {
	Bond          BondRequest `json:"bond"`
	RateShiftBps  float64     `json:"rate_shift_bps"` // e.g. 5 = rates rose 5bps
}

// Requests for hedge ratio and residual DV01 analysis.
type HedgeRequest struct {
	Position        BondRequest `json:"position"`
	HedgeInstrument BondRequest `json:"hedge_instrument"`
	PositionFace    float64     `json:"position_face"` // e.g. 10000000 for $10M
}

// Requests for the implied forward rate between two tenors.
type ForwardRateRequest struct {
	T1 float64 `json:"t1"` // start tenor in years
	T2 float64 `json:"t2"` // end tenor in years
}