package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ecfr-analyzer/internal/database"
	"ecfr-analyzer/internal/models"

	"github.com/google/uuid"
)

type APIResponse struct {
	Data interface{} `json:"data"`
	Meta Meta        `json:"meta"`
}

type Meta struct {
	Total       int       `json:"total"`
	LastUpdated time.Time `json:"lastUpdated"`
}

type AgencyWithMetrics struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Slug           string    `json:"slug"`
	WordCount      int       `json:"wordCount"`
	PercentOfTotal float64   `json:"percentOfTotal"`
	TitleCount     int       `json:"titleCount"`
	Checksum       *string   `json:"checksum,omitempty"`
	ParentID       *uuid.UUID `json:"parentId,omitempty"`
}

type AgencyDetail struct {
	AgencyWithMetrics
	SubAgencies []AgencyWithMetrics `json:"subAgencies"`
	TitleBreakdown []TitleBreakdown `json:"titleBreakdown"`
}

type TitleBreakdown struct {
	TitleNumber int    `json:"titleNumber"`
	TitleName   string `json:"titleName"`
	WordCount   int    `json:"wordCount"`
}

type TitleWithMetrics struct {
	ID               uuid.UUID  `json:"id"`
	Number           int        `json:"number"`
	Name             string     `json:"name"`
	WordCount        int        `json:"wordCount"`
	Checksum         *string    `json:"checksum,omitempty"`
	LatestAmendedOn  *time.Time `json:"latestAmendedOn,omitempty"`
	UpToDateAsOf     *time.Time `json:"upToDateAsOf,omitempty"`
}

type WordCountMetrics struct {
	TotalCFRWords int                 `json:"totalCFRWords"`
	Agencies      []AgencyWithMetrics `json:"agencies"`
}

type ChecksumInfo struct {
	TitleNumber int       `json:"titleNumber"`
	TitleName   string    `json:"titleName"`
	Checksum    *string   `json:"checksum"`
	LastChanged time.Time `json:"lastChanged"`
}

type HistoricalPoint struct {
	Date          string  `json:"date"`
	WordCount     int     `json:"wordCount"`
	ChangePercent float64 `json:"changePercent"`
}

// getCachedAgencyChecksums retrieves checksums from cache, with fallback to real-time calculation
func getCachedAgencyChecksums(agencyIDs []uuid.UUID) map[uuid.UUID]string {
	if len(agencyIDs) == 0 {
		return make(map[uuid.UUID]string)
	}
	
	// First, try to get cached checksums
	var cachedChecksums []models.AgencyChecksum
	err := database.DB.Where("agency_id IN ?", agencyIDs).Find(&cachedChecksums).Error
	if err != nil {
		log.Printf("Warning: Failed to fetch cached checksums: %v", err)
		return calculateBatchAgencyChecksumsLegacy(agencyIDs)
	}
	
	// Map cached results
	result := make(map[uuid.UUID]string)
	foundIDs := make(map[uuid.UUID]bool)
	
	for _, cached := range cachedChecksums {
		result[cached.AgencyID] = cached.Checksum
		foundIDs[cached.AgencyID] = true
	}
	
	// Calculate missing checksums using optimized method
	var missingIDs []uuid.UUID
	for _, agencyID := range agencyIDs {
		if !foundIDs[agencyID] {
			missingIDs = append(missingIDs, agencyID)
		}
	}
	
	if len(missingIDs) > 0 {
		log.Printf("Warning: %d agency checksums not found in cache, calculating real-time", len(missingIDs))
		missingChecksums := calculateBatchAgencyChecksumsOptimized(missingIDs)
		for agencyID, checksum := range missingChecksums {
			result[agencyID] = checksum
		}
	}
	
	return result
}

