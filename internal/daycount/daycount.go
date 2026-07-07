// Converts dates to year fractions according to various day count conventions.
package daycount

import "time"

type Convention int

const (
	Act360 Convention = iota // Act350 = 0
	Act365Fixed // Act365Fixed = 1
	Thirty360Bond // Thirty360Bond = 2
)

// YearFraction returns the accrual fraction of a year between start and end
// dates, according to the given convention. start must not be after end.
func YearFraction(start, end time.Time, conv Convention) float64 {
	switch conv {
	case Act360:
		return actualDays(start, end) / 360.0
	case Act365Fixed:
		return actualDays(start, end) / 365.0
	case Thirty360Bond:
		return thirty360(start, end)
	default:
		panic("daycount: unknown convention")
	}
}

// returns the actual number of calendar days between two dates.
func actualDays(start, end time.Time) float64 {
	// Normalize to UTC midnight
	s := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	e := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	return e.Sub(s).Hours() / 24.0
}

// thirty360 implements the 30/360 Bond Basis (US) convention.
func thirty360(start, end time.Time) float64 {
	y1, m1, d1 := start.Date()
	y2, m2, d2 := end.Date()

	if d1 == 31 {
		d1 = 30
	}
	if d2 == 31 && d1 == 30 {
		d2 = 30
	}

	days := 360*(y2-y1) + 30*(int(m2)-int(m1)) + (d2 - d1)
	return float64(days) / 360.0
}