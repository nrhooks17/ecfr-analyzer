package services

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"ecfr-analyzer/internal/database"
	"ecfr-analyzer/internal/models"
)

type HistoricalService struct {
	client *ECFRClient
}

func NewHistoricalService() *HistoricalService {
	return &HistoricalService{
		client: NewECFRClient(),
	}
}

// CaptureSnapshot captures current word counts and stores them as historical snapshots
func (h *HistoricalService) CaptureSnapshot() error {
	log.Println("Starting historical snapshot capture...")
	
	// Find the latest content date from title_contents table
	var latestContentDate time.Time
	err := database.DB.Table("title_contents").
		Select("MAX(content_date)").
		Scan(&latestContentDate).Error
	if err != nil {
		return fmt.Errorf("failed to find latest content date: %w", err)
	}
	
	snapshotDate := latestContentDate
	
	// Capture overall snapshot (no specific agency or title)
	if err := h.captureOverallSnapshot(snapshotDate); err != nil {
		log.Printf("Failed to capture overall snapshot: %v", err)
		return err
	}
	
	// Capture per-agency snapshots
	if err := h.captureAgencySnapshots(snapshotDate); err != nil {
		log.Printf("Failed to capture agency snapshots: %v", err)
		return err
	}
	
	// Capture per-title snapshots
	if err := h.captureTitleSnapshots(snapshotDate); err != nil {
		log.Printf("Failed to capture title snapshots: %v", err)
		return err
	}
	
	log.Println("Historical snapshot capture completed successfully")
	return nil
}

// captureOverallSnapshot captures total CFR word count
func (h *HistoricalService) captureOverallSnapshot(snapshotDate time.Time) error {
	var totalWords int64
	
	// Sum all word counts from title_contents for today's date
	err := database.DB.Table("title_contents").
		Where("content_date = ?", snapshotDate).
		Select("COALESCE(SUM(word_count), 0)").
		Scan(&totalWords).Error
	if err != nil {
		return err
	}
	
	log.Printf("Overall snapshot: %d total words", totalWords)
	
	// Create overall snapshot (no agency_id or title_id)
	snapshot := &models.HistoricalSnapshot{
		SnapshotDate: snapshotDate,
		WordCount:    &[]int{int(totalWords)}[0],
	}
	
	// Upsert snapshot
	err = database.DB.Where("snapshot_date = ? AND agency_id IS NULL AND title_id IS NULL", 
		snapshotDate).FirstOrCreate(snapshot).Error
	if err != nil {
		return err
	}
	
	return nil
}

// captureAgencySnapshots captures word counts per agency
func (h *HistoricalService) captureAgencySnapshots(snapshotDate time.Time) error {
	log.Println("Capturing per-agency snapshots...")
	
	// Query to get word count per agency
	type AgencyWordCount struct {
		AgencyID  uuid.UUID
		WordCount int64
	}
	
	var agencyWordCounts []AgencyWordCount
	
	// Join agencies -> agency_cfr_references -> titles -> title_contents
	// Sum word counts per agency
	err := database.DB.Table("agencies a").
		Select("a.id as agency_id, COALESCE(SUM(tc.word_count), 0) as word_count").
		Joins("LEFT JOIN agency_cfr_references acr ON a.id = acr.agency_id").
		Joins("LEFT JOIN title_contents tc ON acr.title_id = tc.title_id AND tc.content_date = ?", snapshotDate).
		Group("a.id").
		Scan(&agencyWordCounts).Error
	if err != nil {
		return err
	}
	
	log.Printf("Found %d agencies to snapshot", len(agencyWordCounts))
	
	// Create snapshots for each agency
	for _, awc := range agencyWordCounts {
		if awc.WordCount == 0 {
			continue // Skip agencies with no content
		}
		
		snapshot := &models.HistoricalSnapshot{
			SnapshotDate: snapshotDate,
			AgencyID:     &awc.AgencyID,
			WordCount:    &[]int{int(awc.WordCount)}[0],
		}
		
		// Upsert snapshot
		err = database.DB.Where("snapshot_date = ? AND agency_id = ? AND title_id IS NULL", 
			snapshotDate, awc.AgencyID).FirstOrCreate(snapshot).Error
		if err != nil {
			log.Printf("Error creating agency snapshot for %s: %v", awc.AgencyID, err)
			continue
		}
	}
	
	return nil
}

