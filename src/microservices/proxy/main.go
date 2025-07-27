package main

import (
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Ака «feature‑flag» (0–100) — сколько % трафика /api/movies
// отправлять в новый movies‑service вместо старого монолита.
var migrationPercent int

// адреса бекендов внутри docker‑compose
const (
	moviesBackend    = "http://movies-service:8081"
	monolithBackend  = "http://monolith:8080"
	serviceListen    = ":8000" // наружный порт пробрасываем 8000:8000
	healthEndpoint   = "/health"
	forwardTimeout   = 10 * time.Second
)

func main() {
	// читаем % из env‑переменной (если пусто — 0)
	if p := os.Getenv("MOVIES_MIGRATION_PERCENT"); p != "" {
		if val, err := strconv.Atoi(p); err == nil {
			migrationPercent = val
		}
	}
	rand.Seed(time.Now().UnixNano())

	// /health
	http.HandleFunc(healthEndpoint, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"status":"ok"}`)
	})

	// все остальные запросы
	http.HandleFunc("/", proxyHandler)

	log.Printf("proxy started on %s | movies -> %d%% to new service\n",
		serviceListen, migrationPercent)
	log.Fatal(http.ListenAndServe(serviceListen, nil))
}

// proxyHandler решает, куда отправить запрос
func proxyHandler(w http.ResponseWriter, r *http.Request) {
	targetURL := monolithBackend

	if strings.HasPrefix(r.URL.Path, "/api/movies") {
		// Strangler Fig: часть трафика уходит в новый сервис
		if rand.Intn(100) < migrationPercent {
			targetURL = moviesBackend
		}
	}

	u, _ := url.Parse(targetURL)
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		log.Println("proxy error:", err)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"backend unavailable"}`))
	}
	proxy.ServeHTTP(w, r)
}
