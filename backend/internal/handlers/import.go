package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"ecfr-analyzer/internal/services"
)

var importService = services.NewImportService()
var historicalService = services.NewHistoricalService()

func ImportAgenciesHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HANDLER] ImportAgenciesHandler called")
	if r.Method != http.MethodPost {
		log.Printf("[HANDLER] ImportAgenciesHandler: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("[HANDLER] ImportAgenciesHandler: Starting agency import in background")
	go func() {
		if err := importService.ImportAgencies(); err != nil {
			log.Printf("[HANDLER] ImportAgenciesHandler: Agency import failed: %v", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"message": "Agency import started",
		"status":  "started",
	}
	json.NewEncoder(w).Encode(response)
	log.Printf("[HANDLER] ImportAgenciesHandler: Response sent")
}

func ImportTitlesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	go func() {
		if err := importService.ImportTitles(); err != nil {
			// Error is already logged in the service
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"message": "Title import started",
		"status":  "started",
	}
	json.NewEncoder(w).Encode(response)
}



func ImportHistoricalSnapshotsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	go func() {
		// First capture current snapshot
		if err := historicalService.CaptureSnapshot(); err != nil {
			// Error is already logged in the service
		}
		// Then import historical data from eCFR API
		if err := historicalService.ImportHistoricalData(); err != nil {
			// Error is already logged in the service
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"message": "Historical snapshots import started",
		"status":  "started",
	}
	json.NewEncoder(w).Encode(response)
}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	status := importService.GetStatus()
	json.NewEncoder(w).Encode(status)
}

// GetImportService returns the import service instance for use in main.go
func GetImportService() *services.ImportService {
	return importService
}