// calculateBatchAgencyChecksumsOptimized uses title checksums instead of full XML content
func calculateBatchAgencyChecksumsOptimized(agencyIDs []uuid.UUID) map[uuid.UUID]string {
	if len(agencyIDs) == 0 {
		return make(map[uuid.UUID]string)
	}
	
	type AgencyTitleChecksum struct {
		AgencyID    uuid.UUID `gorm:"column:agency_id"`
		TitleNumber int       `gorm:"column:title_number"`
		Checksum    string    `gorm:"column:checksum"`
	}
	
	var agencyTitleChecksums []AgencyTitleChecksum
	err := database.DB.Table("title_contents tc").
		Select("acr.agency_id, t.number as title_number, tc.checksum").
		Joins("JOIN titles t ON tc.title_id = t.id").
		Joins("JOIN agency_cfr_references acr ON t.id = acr.title_id").
		Where("acr.agency_id IN ? AND tc.checksum IS NOT NULL AND tc.checksum != ''", agencyIDs).
		Order("acr.agency_id ASC, t.number ASC"). // Deterministic order
		Scan(&agencyTitleChecksums).Error
	
	if err != nil {
		log.Printf("Error fetching title checksums: %v", err)
		return make(map[uuid.UUID]string)
	}
	
	// Group checksums by agency
	agencyChecksumsMap := make(map[uuid.UUID][]AgencyTitleChecksum)
	for _, content := range agencyTitleChecksums {
		agencyChecksumsMap[content.AgencyID] = append(agencyChecksumsMap[content.AgencyID], content)
	}
	
	checksums := make(map[uuid.UUID]string)
	for agencyID, titleChecksums := range agencyChecksumsMap {
		if len(titleChecksums) == 0 {
			continue
		}
		
		// Combine title checksums in deterministic order (much faster than XML content)
		var combinedChecksums strings.Builder
		for _, tc := range titleChecksums {
			combinedChecksums.WriteString(fmt.Sprintf("TITLE_%d:%s\n", tc.TitleNumber, tc.Checksum))
		}
		
		// Calculate SHA-256 checksum
		hash := sha256.Sum256([]byte(combinedChecksums.String()))
		checksums[agencyID] = fmt.Sprintf("%x", hash)
	}
	
	return checksums
}

// calculateBatchAgencyChecksumsLegacy - fallback method using full XML content (kept for compatibility)
func calculateBatchAgencyChecksumsLegacy(agencyIDs []uuid.UUID) map[uuid.UUID]string {
	if len(agencyIDs) == 0 {
		return make(map[uuid.UUID]string)
	}
	
	// Limit to prevent memory issues
	if len(agencyIDs) > 10 {
		log.Printf("Warning: Legacy checksum calculation limited to first 10 agencies to prevent memory issues")
		agencyIDs = agencyIDs[:10]
	}
	
	type AgencyTitleContent struct {
		AgencyID    uuid.UUID `gorm:"column:agency_id"`
		TitleNumber int       `gorm:"column:title_number"`
		Content     string    `gorm:"column:xml_content"`
	}
	
	var agencyTitleContents []AgencyTitleContent
	err := database.DB.Table("title_contents tc").
		Select("acr.agency_id, t.number as title_number, tc.xml_content").
		Joins("JOIN titles t ON tc.title_id = t.id").
		Joins("JOIN agency_cfr_references acr ON t.id = acr.title_id").
		Where("acr.agency_id IN ? AND tc.xml_content IS NOT NULL AND tc.xml_content != ''", agencyIDs).
		Order("acr.agency_id ASC, t.number ASC"). // Deterministic order
		Scan(&agencyTitleContents).Error
	
	if err != nil {
		return make(map[uuid.UUID]string)
	}
	
	// Group content by agency and calculate checksums
	agencyContentMap := make(map[uuid.UUID][]AgencyTitleContent)
	for _, content := range agencyTitleContents {
		agencyContentMap[content.AgencyID] = append(agencyContentMap[content.AgencyID], content)
	}
	
	checksums := make(map[uuid.UUID]string)
	for agencyID, contents := range agencyContentMap {
		if len(contents) == 0 {
			continue
		}
		
		// Concatenate all content in deterministic order
		var combinedContent strings.Builder
		for _, tc := range contents {
			combinedContent.WriteString(fmt.Sprintf("TITLE_%d:", tc.TitleNumber))
			combinedContent.WriteString(tc.Content)
			combinedContent.WriteString("\n")
		}
		
		// Calculate SHA-256 checksum
		hash := sha256.Sum256([]byte(combinedContent.String()))
		checksums[agencyID] = fmt.Sprintf("%x", hash)
	}
	
	return checksums
}

