package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/christophernnh/curve-engine/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialise the curve cache. The first request will trigger a
	// live fetch from Treasury.gov; subsequent requests within 4 hours
	// use the cached curve.
	cache := &api.CurveCache{}

	// Eagerly warm the cache on startup so the first real request
	// doesn't wait for the Treasury.gov fetch.
	log.Println("Warming curve cache from Treasury.gov...")
	if _, _, _, err := cache.Get(); err != nil {
		log.Printf("Warning: initial curve fetch failed: %v (will retry on first request)", err)
	} else {
		log.Println("Curve cache warmed successfully.")
	}

	router := api.NewRouter(cache)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Curve engine API listening on http://localhost%s\n", addr)
	log.Println("Endpoints:")
	log.Println("  GET  /health")
	log.Println("  GET  /api/curve")
	log.Println("  GET  /api/curve/forward?t1=5&t2=10")
	log.Println("  POST /api/price")
	log.Println("  POST /api/risk")
	log.Println("  POST /api/carry")
	log.Println("  POST /api/pnl")
	log.Println("  POST /api/hedge")

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}