// Package pricing implements NPV and YTM calculations for fixed income
// instruments, using a bootstrapped DiscountCurve as the rate source.
package pricing

import (
	"fmt"
	"math"

	"github.com/christophernnh/curve-engine/internal/curve"
	"github.com/christophernnh/curve-engine/internal/daycount"
)

// Bullet bonds, single principal repayment at maturity with periodic coupon payments
type Bond struct {
	Maturity     float64            // years to maturity
	CouponRate   float64            // annual coupon rate, decimal (e.g. 0.05 = 5%)
	FaceValue    float64            // principal, typically 1.0 or 100.0
	Frequency    float64            // coupon payments per year (2 = semi-annual)
	CreditSpread float64            // spread over base curve, decimal (e.g. 0.0050 = 50bps)
	DayCount     daycount.Convention // day-count convention for accrual
}

// Standard treasury bond using act/365 fixed day count, semi-annual coupons, and no credit spread.
func NewTreasuryBond(maturity, couponRate, faceValue float64) Bond {
	return Bond{
		Maturity:     maturity,
		CouponRate:   couponRate,
		FaceValue:    faceValue,
		Frequency:    2.0, // semi-annual, standard US Treasury
		CreditSpread: 0.0,
		DayCount:     daycount.Act365Fixed,
	}
}

//  Corporate bond with a credit spread
func NewCorporateBond(maturity, couponRate, faceValue, creditSpreadBps float64) Bond {
	return Bond{
		Maturity:     maturity,
		CouponRate:   couponRate,
		FaceValue:    faceValue,
		Frequency:    2.0,
		CreditSpread: creditSpreadBps / 10000.0, // bps to decimal
		DayCount:     daycount.Thirty360Bond,
	}
}

// Cashflow represents a single future payment from the bond.
type Cashflow struct {
	Time   float64 // years from today
	Amount float64 // total payment (coupon, principal, or both)
}

// Schedule generates the complete cashflow schedule for this bond:
func (b Bond) Schedule() []Cashflow {
	step := 1.0 / b.Frequency
	couponAmount := b.CouponRate / b.Frequency * b.FaceValue

	var cashflows []Cashflow
	for t := b.Maturity; t > 1e-9; t -= step {
		amount := couponAmount
		if math.Abs(t-b.Maturity) < 1e-9 {
			// Final payment: last coupon + full principal repayment.
			amount += b.FaceValue
		}
		cashflows = append([]Cashflow{{Time: t, Amount: amount}}, cashflows...)
	}
	return cashflows
}

// adjustedDiscountFactor applies the bond's credit spread for corporate bonds.
func (b Bond) adjustedDiscountFactor(t float64, c *curve.DiscountCurve) float64 {
	baseDF := c.DiscountFactor(t)
	spreadDF := math.Exp(-b.CreditSpread * t)
	return baseDF * spreadDF
}

// NPV computes the present value of all bond cashflows discounted by
// the provided curve plus any credit spread. For a bond trading at
// par, NPV == FaceValue.
func NPV(b Bond, c *curve.DiscountCurve) float64 {
	pv := 0.0
	for _, cf := range b.Schedule() {
		pv += cf.Amount * b.adjustedDiscountFactor(cf.Time, c)
	}
	return pv
}

// Price returns the NPV as a percentage of face value. e.g. 97.5
func Price(b Bond, c *curve.DiscountCurve) float64 {
	return NPV(b, c) / b.FaceValue * 100.0
}

// YTM (Yield to Maturity) finds the single flat discount rate that provides the same NPV + credit spread
// Uses newton-raphson root-finder
func YTM(b Bond, c *curve.DiscountCurve) (float64, error) {
	targetPrice := NPV(b, c)

	objective := func(y float64) float64 {
		pv := 0.0
		for _, cf := range b.Schedule() {
			periods := cf.Time * b.Frequency
			pv += cf.Amount / math.Pow(1.0+y/b.Frequency, periods)
		}
		return pv - targetPrice
	}

	// Seed with the rough approximation: coupon/price + small adjustment
	seed := b.CouponRate + b.CreditSpread

	ytm, err := solveNewton(objective, seed)
	if err != nil {
		return 0, fmt.Errorf("pricing: YTM failed to converge: %w", err)
	}
	return ytm, nil
}