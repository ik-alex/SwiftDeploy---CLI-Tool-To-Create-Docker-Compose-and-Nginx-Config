// package main

// import (
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"os"
// 	"sync"
// 	"time"

// 	"github.com/go-chi/chi/v5"
// 	"github.com/go-chi/chi/v5/middleware"
// )

// var timeDuration = time.Duration(0)
// var syncMutex sync.Mutex
// var errorRate float64 = 0.0

// // -----------------------------------------
// // Response matches for your required JSON structure
// // ------------------------------------------

// type StatusResponse struct {
// 	Mode string `json:"mode"`
// 	Version string `json:"version"`
// 	Timestamp time.Time `json:"timestamp"`
// }

// // -----------------------------------------
// // A struct to parse the chaos configuration from the request body
// // -----------------------------------------

// type ChaosConfig struct {
// 	Mode string `json:"mode"`
// 	Duration int64 `json:"duration"`
// 	Rate float64 `json:"rate"`
// }


// // ------------------------------------------
// // Tracking time of chaos
// // ------------------------------------------

// var startTime = time.Now()

// func getUpTime() float64 {
// 	return time.Since(startTime).Seconds()
// }

// // -----------------------------------------
// // Response helpers
// // -----------------------------------------

// func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(statusCode)
// 	json.NewEncoder(w).Encode(payload)
// }

// func readJSON(w http.ResponseWriter, r *http.Request, data any) error {
// 	maxBytes := 1_048_578 // 1mb
// 	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

// 	decoder := json.NewDecoder(r.Body)
// 	decoder.DisallowUnknownFields()

// 	return decoder.Decode(data)
// }

// func writeJSONError(w http.ResponseWriter, status int, message string) {
// 	type envelope struct {
// 		Error string `json:"error"`
// 	}

// 	writeJSON(w, status, &envelope{Error: message})
// }


// // ----------------------------------------
// // Handlers
// // ----------------------------------------

// func rootHandler(w http.ResponseWriter, r *http.Request) {

// 	// reading mode from environment variable
// 	mode := os.Getenv("MODE")
// 	if mode == "" {
// 		mode = "stable"
// 	}
// 	writeJSON(w, http.StatusOK, StatusResponse{
// 		Mode:      mode,
// 		Version:   "1.0.0",
// 		Timestamp: time.Now(),
// 	})
// }

// func healthzHandler(w http.ResponseWriter, r *http.Request) {
// 	writeJSON(w, http.StatusOK, map[string]any{
// 		"status": "ok",
// 		"uptime": getUpTime(),
// 	})
// }

// func chaosHandler(w http.ResponseWriter, r *http.Request) {
// 	// reading mode from environment variable
// 	mode := os.Getenv("MODE")
// 	if mode == "canary" {
// 		var chaosConfig ChaosConfig
// 		if err := readJSON(w, r, &chaosConfig); err != nil {
// 			writeJSONError(w, http.StatusBadRequest, err.Error())
// 			return
// 		}

// 		syncMutex.Lock()
// 		defer syncMutex.Unlock()
// 		switch chaosConfig.Mode {
// 		case "slow":
// 			timeDuration = time.Duration(chaosConfig.Duration) * time.Second
// 		case "error":
// 			errorRate = chaosConfig.Rate
// 		case "recover":
// 			timeDuration = time.Duration(0)
// 			errorRate = 0.0
// 		default:
// 			writeJSONError(w, http.StatusBadRequest, "Invalid chaos mode")
// 			return
// 		}
		
// 		writeJSON(w, http.StatusOK, StatusResponse{
// 		Mode:      mode,
// 		Version:   "1.0.0",
// 		Timestamp: time.Now(),
// 	})

// 	} else {
// 		writeJSONError(w, http.StatusForbidden, "Chaos mode is only available in canary mode")
// 		return
// 	}

// }

// // ------------------------------------------
// // Middleware to inject chaos into responses
// // ------------------------------------------
//  func canaryMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
//  }


// // ----------------------------------------
// // Main
// // ----------------------------------------

// func main() {
// 	r := chi.NewRouter()

// 	// Middleware
// 	r.Use(middleware.Logger)
// 	r.Use(middleware.Recoverer)
// 	r.Use(canaryMiddleware)

// 	// Routes
// 	r.Get("/", rootHandler)
// 	r.Get("/healthz", healthzHandler)
// 	r.Post("/chaos", chaosHandler)

// 	// Start server
// 	port := os.Getenv("PORT")
// 	if port == "" {
// 		port = "3000"
// 	}

// 	fmt.Printf("Server starting on port %s\n", port)
// 	if err := http.ListenAndServe(":"+port, r); err != nil {
// 		fmt.Fprintf(os.Stderr, "Server failed: %v\n", err)
// 		os.Exit(1)
// 	}
// }



package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// ── Chaos State ──

var (
	chaosMu      sync.RWMutex
	chaosSlowDur time.Duration
	chaosErrRate float64
	startTime    = time.Now()
)

// ── Helpers ──

func getMode() string {
	mode := os.Getenv("MODE")
	if mode == "" {
		return "stable"
	}
	return mode
}

func getVersion() string {
	v := os.Getenv("APP_VERSION")
	if v == "" {
		return "1.0.0"
	}
	return v
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// ── Handlers ──

func rootHandler(w http.ResponseWriter, r *http.Request) {
	mode := getMode()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":   fmt.Sprintf("Welcome! Running in %s mode", mode),
		"mode":      mode,
		"version":   getVersion(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"uptime": time.Since(startTime).Seconds(),
		"mode":   getMode(),
	})
}

func chaosHandler(w http.ResponseWriter, r *http.Request) {
	if getMode() != "canary" {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error": "Chaos endpoint is only available in canary mode",
		})
		return
	}

	var body struct {
		Mode     string  `json:"mode"`
		Duration int     `json:"duration"`
		Rate     float64 `json:"rate"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid JSON body",
		})
		return
	}

	chaosMu.Lock()
	defer chaosMu.Unlock()

	switch body.Mode {
	case "slow":
		chaosSlowDur = time.Duration(body.Duration) * time.Second
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":   "chaos activated",
			"chaos":    "slow",
			"duration": body.Duration,
		})
	case "error":
		chaosErrRate = body.Rate
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "chaos activated",
			"chaos":  "error",
			"rate":   body.Rate,
		})
	case "recover":
		chaosSlowDur = 0
		chaosErrRate = 0
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "recovered",
			"chaos":  "none",
		})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid chaos mode. Use: slow, error, or recover",
		})
	}
}

// ── Middleware ──

func canaryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mode := getMode()

		if mode == "canary" {
			w.Header().Set("X-Mode", "canary")

			chaosMu.RLock()
			slowDur := chaosSlowDur
			errRate := chaosErrRate
			chaosMu.RUnlock()

			// Error chaos
			if errRate > 0 && rand.Float64() < errRate {
				writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "Chaos-induced server error",
				})
				return
			}

			// Slow chaos
			if slowDur > 0 {
				time.Sleep(slowDur)
			}
		}

		next.ServeHTTP(w, r)
	})
}

// ── Main ──

func main() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(canaryMiddleware)

	r.Get("/", rootHandler)
	r.Get("/healthz", healthzHandler)
	r.Post("/chaos", chaosHandler)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "3000"
	}

	fmt.Printf("SwiftDeploy service starting on port %s (mode: %s, version: %s)\n",
		port, getMode(), getVersion())

	if err := http.ListenAndServe(":"+port, r); err != nil {
		fmt.Fprintf(os.Stderr, "Server failed: %v\n", err)
		os.Exit(1)
	}
}