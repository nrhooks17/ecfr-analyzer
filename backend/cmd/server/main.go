package main

import (
	"log"
	"net/http"
	"time"

	"ecfr-analyzer/internal/database"
	"ecfr-analyzer/internal/handlers"
)

func main() {
	// Connect to database
	if err := database.Connect(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	// Start data loader
	// startDataLoader()

	// Set up routes
	mux := http.NewServeMux()
	
	// Health endpoint
	mux.HandleFunc("/health", handlers.HealthHandler)
	
	// Import endpoints
	mux.HandleFunc("/api/v1/import/agencies", handlers.ImportAgenciesHandler)
	mux.HandleFunc("/api/v1/import/titles", handlers.ImportTitlesHandler)
	mux.HandleFunc("/api/v1/import/historical-snapshots", handlers.ImportHistoricalSnapshotsHandler)
	
	// Status endpoint
	mux.HandleFunc("/api/v1/status", handlers.StatusHandler)
	
	// Retrieval endpoints
	mux.HandleFunc("/api/v1/agencies", handlers.AgenciesHandler)
	mux.HandleFunc("/api/v1/agencies/", handlers.AgencyDetailHandler)
	mux.HandleFunc("/api/v1/titles", handlers.TitlesHandler)
	
	// Metrics endpoints
	mux.HandleFunc("/api/v1/metrics/word-counts", handlers.WordCountMetricsHandler)
	mux.HandleFunc("/api/v1/metrics/checksums", handlers.ChecksumsHandler)
	mux.HandleFunc("/api/v1/metrics/agency-checksums", handlers.AgencyChecksumsHandler)
	mux.HandleFunc("/api/v1/metrics/history", handlers.HistoryHandler)
	
	// Export endpoints
	mux.HandleFunc("/api/v1/export/", handlers.ExportHandler)
	
	// Checksum calculation endpoint
	mux.HandleFunc("/api/v1/calculate-checksums", handlers.CalculateChecksumsHandler)

	// Apply middleware chain: logging -> CORS
	handler := loggingMiddleware(enableCORS(mux))

	// Start server
	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func startDataLoader() {
	importService := handlers.GetImportService()
	
	// Initial load on startup
	go func() {
		log.Println("Starting initial data load...")
		if err := importService.LoadAllData(); err != nil {
			log.Printf("Initial data load failed: %v", err)
		} else {
			log.Println("Initial data load completed successfully")
		}
	}()
	
	// Hourly refresh
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		
		for range ticker.C {
			log.Println("Starting scheduled data refresh...")
			if err := importService.LoadAllData(); err != nil {
				log.Printf("Scheduled data refresh failed: %v", err)
			} else {
				log.Println("Scheduled data refresh completed successfully")
			}
		}
	}()
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
		
		// Log request start
		log.Printf("[REQUEST] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		
		// Process request
		next.ServeHTTP(wrapped, r)
		
		// Log request completion with timing
		duration := time.Since(start)
		log.Printf("[RESPONSE] %s %s -> %d (%v)", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}