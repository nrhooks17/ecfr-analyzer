package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"ecfr-analyzer/internal/database"
	"ecfr-analyzer/internal/models"

	"github.com/google/uuid"
)

func main() {
	// Set up logging with colors
	log.SetFlags(log.LstdFlags)
	
	printStatus("Starting agency checksum calculation...")

	// Connect to database using the same logic as the main application
	printStatus("Connecting to database...")
	if err := database.Connect(); err != nil {
		printError(fmt.Sprintf("Failed to connect to database: %v", err))
		printError("Make sure the database is running and environment variables are set")
		printWarning("Check database configuration - same variables as main application")
		os.Exit(1)
	}
	defer database.Close()
	printSuccess("Database connection established")

	// Calculate checksums for all agencies
	startTime := time.Now()
	if err := calculateAllAgencyChecksums(); err != nil {
		printError(fmt.Sprintf("Failed to calculate checksums: %v", err))
		os.Exit(1)
	}
	
	duration := time.Since(startTime)
	printSuccess("Agency checksum calculation completed successfully!")
	printSuccess(fmt.Sprintf("Total time: %v", duration))
	printStatus("You can now load the dashboard - it should be much faster!")
}


// Color output functions
func printStatus(msg string) {
	fmt.Printf("\033[0;34m[INFO]\033[0m %s\n", msg)
}

func printSuccess(msg string) {
	fmt.Printf("\033[0;32m[SUCCESS]\033[0m %s\n", msg)
}

func printWarning(msg string) {
	fmt.Printf("\033[1;33m[WARNING]\033[0m %s\n", msg)
}

func printError(msg string) {
	fmt.Printf("\033[0;31m[ERROR]\033[0m %s\n", msg)
}

func calculateAllAgencyChecksums() error {
	// Get all agencies
	var agencies []models.Agency
	if err := database.DB.Find(&agencies).Error; err != nil {
		return fmt.Errorf("failed to fetch agencies: %w", err)
	}

	printStatus(fmt.Sprintf("Found %d agencies to process", len(agencies)))

	successCount := 0
	errorCount := 0
	skippedCount := 0

	// Process each agency
	for i, agency := range agencies {
		result, err := calculateAndStoreAgencyChecksum(agency.ID)
		if err != nil {
			printError(fmt.Sprintf("Failed to process %s: %v", agency.Name, err))
			errorCount++
			continue
		}
		
		switch result {
		case "created":
			successCount++
		case "updated": 
			successCount++
		case "skipped":
			skippedCount++
		}
		
		// Progress indicator
		if (i+1)%10 == 0 || i+1 == len(agencies) {
			printStatus(fmt.Sprintf("Progress: %d/%d agencies processed", i+1, len(agencies)))
		}
	}

	printSuccess(fmt.Sprintf("Calculation completed: %d created/updated, %d skipped (no change), %d errors", 
		successCount, skippedCount, errorCount))
	
	if errorCount > 0 {
		return fmt.Errorf("%d agencies failed to process", errorCount)
	}
	
	return nil
}

func calculateAndStoreAgencyChecksum(agencyID uuid.UUID) (string, error) {
	// Get all title checksums for this agency (using existing title_contents.checksum)
	type TitleChecksum struct {
		TitleNumber int
		Checksum    string
	}

	var titleChecksums []TitleChecksum
	err := database.DB.Table("title_contents tc").
		Select("t.number as title_number, tc.checksum").
		Joins("JOIN titles t ON tc.title_id = t.id").
		Joins("JOIN agency_cfr_references acr ON acr.title_id = t.id").
		Where("acr.agency_id = ? AND tc.checksum IS NOT NULL AND tc.checksum != ''", agencyID).
		Order("t.number ASC"). // Deterministic order
		Scan(&titleChecksums).Error

	if err != nil {
		return "", fmt.Errorf("failed to fetch title checksums: %w", err)
	}

	if len(titleChecksums) == 0 {
		// No content for this agency, skip
		return "skipped", nil
	}

	// Create deterministic content hash from title checksums
	var contentBuilder strings.Builder
	for _, tc := range titleChecksums {
		contentBuilder.WriteString(fmt.Sprintf("TITLE_%d:%s\n", tc.TitleNumber, tc.Checksum))
	}

	contentHash := fmt.Sprintf("%x", sha256.Sum256([]byte(contentBuilder.String())))
	
	// Create agency checksum from the combined content
	agencyChecksum := fmt.Sprintf("%x", sha256.Sum256([]byte(contentBuilder.String())))

	// Check if we need to update (content changed)
	var existingChecksum models.AgencyChecksum
	err = database.DB.Where("agency_id = ?", agencyID).First(&existingChecksum).Error
	
	if err == nil {
		// Record exists, check if content hash changed
		if existingChecksum.ContentHash == contentHash {
			// No change needed
			return "skipped", nil
		}
		
		// Update existing record
		existingChecksum.Checksum = agencyChecksum
		existingChecksum.ContentHash = contentHash
		existingChecksum.UpdatedAt = time.Now().UTC()
		
		if err := database.DB.Save(&existingChecksum).Error; err != nil {
			return "", err
		}
		return "updated", nil
	} else {
		// Create new record
		newChecksum := models.AgencyChecksum{
			AgencyID:    agencyID,
			Checksum:    agencyChecksum,
			ContentHash: contentHash,
			UpdatedAt:   time.Now().UTC(),
		}
		
		if err := database.DB.Create(&newChecksum).Error; err != nil {
			return "", err
		}
		return "created", nil
	}
}