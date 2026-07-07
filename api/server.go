// Package api wires together the HTTP router, middleware, and handlers
// Handlers translate between JSON and domain types.
package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/christophernnh/curve-engine/api/handlers"
	"github.com/christophernnh/curve-engine/internal/curve"
	"github.com/christophernnh/curve-engine/internal/marketdata"
)

// Cache to prevent repeated fetching and bootstrapping
type CurveCache struct {
	mu          sync.RWMutex
	curve       *curve.DiscountCurve
	parYields   map[float64]float64 // maturity → original par yield
	lastFetched time.Time
	sourceDate  string
}

// Get returns the cached curve, refreshing from Treasury.gov if the cache is stale (older than 4 hours) or empty.
func (c *CurveCache) Get() (*curve.DiscountCurve, map[float64]float64, string, error) {
	c.mu.RLock()
	if c.curve != nil && time.Since(c.lastFetched) < 4*time.Hour {
		defer c.mu.RUnlock()
		return c.curve, c.parYields, c.sourceDate, nil
	}
	c.mu.RUnlock()

	// Stale or empty -- refresh.
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	yearMonth := now.Year()*100 + int(now.Month())
	bonds, err := marketdata.FetchLatestParBonds(yearMonth)
	if err != nil {
		prev := now.AddDate(0, -1, 0)
		yearMonth = prev.Year()*100 + int(prev.Month())
		bonds, err = marketdata.FetchLatestParBonds(yearMonth)
		if err != nil {
			// Return stale cache if fetch fails -- better than no data.
			if c.curve != nil {
				return c.curve, c.parYields, c.sourceDate, nil
			}
			return nil, nil, "", err
		}
	}

	bootstrapped, err := curve.Bootstrap(bonds, curve.LogLinearDFInterpolator{})
	if err != nil {
		return nil, nil, "", err
	}

	yields := make(map[float64]float64)
	for _, b := range bonds {
		pb := b.(curve.ParBond)
		yields[pb.Maturity()] = pb.Yield()
	}

	c.curve = bootstrapped
	c.parYields = yields
	c.lastFetched = now
	c.sourceDate = now.Format("2006-01-02")

	return c.curve, c.parYields, c.sourceDate, nil
}

// NewRouter builds and returns the fully configured chi router.
func NewRouter(cache *CurveCache) http.Handler {
	r := chi.NewRouter()

	// ---- Middleware ----
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	// ALLOWED_ORIGINS env var: comma-separated list of allowed origins.
	// ALLOWED_ORIGINS=https://curve-frontend.vercel.app
	allowedOrigins := []string{"*"}
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		allowedOrigins = strings.Split(origins, ",")
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	h := handlers.New(cache)

	// ---- Routes ----
	r.Route("/api", func(r chi.Router) {
		// Curve endpoints
		r.Get("/curve", h.GetCurve)
		r.Get("/curve/forward", h.GetForwardRate)

		// Instrument endpoints
		r.Post("/price", h.Price)
		r.Post("/risk", h.Risk)
		r.Post("/carry", h.Carry)
		r.Post("/pnl", h.PnL)
		r.Post("/hedge", h.Hedge)
	})

	// Health check -- useful for deployment monitoring.
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	return r
}