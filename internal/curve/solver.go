package curve

import "fmt"

const (
	solverMaxIterations  = 100
	solverTolerance      = 1e-10
	solverDerivativeStep = 1e-6
)

// Newton-Raphson helper with a numerically estimated derivative
// (finite difference), since f here runs through curve interpolation
// and has no convenient closed-form derivative.
func solveNewton(f func(float64) float64, guess float64) (float64, error) {
	x := guess
	for i := 0; i < solverMaxIterations; i++ {
		fx := f(x)
		if abs(fx) < solverTolerance {
			return x, nil
		}

		// Numerical derivative: f'(x) ≈ (f(x+h) - f(x)) / h
		fxh := f(x + solverDerivativeStep)
		deriv := (fxh - fx) / solverDerivativeStep
		if deriv == 0 {
			return 0, fmt.Errorf("curve: zero derivative encountered during root-find at x=%v", x)
		}

		x = x - fx/deriv
	}
	return 0, fmt.Errorf("curve: root-find failed to converge after %d iterations", solverMaxIterations)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}