// captureTitleSnapshots captures word counts per title
func (h *HistoricalService) captureTitleSnapshots(snapshotDate time.Time) error {
	log.Println("Capturing per-title snapshots...")
	
	// Get all title contents for today
	var titleContents []models.TitleContent
	err := database.DB.Where("content_date = ?", snapshotDate).Find(&titleContents).Error
	if err != nil {
		return err
	}
	
	log.Printf("Found %d titles to snapshot", len(titleContents))
	
	// Create snapshots for each title
	for _, tc := range titleContents {
		if tc.WordCount == nil || *tc.WordCount == 0 {
			continue // Skip titles with no word count
		}
		
		snapshot := &models.HistoricalSnapshot{
			SnapshotDate: snapshotDate,
			TitleID:      &tc.TitleID,
			WordCount:    tc.WordCount,
			Checksum:     tc.Checksum,
		}
		
		// Upsert snapshot
		err = database.DB.Where("snapshot_date = ? AND title_id = ? AND agency_id IS NULL", 
			snapshotDate, tc.TitleID).FirstOrCreate(snapshot).Error
		if err != nil {
			log.Printf("Error creating title snapshot for %s: %v", tc.TitleID, err)
			continue
		}
	}
	
	return nil
}

// ImportHistoricalData imports historical data from eCFR API for the past 2 years
func (h *HistoricalService) ImportHistoricalData() error {
	log.Println("Starting historical data import from eCFR API...")
	
	// Get all active titles from database
	var titles []models.Title
	err := database.DB.Raw("SELECT * FROM titles WHERE reserved = false").Scan(&titles).Error
	if err != nil {
		return fmt.Errorf("failed to fetch titles: %w", err)
	}
	
	log.Printf("Found %d active titles to import historical data for", len(titles))
	
	// Generate monthly snapshots for the past 24 months
	now := time.Now().UTC()
	for monthsBack := 1; monthsBack <= 24; monthsBack++ {
		snapshotDate := now.AddDate(0, -monthsBack, 0)
		snapshotDate = time.Date(snapshotDate.Year(), snapshotDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		
		log.Printf("Processing historical data for %s", snapshotDate.Format("2006-01"))
		
		// Skip if we already have data for this month
		var existingCount int64
		database.DB.Model(&models.HistoricalSnapshot{}).Where("snapshot_date = ?", snapshotDate).Count(&existingCount)
		if existingCount > 0 {
			log.Printf("Skipping %s - data already exists", snapshotDate.Format("2006-01"))
			continue
		}
		
		// Import historical snapshots for this date
		if err := h.importSnapshotsForDate(titles, snapshotDate); err != nil {
			log.Printf("Error importing snapshots for %s: %v", snapshotDate.Format("2006-01"), err)
			continue
		}
		
		// Add delay to avoid overwhelming the API
		time.Sleep(500 * time.Millisecond)
	}
	
	log.Println("Historical data import completed")
	return nil
}

// importSnapshotsForDate imports historical snapshots for a specific date
func (h *HistoricalService) importSnapshotsForDate(titles []models.Title, snapshotDate time.Time) error {
	dateStr := snapshotDate.Format("2006-01-02")
	totalWords := int64(0)
	validTitles := 0
	
	log.Printf("Importing historical data for %d titles on %s", len(titles), dateStr)
	
	// Process each title
	for _, title := range titles {
		// Fetch historical structure data from eCFR API
		structureData, err := h.client.FetchTitleStructure(title.Number, dateStr)
		if err != nil {
			log.Printf("Failed to fetch structure for title %d on %s: %v", title.Number, dateStr, err)
			continue
		}
		
		// Extract size (character count) from structure data
		charCount := structureData.Size
		if charCount == 0 {
			continue // Skip titles with no content
		}
		
		// Estimate word count from character count (roughly 5 chars per word)
		estimatedWordCount := int(charCount / 5)
		totalWords += int64(estimatedWordCount)
		validTitles++
		
		// Create title snapshot
		titleSnapshot := &models.HistoricalSnapshot{
			SnapshotDate: snapshotDate,
			TitleID:      &title.ID,
			WordCount:    &estimatedWordCount,
		}
		
		// Store title snapshot
		err = database.DB.Where("snapshot_date = ? AND title_id = ? AND agency_id IS NULL",
			snapshotDate, title.ID).FirstOrCreate(titleSnapshot).Error
		if err != nil {
			log.Printf("Error creating title snapshot for %d on %s: %v", title.Number, dateStr, err)
		}
		
		// Small delay to avoid overwhelming API
		time.Sleep(100 * time.Millisecond)
	}
	
	log.Printf("Processed %d valid titles with %d total estimated words for %s", validTitles, totalWords, dateStr)
	
	// Create overall snapshot (total across all titles)
	if totalWords > 0 {
		overallSnapshot := &models.HistoricalSnapshot{
			SnapshotDate: snapshotDate,
			WordCount:    &[]int{int(totalWords)}[0],
		}
		
		err := database.DB.Where("snapshot_date = ? AND agency_id IS NULL AND title_id IS NULL",
			snapshotDate).FirstOrCreate(overallSnapshot).Error
		if err != nil {
			log.Printf("Error creating overall snapshot for %s: %v", dateStr, err)
		}
	}
	
	// Create agency snapshots by aggregating title data
	return h.createAgencySnapshotsFromTitles(snapshotDate)
}

// createAgencySnapshotsFromTitles creates agency snapshots by aggregating title snapshots
func (h *HistoricalService) createAgencySnapshotsFromTitles(snapshotDate time.Time) error {
	type AgencyWordCount struct {
		AgencyID  uuid.UUID
		WordCount int64
	}
	
	var agencyWordCounts []AgencyWordCount
	
	// Join agencies -> agency_cfr_references -> historical_snapshots (title-based)
	// Sum word counts per agency for this snapshot date
	err := database.DB.Table("agencies a").
		Select("a.id as agency_id, COALESCE(SUM(hs.word_count), 0) as word_count").
		Joins("LEFT JOIN agency_cfr_references acr ON a.id = acr.agency_id").
		Joins("LEFT JOIN historical_snapshots hs ON acr.title_id = hs.title_id AND hs.snapshot_date = ? AND hs.agency_id IS NULL", snapshotDate).
		Group("a.id").
		Having("COALESCE(SUM(hs.word_count), 0) > 0").
		Scan(&agencyWordCounts).Error
	if err != nil {
		return err
	}
	
	log.Printf("Creating agency snapshots for %d agencies on %s", len(agencyWordCounts), snapshotDate.Format("2006-01-02"))
	
	// Create snapshots for each agency
	for _, awc := range agencyWordCounts {
		snapshot := &models.HistoricalSnapshot{
			SnapshotDate: snapshotDate,
			AgencyID:     &awc.AgencyID,
			WordCount:    &[]int{int(awc.WordCount)}[0],
		}
		
		err = database.DB.Where("snapshot_date = ? AND agency_id = ? AND title_id IS NULL",
			snapshotDate, awc.AgencyID).FirstOrCreate(snapshot).Error
		if err != nil {
			log.Printf("Error creating agency snapshot for %s on %s: %v", awc.AgencyID, snapshotDate.Format("2006-01-02"), err)
		}
	}
	
	return nil
}