package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
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

// ── Prometheus Metrics ──

type MetricsCollector struct {
	mu             sync.Mutex
	requestCounts  map[string]int64
	histogramSums  map[string]float64
	histogramCount map[string]int64
	histogramBuckets map[string]map[float64]int64
	buckets        []float64
}

var metrics = &MetricsCollector{
	requestCounts:    make(map[string]int64),
	histogramSums:    make(map[string]float64),
	histogramCount:   make(map[string]int64),
	histogramBuckets: make(map[string]map[float64]int64),
	buckets:          []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
}

func (m *MetricsCollector) RecordRequest(method, path string, statusCode int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// http_requests_total
	key := fmt.Sprintf("method=%s,path=%s,status_code=%d", method, path, statusCode)
	m.requestCounts[key]++

	// http_request_duration_seconds histogram
	histKey := fmt.Sprintf("method=%s,path=%s", method, path)
	secs := duration.Seconds()
	m.histogramSums[histKey] += secs
	m.histogramCount[histKey]++

	if _, ok := m.histogramBuckets[histKey]; !ok {
		m.histogramBuckets[histKey] = make(map[float64]int64)
	}
	for _, b := range m.buckets {
		if secs <= b {
			m.histogramBuckets[histKey][b]++
		}
	}
}

func (m *MetricsCollector) Render() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := ""

	// http_requests_total
	out += "# HELP http_requests_total Total number of HTTP requests.\n"
	out += "# TYPE http_requests_total counter\n"
	for key, count := range m.requestCounts {
		out += fmt.Sprintf("http_requests_total{%s} %d\n", key, count)
	}

	// http_request_duration_seconds
	out += "# HELP http_request_duration_seconds HTTP request latency in seconds.\n"
	out += "# TYPE http_request_duration_seconds histogram\n"

	sortedBuckets := make([]float64, len(m.buckets))
	copy(sortedBuckets, m.buckets)
	sort.Float64s(sortedBuckets)

	for histKey, bucketCounts := range m.histogramBuckets {
		cumulative := int64(0)
		for _, b := range sortedBuckets {
			cumulative += bucketCounts[b]
			out += fmt.Sprintf("http_request_duration_seconds_bucket{%s,le=\"%s\"} %d\n",
				histKey, formatFloat(b), cumulative)
		}
		out += fmt.Sprintf("http_request_duration_seconds_bucket{%s,le=\"+Inf\"} %d\n",
			histKey, m.histogramCount[histKey])
		out += fmt.Sprintf("http_request_duration_seconds_sum{%s} %f\n",
			histKey, m.histogramSums[histKey])
		out += fmt.Sprintf("http_request_duration_seconds_count{%s} %d\n",
			histKey, m.histogramCount[histKey])
	}

	// app_uptime_seconds
	out += "# HELP app_uptime_seconds Time since application start.\n"
	out += "# TYPE app_uptime_seconds gauge\n"
	out += fmt.Sprintf("app_uptime_seconds %f\n", time.Since(startTime).Seconds())

	// app_mode
	out += "# HELP app_mode Current application mode (0=stable, 1=canary).\n"
	out += "# TYPE app_mode gauge\n"
	modeVal := 0
	if getMode() == "canary" {
		modeVal = 1
	}
	out += fmt.Sprintf("app_mode %d\n", modeVal)

	// chaos_active
	out += "# HELP chaos_active Current chaos state (0=none, 1=slow, 2=error).\n"
	out += "# TYPE chaos_active gauge\n"
	chaosMu.RLock()
	chaosVal := 0
	if chaosSlowDur > 0 {
		chaosVal = 1
	} else if chaosErrRate > 0 {
		chaosVal = 2
	}
	chaosMu.RUnlock()
	out += fmt.Sprintf("chaos_active %d\n", chaosVal)

	return out
}

func formatFloat(f float64) string {
	if f == math.Trunc(f) {
		return strconv.FormatFloat(f, 'f', 1, 64)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

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
		"message":   fmt.Sprintf("Welcome to SwiftDeploy (%s mode)", mode),
		"mode":      mode,
		"version":   getVersion(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "ok",
		"mode":           getMode(),
		"uptime_seconds": time.Since(startTime).Seconds(),
	})
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(metrics.Render()))
}

func chaosHandler(w http.ResponseWriter, r *http.Request) {
	if getMode() != "canary" {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error": "chaos endpoint only available in canary mode",
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
		chaosErrRate = 0
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"chaos": map[string]interface{}{"mode": "slow", "duration": body.Duration, "rate": 0.0},
		})
	case "error":
		chaosErrRate = body.Rate
		chaosSlowDur = 0
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"chaos": map[string]interface{}{"mode": "error", "duration": 0, "rate": body.Rate},
		})
	case "recover":
		chaosSlowDur = 0
		chaosErrRate = 0
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"chaos": map[string]interface{}{"mode": nil, "duration": 0, "rate": 0.0},
		})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid chaos mode. Use: slow, error, or recover",
		})
	}
}

// ── Middleware ──

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(ww, r)
		duration := time.Since(start)
		path := r.URL.Path
		metrics.RecordRequest(r.Method, path, ww.status, duration)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func canaryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mode := getMode()

		if mode == "canary" {
			w.Header().Set("X-Mode", "canary")

			chaosMu.RLock()
			slowDur := chaosSlowDur
			errRate := chaosErrRate
			chaosMu.RUnlock()

			if errRate > 0 && rand.Float64() < errRate {
				writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "Chaos-induced server error",
				})
				return
			}

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
	r.Use(metricsMiddleware)
	r.Use(canaryMiddleware)

	r.Get("/", rootHandler)
	r.Get("/healthz", healthzHandler)
	r.Get("/metrics", metricsHandler)
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
