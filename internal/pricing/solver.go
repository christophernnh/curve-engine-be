package pricing

import "fmt"

const (
	solverMaxIterations  = 100
	solverTolerance      = 1e-10
	solverDerivativeStep = 1e-6
)

func solveNewton(f func(float64) float64, guess float64) (float64, error) {
	x := guess
	for i := 0; i < solverMaxIterations; i++ {
		fx := f(x)
		if absVal(fx) < solverTolerance {
			return x, nil
		}
		fxh := f(x + solverDerivativeStep)
		deriv := (fxh - fx) / solverDerivativeStep
		if deriv == 0 {
			return 0, fmt.Errorf("zero derivative at x=%v", x)
		}
		x = x - fx/deriv
	}
	return 0, fmt.Errorf("failed to converge after %d iterations", solverMaxIterations)
}

func absVal(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}