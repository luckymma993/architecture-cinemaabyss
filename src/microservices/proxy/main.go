package main

import (
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

type Config struct {
	Port                   string
	MonolithURL            string
	MoviesServiceURL       string
	EventsServiceURL       string
	GradualMigration       bool
	MoviesMigrationPercent int
}

func loadConfig() Config {
	percentStr := getEnv("MOVIES_MIGRATION_PERCENT", "0")
	percent, err := strconv.Atoi(percentStr)
	if err != nil {
		percent = 0
	}

	return Config{
		Port:                   getEnv("PORT", "8000"),
		MonolithURL:            getEnv("MONOLITH_URL", "http://monolith:8080"),
		MoviesServiceURL:       getEnv("MOVIES_SERVICE_URL", "http://movies-service:8081"),
		EventsServiceURL:       getEnv("EVENTS_SERVICE_URL", "http://events-service:8082"),
		GradualMigration:       getEnv("GRADUAL_MIGRATION", "false") == "true",
		MoviesMigrationPercent: percent,
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func createProxy(targetURL string) *httputil.ReverseProxy {
	url, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("Proxy creation error %s: %v", targetURL, err)
	}
	return httputil.NewSingleHostReverseProxy(url)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	cfg := loadConfig()

	monolithProxy := createProxy(cfg.MonolithURL)
	moviesProxy := createProxy(cfg.MoviesServiceURL)
	eventsProxy := createProxy(cfg.EventsServiceURL)

	log.Printf("Starting proxy service on port %s", cfg.Port)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Strangler Fig Proxy is healthy"))
			return
		}

		if strings.HasPrefix(path, "/api/events") {
			log.Printf("[PROXY] Routing -> Events Service: %s", path)
			eventsProxy.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(path, "/api/movies") {
			if !cfg.GradualMigration {
				log.Printf("[PROXY] Routing -> Movies Service (100%%): %s", path)
				moviesProxy.ServeHTTP(w, r)
				return
			}

			chance := rand.Intn(100) + 1
			if chance <= cfg.MoviesMigrationPercent {
				log.Printf("[PROXY] Routing -> Movies Service (chance %d <= %d): %s", chance, cfg.MoviesMigrationPercent, path)
				moviesProxy.ServeHTTP(w, r)
			} else {
				log.Printf("[PROXY] Routing -> Monolith (chance %d > %d): %s", chance, cfg.MoviesMigrationPercent, path)
				monolithProxy.ServeHTTP(w, r)
			}
			return
		}

		log.Printf("[PROXY] Routing -> Monolith: %s", path)
		monolithProxy.ServeHTTP(w, r)
	})

	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatalf("Server startup error: %v", err)
	}
}
