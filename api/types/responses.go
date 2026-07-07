package types

// ErrorResponse for 4xx or 5xx response codes
type ErrorResponse struct {
	Error string `json:"error"`
}

// Bootstrapped curve state.
type CurveResponse struct {
	// Pillars are the raw bootstrapped points (maturity, zero rate)
	Pillars []PillarPoint `json:"pillars"`

	// These are interpolated from the bootstrapped curve.
	Rates []TenorRate `json:"rates"`

	// Interpolation scheme used: either "log_linear_df" or "linear_zero"
	Interpolation string `json:"interpolation"`

	// SourceDate is the Treasury.gov date of the data (e.g. "2026-07-01").
	SourceDate string `json:"source_date"`
}

// Bootstrapped pillar: maturity and solved zero rate. Data from treasury.gov
type PillarPoint struct {
	MaturityYears  float64 `json:"maturity_years"`
	ZeroRate       float64 `json:"zero_rate"`        // decimal, e.g. 0.0447
	DiscountFactor float64 `json:"discount_factor"`  // D(t)
	ParYield       float64 `json:"par_yield"`        // original Treasury quote
}

// Interpolated tenor from bootstrapping process.
type TenorRate struct {
	MaturityYears  float64 `json:"maturity_years"`
	ZeroRate       float64 `json:"zero_rate"`
	DiscountFactor float64 `json:"discount_factor"`
}

// ForwardRateResponse returns the implied forward rate.
type ForwardRateResponse struct {
	T1          float64 `json:"t1"`
	T2          float64 `json:"t2"`
	ForwardRate float64 `json:"forward_rate"` // decimal
}

// PriceResponse returns bond valuation results.
type PriceResponse struct {
	NPV        float64 `json:"npv"`         // dollar value per face unit
	PricePct   float64 `json:"price_pct"`   // percentage of face, e.g. 96.14
	YTM        float64 `json:"ytm"`         // decimal, e.g. 0.0448
	Cashflows  []CashflowPoint `json:"cashflows"`
}

// CashflowPoint is one bond payment (coupon or principal).
type CashflowPoint struct {
	TimeYears      float64 `json:"time_years"`
	Amount         float64 `json:"amount"`
	PresentValue   float64 `json:"present_value"`
}

// All risk measures for a bond.
type RiskResponse struct {
	DV01             float64            `json:"dv01"`              // per face unit per 1bp
	ModifiedDuration float64            `json:"modified_duration"` // years
	Convexity        float64            `json:"convexity"`
	BucketedDV01     []BucketedDV01Point `json:"bucketed_dv01"`
}

// BucketedDV01Point is one pillar's DV01 contribution.
type BucketedDV01Point struct {
	MaturityYears float64 `json:"maturity_years"`
	DV01          float64 `json:"dv01"`
}

// CarryResponse returns carry, rolldown, and breakeven.
type CarryResponse struct {
	HorizonMonths float64 `json:"horizon_months"`
	Carry         float64 `json:"carry"`          // per face unit
	Rolldown      float64 `json:"rolldown"`        // per face unit
	Total         float64 `json:"total"`           // carry + rolldown
	BreakevenBps  float64 `json:"breakeven_bps"`   // rate move to wipe out carry
}

// PnLResponse returns P&L attribution.
type PnLResponse struct {
	RateShiftBps  float64 `json:"rate_shift_bps"`
	ActualPnL     float64 `json:"actual_pnl"`
	DV01PnL       float64 `json:"dv01_pnl"`
	ConvexityPnL  float64 `json:"convexity_pnl"`
	ExplainedPnL  float64 `json:"explained_pnl"`
	Residual      float64 `json:"residual"`
	CarryPnL      float64 `json:"carry_pnl"` // informational, separate track
}

// HedgeResponse returns hedge ratio and residual risk.
type HedgeResponse struct {
	HedgeRatio        float64              `json:"hedge_ratio"`
	HedgeNotional     float64              `json:"hedge_notional"`
	PositionDV01      float64              `json:"position_dv01"`
	HedgeDV01PerUnit  float64              `json:"hedge_dv01_per_unit"`
	ConvexityMismatch float64              `json:"convexity_mismatch"`
	ResidualDV01      []BucketedDV01Point  `json:"residual_dv01"`
	TotalResidualDV01 float64              `json:"total_residual_dv01"`
}