// calculateAgencyChecksum calculates checksum for a single agency (fallback for individual calls)
func calculateAgencyChecksum(agencyID uuid.UUID) string {
	checksums := getCachedAgencyChecksums([]uuid.UUID{agencyID})
	if checksum, exists := checksums[agencyID]; exists {
		return checksum
	}
	return ""
}

func AgenciesHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HANDLER] AgenciesHandler called")
	if r.Method != http.MethodGet {
		log.Printf("[HANDLER] AgenciesHandler: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Calculate total words for percentage calculation
	var totalWords int64
	database.DB.Table("title_contents").
		Select("COALESCE(SUM(word_count), 0)").
		Where("word_count IS NOT NULL").
		Scan(&totalWords)

	// Get all agencies with their metrics in a single optimized query
	type AgencyMetrics struct {
		ID         string
		Name       string
		Slug       string
		ParentID   *string
		WordCount  int64
		TitleCount int64
	}
	
	var agencyMetrics []AgencyMetrics
	err := database.DB.Raw(`
		SELECT 
			a.id,
			a.name,
			a.slug,
			a.parent_id,
			COALESCE(SUM(tc.word_count), 0) as word_count,
			COUNT(DISTINCT acr.title_id) as title_count
		FROM agencies a
		LEFT JOIN agency_cfr_references acr ON a.id = acr.agency_id
		LEFT JOIN title_contents tc ON acr.title_id = tc.title_id AND tc.word_count IS NOT NULL
		GROUP BY a.id, a.name, a.slug, a.parent_id
		ORDER BY word_count DESC
	`).Scan(&agencyMetrics).Error
	
	if err != nil {
		http.Error(w, "Failed to fetch agencies", http.StatusInternalServerError)
		return
	}

	// Collect agency IDs for batch checksum calculation
	agencyIDs := make([]uuid.UUID, len(agencyMetrics))
	for i, metrics := range agencyMetrics {
		agencyIDs[i] = uuid.MustParse(metrics.ID)
	}
	
	// Calculate all checksums in a single batch operation
	checksums := getCachedAgencyChecksums(agencyIDs)

	// Build response with calculated metrics
	var agenciesWithMetrics []AgencyWithMetrics
	for _, metrics := range agencyMetrics {
		var parentID *uuid.UUID
		if metrics.ParentID != nil {
			if parsed, err := uuid.Parse(*metrics.ParentID); err == nil {
				parentID = &parsed
			}
		}

		percentOfTotal := float64(0)
		if totalWords > 0 {
			percentOfTotal = float64(metrics.WordCount) / float64(totalWords) * 100
		}

		// Get checksum from batch calculation
		var checksum *string
		agencyID := uuid.MustParse(metrics.ID)
		if checksumValue, exists := checksums[agencyID]; exists && checksumValue != "" {
			checksum = &checksumValue
		}

		agenciesWithMetrics = append(agenciesWithMetrics, AgencyWithMetrics{
			ID:             agencyID,
			Name:           metrics.Name,
			Slug:           metrics.Slug,
			WordCount:      int(metrics.WordCount),
			PercentOfTotal: percentOfTotal,
			TitleCount:     int(metrics.TitleCount),
			Checksum:       checksum,
			ParentID:       parentID,
		})
	}

	response := APIResponse{
		Data: agenciesWithMetrics,
		Meta: Meta{
			Total:       len(agenciesWithMetrics),
			LastUpdated: time.Now(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func AgencyDetailHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HANDLER] AgencyDetailHandler called")
	if r.Method != http.MethodGet {
		log.Printf("[HANDLER] AgencyDetailHandler: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract slug from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/agencies/")
	slug := strings.Split(path, "/")[0]

	var agency models.Agency
	if err := database.DB.Where("slug = ?", slug).First(&agency).Error; err != nil {
		http.Error(w, "Agency not found", http.StatusNotFound)
		return
	}

	// Get agency metrics in a single query
	type MainAgencyMetrics struct {
		WordCount  int64
		TitleCount int64
	}
	
	var metrics MainAgencyMetrics
	database.DB.Raw(`
		SELECT 
			COALESCE(SUM(tc.word_count), 0) as word_count,
			COUNT(DISTINCT acr.title_id) as title_count
		FROM agency_cfr_references acr
		LEFT JOIN title_contents tc ON acr.title_id = tc.title_id AND tc.word_count IS NOT NULL
		WHERE acr.agency_id = ?
	`, agency.ID).Scan(&metrics)

	// Get sub-agencies with their metrics in one query
	type SubAgencyMetrics struct {
		ID         string
		Name       string
		Slug       string
		ParentID   *string
		WordCount  int64
		TitleCount int64
	}
	
	var subAgenciesMetrics []SubAgencyMetrics
	database.DB.Raw(`
		SELECT 
			a.id,
			a.name,
			a.slug,
			a.parent_id,
			COALESCE(SUM(tc.word_count), 0) as word_count,
			COUNT(DISTINCT acr.title_id) as title_count
		FROM agencies a
		LEFT JOIN agency_cfr_references acr ON a.id = acr.agency_id
		LEFT JOIN title_contents tc ON acr.title_id = tc.title_id AND tc.word_count IS NOT NULL
		WHERE a.parent_id = ?
		GROUP BY a.id, a.name, a.slug, a.parent_id
	`, agency.ID).Scan(&subAgenciesMetrics)

	// Collect sub-agency IDs for batch checksum calculation
	var subAgencyIDs []uuid.UUID
	for _, subMetrics := range subAgenciesMetrics {
		if subAgencyID, err := uuid.Parse(subMetrics.ID); err == nil {
			subAgencyIDs = append(subAgencyIDs, subAgencyID)
		}
	}
	
	// Calculate checksums for all sub-agencies in batch
	subChecksums := getCachedAgencyChecksums(subAgencyIDs)

	var subAgenciesWithMetrics []AgencyWithMetrics
	for _, subMetrics := range subAgenciesMetrics {
		var parentID *uuid.UUID
		if subMetrics.ParentID != nil {
			if parsed, err := uuid.Parse(*subMetrics.ParentID); err == nil {
				parentID = &parsed
			}
		}
		
		// Get checksum from batch calculation
		var subChecksum *string
		subAgencyID := uuid.MustParse(subMetrics.ID)
		if checksumValue, exists := subChecksums[subAgencyID]; exists && checksumValue != "" {
			subChecksum = &checksumValue
		}

		subAgenciesWithMetrics = append(subAgenciesWithMetrics, AgencyWithMetrics{
			ID:         subAgencyID,
			Name:       subMetrics.Name,
			Slug:       subMetrics.Slug,
			WordCount:  int(subMetrics.WordCount),
			TitleCount: int(subMetrics.TitleCount),
			Checksum:   subChecksum,
			ParentID:   parentID,
		})
	}

	// Get title breakdown
	var titleBreakdowns []TitleBreakdown
	rows, err := database.DB.Table("title_contents").
		Select("titles.number, titles.name, COALESCE(SUM(title_contents.word_count), 0) as word_count").
		Joins("JOIN titles ON titles.id = title_contents.title_id").
		Joins("JOIN agency_cfr_references ON agency_cfr_references.title_id = titles.id").
		Where("agency_cfr_references.agency_id = ? AND title_contents.word_count IS NOT NULL", agency.ID).
		Group("titles.number, titles.name").
		Rows()

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var breakdown TitleBreakdown
			rows.Scan(&breakdown.TitleNumber, &breakdown.TitleName, &breakdown.WordCount)
			titleBreakdowns = append(titleBreakdowns, breakdown)
		}
	}

	// Calculate checksum for this agency
	var checksum *string
	if checksumValue := calculateAgencyChecksum(agency.ID); checksumValue != "" {
		checksum = &checksumValue
	}

	agencyDetail := AgencyDetail{
		AgencyWithMetrics: AgencyWithMetrics{
			ID:         agency.ID,
			Name:       agency.Name,
			Slug:       agency.Slug,
			WordCount:  int(metrics.WordCount),
			TitleCount: int(metrics.TitleCount),
			Checksum:   checksum,
			ParentID:   agency.ParentID,
		},
		SubAgencies:    subAgenciesWithMetrics,
		TitleBreakdown: titleBreakdowns,
	}

	response := APIResponse{
		Data: agencyDetail,
		Meta: Meta{
			Total:       1,
			LastUpdated: time.Now(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func TitlesHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HANDLER] TitlesHandler called")
	if r.Method != http.MethodGet {
		log.Printf("[HANDLER] TitlesHandler: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var titles []models.Title
	var titlesWithMetrics []TitleWithMetrics

	if err := database.DB.Order("number").Find(&titles).Error; err != nil {
		http.Error(w, "Failed to fetch titles", http.StatusInternalServerError)
		return
	}

	// Get latest content for all titles in one query
	type TitleMetrics struct {
		TitleID   string
		WordCount *int
		Checksum  *string
	}
	
	var titleMetrics []TitleMetrics
	database.DB.Raw(`
		SELECT DISTINCT ON (tc.title_id) 
			tc.title_id, 
			tc.word_count, 
			tc.checksum 
		FROM title_contents tc 
		ORDER BY tc.title_id, tc.content_date DESC
	`).Scan(&titleMetrics)
	
	// Create a map for quick lookup
	metricsMap := make(map[string]TitleMetrics)
	for _, metric := range titleMetrics {
		metricsMap[metric.TitleID] = metric
	}

	for _, title := range titles {
		var wordCount int64
		var checksum *string
		
		if metrics, exists := metricsMap[title.ID.String()]; exists {
			if metrics.WordCount != nil {
				wordCount = int64(*metrics.WordCount)
			}
			checksum = metrics.Checksum
		}

		titlesWithMetrics = append(titlesWithMetrics, TitleWithMetrics{
			ID:              title.ID,
			Number:          title.Number,
			Name:            title.Name,
			WordCount:       int(wordCount),
			Checksum:        checksum,
			LatestAmendedOn: title.LatestAmendedOn,
			UpToDateAsOf:    title.UpToDateAsOf,
		})
	}

	response := APIResponse{
		Data: titlesWithMetrics,
		Meta: Meta{
			Total:       len(titlesWithMetrics),
			LastUpdated: time.Now(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func WordCountMetricsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HANDLER] WordCountMetricsHandler called")
	if r.Method != http.MethodGet {
		log.Printf("[HANDLER] WordCountMetricsHandler: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Calculate total CFR words
	var totalWords int64
	database.DB.Table("title_contents").
		Select("COALESCE(SUM(word_count), 0)").
		Where("word_count IS NOT NULL").
		Scan(&totalWords)

	// Reuse optimized agencies query from AgenciesHandler
	type AgencyMetrics struct {
		ID         string
		Name       string
		Slug       string
		ParentID   *string
		WordCount  int64
		TitleCount int64
	}
	
	var agencyMetrics []AgencyMetrics
	err := database.DB.Raw(`
		SELECT 
			a.id,
			a.name,
			a.slug,
			a.parent_id,
			COALESCE(SUM(tc.word_count), 0) as word_count,
			COUNT(DISTINCT acr.title_id) as title_count
		FROM agencies a
		LEFT JOIN agency_cfr_references acr ON a.id = acr.agency_id
		LEFT JOIN title_contents tc ON acr.title_id = tc.title_id AND tc.word_count IS NOT NULL
		GROUP BY a.id, a.name, a.slug, a.parent_id
		ORDER BY word_count DESC
	`).Scan(&agencyMetrics).Error
	
	if err != nil {
		http.Error(w, "Failed to fetch agencies", http.StatusInternalServerError)
		return
	}

	// Collect agency IDs for batch checksum calculation
	agencyIDs := make([]uuid.UUID, len(agencyMetrics))
	for i, metrics := range agencyMetrics {
		agencyIDs[i] = uuid.MustParse(metrics.ID)
	}
	
	// Calculate all checksums in a single batch operation
	checksums := getCachedAgencyChecksums(agencyIDs)

	// Build response with calculated metrics
	var agenciesWithMetrics []AgencyWithMetrics
	for _, metrics := range agencyMetrics {
		var parentID *uuid.UUID
		if metrics.ParentID != nil {
			if parsed, err := uuid.Parse(*metrics.ParentID); err == nil {
				parentID = &parsed
			}
		}

		percentOfTotal := float64(0)
		if totalWords > 0 {
			percentOfTotal = float64(metrics.WordCount) / float64(totalWords) * 100
		}

		// Get checksum from batch calculation
		var checksum *string
		agencyID := uuid.MustParse(metrics.ID)
		if checksumValue, exists := checksums[agencyID]; exists && checksumValue != "" {
			checksum = &checksumValue
		}

		agenciesWithMetrics = append(agenciesWithMetrics, AgencyWithMetrics{
			ID:             agencyID,
			Name:           metrics.Name,
			Slug:           metrics.Slug,
			WordCount:      int(metrics.WordCount),
			PercentOfTotal: percentOfTotal,
			TitleCount:     int(metrics.TitleCount),
			Checksum:       checksum,
			ParentID:       parentID,
		})
	}

	wordCountMetrics := WordCountMetrics{
		TotalCFRWords: int(totalWords),
		Agencies:      agenciesWithMetrics,
	}

	response := APIResponse{
		Data: wordCountMetrics,
		Meta: Meta{
			Total:       len(agenciesWithMetrics),
			LastUpdated: time.Now(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func ChecksumsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var checksumInfos []ChecksumInfo

	rows, err := database.DB.Table("title_contents").
		Select("titles.number, titles.name, title_contents.checksum, title_contents.created_at").
		Joins("JOIN titles ON titles.id = title_contents.title_id").
		Where("title_contents.checksum IS NOT NULL").
		Order("titles.number").
		Rows()

	if err != nil {
		http.Error(w, "Failed to fetch checksums", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var info ChecksumInfo
		rows.Scan(&info.TitleNumber, &info.TitleName, &info.Checksum, &info.LastChanged)
		checksumInfos = append(checksumInfos, info)
	}

	response := APIResponse{
		Data: checksumInfos,
		Meta: Meta{
			Total:       len(checksumInfos),
			LastUpdated: time.Now(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type AgencyChecksumInfo struct {
	AgencyID     string  `json:"agencyId"`
	AgencyName   string  `json:"agencyName"`
	AgencySlug   string  `json:"agencySlug"`
	Checksum     *string `json:"checksum"`
	WordCount    int     `json:"wordCount"`
	TitleCount   int     `json:"titleCount"`
	LastChanged  time.Time `json:"lastChanged"`
}

func AgencyChecksumsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Reuse optimized agencies query to get agencies with metrics
	type AgencyMetrics struct {
		ID         string
		Name       string
		Slug       string
		WordCount  int64
		TitleCount int64
	}
	
	var agencyMetrics []AgencyMetrics
	err := database.DB.Raw(`
		SELECT 
			a.id,
			a.name,
			a.slug,
			COALESCE(SUM(tc.word_count), 0) as word_count,
			COUNT(DISTINCT acr.title_id) as title_count
		FROM agencies a
		LEFT JOIN agency_cfr_references acr ON a.id = acr.agency_id
		LEFT JOIN title_contents tc ON acr.title_id = tc.title_id AND tc.word_count IS NOT NULL
		GROUP BY a.id, a.name, a.slug
		HAVING COALESCE(SUM(tc.word_count), 0) > 0
		ORDER BY word_count DESC
	`).Scan(&agencyMetrics).Error
	
	if err != nil {
		http.Error(w, "Failed to fetch agencies", http.StatusInternalServerError)
		return
	}

	// Collect agency IDs for batch checksum calculation
	agencyIDs := make([]uuid.UUID, len(agencyMetrics))
	for i, metrics := range agencyMetrics {
		agencyIDs[i] = uuid.MustParse(metrics.ID)
	}
	
	// Calculate all checksums in a single batch operation
	checksums := getCachedAgencyChecksums(agencyIDs)

	var agencyChecksumInfos []AgencyChecksumInfo
	for _, metrics := range agencyMetrics {
		// Get checksum from batch calculation
		var checksum *string
		agencyID := uuid.MustParse(metrics.ID)
		if checksumValue, exists := checksums[agencyID]; exists && checksumValue != "" {
			checksum = &checksumValue
		}

		agencyChecksumInfos = append(agencyChecksumInfos, AgencyChecksumInfo{
			AgencyID:    metrics.ID,
			AgencyName:  metrics.Name,
			AgencySlug:  metrics.Slug,
			Checksum:    checksum,
			WordCount:   int(metrics.WordCount),
			TitleCount:  int(metrics.TitleCount),
			LastChanged: time.Now(), // Would be actual last modified time in real implementation
		})
	}

	response := APIResponse{
		Data: agencyChecksumInfos,
		Meta: Meta{
			Total:       len(agencyChecksumInfos),
			LastUpdated: time.Now(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func HistoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	agencySlug := r.URL.Query().Get("agency")
	monthsStr := r.URL.Query().Get("months")
	
	months := 12 // default to 12 months
	if monthsStr != "" {
		if m, err := strconv.Atoi(monthsStr); err == nil && m > 0 {
			months = m
		}
	}

	// Calculate date range
	endDate := time.Now().UTC()
	startDate := endDate.AddDate(0, -months, 0)

	var history []HistoricalPoint
	var err error

	if agencySlug != "" {
		// Get history for specific agency
		history, err = getAgencyHistory(agencySlug, startDate, endDate)
	} else {
		// Get overall CFR history
		history, err = getOverallHistory(startDate, endDate)
	}

	if err != nil {
		http.Error(w, "Failed to fetch historical data", http.StatusInternalServerError)
		return
	}

	response := APIResponse{
		Data: history,
		Meta: Meta{
			Total:       len(history),
			LastUpdated: time.Now(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getOverallHistory retrieves overall CFR word count history
func getOverallHistory(startDate, endDate time.Time) ([]HistoricalPoint, error) {
	type SnapshotData struct {
		SnapshotDate time.Time
		WordCount    int
	}

	var snapshots []SnapshotData
	
	// Query historical snapshots for overall data (no agency_id or title_id)
	err := database.DB.Table("historical_snapshots").
		Select("snapshot_date, word_count").
		Where("snapshot_date >= ? AND snapshot_date <= ?", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).
		Where("agency_id IS NULL AND title_id IS NULL").
		Order("snapshot_date ASC").
		Scan(&snapshots).Error
	
	if err != nil {
		return nil, err
	}

	// Convert to HistoricalPoint format with change percentages
	var history []HistoricalPoint
	for i, snapshot := range snapshots {
		var changePercent float64
		if i > 0 && snapshots[i-1].WordCount > 0 {
			changePercent = float64(snapshot.WordCount-snapshots[i-1].WordCount) / float64(snapshots[i-1].WordCount) * 100
		}

		history = append(history, HistoricalPoint{
			Date:          snapshot.SnapshotDate.Format("2006-01-02"),
			WordCount:     snapshot.WordCount,
			ChangePercent: changePercent,
		})
	}

	return history, nil
}

// getAgencyHistory retrieves word count history for a specific agency
func getAgencyHistory(agencySlug string, startDate, endDate time.Time) ([]HistoricalPoint, error) {
	type SnapshotData struct {
		SnapshotDate time.Time
		WordCount    int
	}

	var snapshots []SnapshotData
	
	// Query historical snapshots for specific agency
	err := database.DB.Table("historical_snapshots hs").
		Select("hs.snapshot_date, hs.word_count").
		Joins("JOIN agencies a ON a.id = hs.agency_id").
		Where("a.slug = ?", agencySlug).
		Where("hs.snapshot_date >= ? AND hs.snapshot_date <= ?", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).
		Where("hs.title_id IS NULL"). // Agency-level snapshots only
		Order("hs.snapshot_date ASC").
		Scan(&snapshots).Error
	
	if err != nil {
		return nil, err
	}

	// Convert to HistoricalPoint format with change percentages
	var history []HistoricalPoint
	for i, snapshot := range snapshots {
		var changePercent float64
		if i > 0 && snapshots[i-1].WordCount > 0 {
			changePercent = float64(snapshot.WordCount-snapshots[i-1].WordCount) / float64(snapshots[i-1].WordCount) * 100
		}

		history = append(history, HistoricalPoint{
			Date:          snapshot.SnapshotDate.Format("2006-01-02"),
			WordCount:     snapshot.WordCount,
			ChangePercent: changePercent,
		})
	}

	return history, nil
}

func ExportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract export type from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/export/")
	exportType := strings.Split(path, "/")[0]

	switch exportType {
	case "agencies":
		AgenciesHandler(w, r) // Reuse existing handler logic
	case "titles":
		TitlesHandler(w, r) // Reuse existing handler logic
	case "metrics":
		WordCountMetricsHandler(w, r) // Reuse existing handler logic
	default:
		http.Error(w, "Invalid export type", http.StatusBadRequest)
	}
}

func CalculateChecksumsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed - use POST", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("[HANDLER] CalculateChecksumsHandler called")

	// Get all agencies
	var agencies []models.Agency
	if err := database.DB.Find(&agencies).Error; err != nil {
		log.Printf("[HANDLER] Failed to fetch agencies: %v", err)
		http.Error(w, "Failed to fetch agencies", http.StatusInternalServerError)
		return
	}

	log.Printf("[HANDLER] Found %d agencies to process", len(agencies))

	successCount := 0
	errorCount := 0
	skippedCount := 0

	// Process each agency
	for _, agency := range agencies {
		result, err := calculateAndStoreAgencyChecksum(agency.ID)
		if err != nil {
			log.Printf("[HANDLER] Failed to process agency %s: %v", agency.Name, err)
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
	}

	log.Printf("[HANDLER] Calculation completed: %d created/updated, %d skipped, %d errors", 
		successCount, skippedCount, errorCount)

	response := map[string]interface{}{
		"success": errorCount == 0,
		"message": fmt.Sprintf("Processed %d agencies", len(agencies)),
		"stats": map[string]int{
			"total": len(agencies),
			"created_updated": successCount,
			"skipped": skippedCount,
			"errors": errorCount,
		},
	}

	if errorCount > 0 {
		response["message"] = fmt.Sprintf("Processed %d agencies with %d errors", len(agencies), errorCount)
		w.WriteHeader(http.StatusPartialContent)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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