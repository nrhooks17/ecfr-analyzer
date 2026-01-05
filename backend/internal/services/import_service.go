package services

import (
	"crypto/sha256"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"ecfr-analyzer/internal/database"
	"ecfr-analyzer/internal/models"
)

type ImportStatus struct {
	IsLoading       bool      `json:"isLoading"`
	CurrentStep     string    `json:"currentStep"`
	Progress        int       `json:"progress"`
	LastUpdated     time.Time `json:"lastUpdated"`
	Error           string    `json:"error"`
	TotalTitles     int       `json:"totalTitles"`
	CurrentTitle    int       `json:"currentTitle"`
	OverallStep     int       `json:"overallStep"`     // 1-4: which major step we're on
	TotalSteps      int       `json:"totalSteps"`      // Always 4
	AgenciesDone    bool      `json:"agenciesDone"`
	TitlesDone      bool      `json:"titlesDone"`
	ReferencesDone  bool      `json:"referencesDone"`
	ContentDone     bool      `json:"contentDone"`
	HistoricalDone  bool      `json:"historicalDone"`
}

type ImportService struct {
	client            *ECFRClient
	contentDownloader *ContentDownloader
	status            *ImportStatus
	mutex             sync.RWMutex
}

func NewImportService() *ImportService {
	return &ImportService{
		client:            NewECFRClient(),
		contentDownloader: NewContentDownloader(),
		status: &ImportStatus{
			IsLoading:      false,
			CurrentStep:    "Ready",
			Progress:       0,
			LastUpdated:    time.Now(),
			TotalSteps:     4,
			OverallStep:    0,
			AgenciesDone:   false,
			TitlesDone:     false,
			ReferencesDone: false,
			ContentDone:    false,
			HistoricalDone: false,
		},
	}
}

func (s *ImportService) GetStatus() ImportStatus {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return *s.status
}

func (s *ImportService) updateStatus(step string, progress int, err string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.status.CurrentStep = step
	s.status.Progress = progress
	s.status.LastUpdated = time.Now()
	s.status.Error = err
}

func (s *ImportService) setLoading(loading bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.status.IsLoading = loading
}

func (s *ImportService) ImportAgencies() error {
	log.Println("Starting agency import...")
	s.setOverallStep(1, "Importing agencies")
	s.updateStatus("Importing agencies", 0, "")

	agencies, err := s.client.FetchAgencies()
	if err != nil {
		s.updateStatus("Failed to import agencies", 0, err.Error())
		return err
	}

	// Create a map to track agencies by slug for parent relationships
	agencyMap := make(map[string]*models.Agency)

	// First pass: create all agencies without parent relationships
	for i, agencyData := range agencies.Agencies {
		agency := &models.Agency{
			Name:      agencyData.Name,
			ShortName: &agencyData.ShortName,
			Slug:      agencyData.Slug,
		}

		// Upsert agency
		err := database.DB.Where("slug = ?", agency.Slug).FirstOrCreate(agency).Error
		if err != nil {
			log.Printf("Error creating agency %s: %s", agency.Slug, err.Error())
			continue
		}

		agencyMap[agency.Slug] = agency

		progress := int(float64(i+1) / float64(len(agencies.Agencies)) * 100)
		s.updateStatus(fmt.Sprintf("Importing agencies (%d/%d)", i+1, len(agencies.Agencies)), progress, "")
	}

	// Second pass: process hierarchical structure (children arrays)
	for _, agencyData := range agencies.Agencies {
		if parent, exists := agencyMap[agencyData.Slug]; exists {
			// Process children of this agency
			for _, childData := range agencyData.Children {
				// Create child agency if it doesn't exist
				childAgency := &models.Agency{
					Name:      childData.Name,
					ShortName: &childData.ShortName,
					Slug:      childData.Slug,
					ParentID:  &parent.ID,
				}

				// Upsert child agency
				err := database.DB.Where("slug = ?", childAgency.Slug).FirstOrCreate(childAgency).Error
				if err != nil {
					log.Printf("Error creating child agency %s: %s", childAgency.Slug, err.Error())
					continue
				}

				// Update parent_id if it changed
				if childAgency.ParentID == nil || *childAgency.ParentID != parent.ID {
					childAgency.ParentID = &parent.ID
					database.DB.Save(childAgency)
				}

				agencyMap[childAgency.Slug] = childAgency
			}
		}
	}

	// Third pass: create CFR references for all agencies (parent and children)
	log.Println("Creating CFR references for agencies...")
	totalRefs := 0
	
	var processAgencyCFRRefs func(agencyData AgencyData)
	processAgencyCFRRefs = func(agencyData AgencyData) {
		if agency, exists := agencyMap[agencyData.Slug]; exists {
			for _, ref := range agencyData.CFRReferences {
				var title models.Title
				if err := database.DB.Where("number = ?", ref.Title).First(&title).Error; err == nil {
					cfrRef := &models.AgencyCFRReference{
						AgencyID: agency.ID,
						TitleID:  title.ID,
						Chapter:  &ref.Chapter,
					}
					database.DB.Where("agency_id = ? AND title_id = ? AND chapter = ?", 
						cfrRef.AgencyID, cfrRef.TitleID, ref.Chapter).FirstOrCreate(cfrRef)
					totalRefs++
				}
			}
			
			// Recursively process children
			for _, childData := range agencyData.Children {
				processAgencyCFRRefs(childData)
			}
		}
	}
	
	// Process all agencies (parent and children)
	for _, agencyData := range agencies.Agencies {
		processAgencyCFRRefs(agencyData)
	}

	log.Printf("Successfully imported %d agencies with %d CFR references", len(agencyMap), totalRefs)
	s.markStepComplete("agencies")
	s.markStepComplete("references") // CFR references are now done with agencies
	return nil
}

func (s *ImportService) ImportTitles() error {
	log.Println("Starting title import...")
	s.setOverallStep(2, "Importing titles")
	s.updateStatus("Importing titles", 0, "")

	titles, err := s.client.FetchTitles()
	if err != nil {
		s.updateStatus("Failed to import titles", 0, err.Error())
		return err
	}

	for i, titleData := range titles.Titles {
		title := &models.Title{
			Number:   titleData.Number,
			Name:     titleData.Name,
			Reserved: titleData.Reserved,
		}

		// Parse dates
		if titleData.LatestAmendedOn != "" {
			if date, err := time.Parse("2006-01-02", titleData.LatestAmendedOn); err == nil {
				title.LatestAmendedOn = &date
			}
		}
		if titleData.LatestIssueDate != "" {
			if date, err := time.Parse("2006-01-02", titleData.LatestIssueDate); err == nil {
				title.LatestIssueDate = &date
			}
		}
		if titleData.UpToDateAsOf != "" {
			if date, err := time.Parse("2006-01-02", titleData.UpToDateAsOf); err == nil {
				title.UpToDateAsOf = &date
			}
		}

		// Upsert title
		err := database.DB.Where("number = ?", title.Number).FirstOrCreate(title).Error
		if err != nil {
			log.Printf("Error creating title %d: %s", title.Number, err.Error())
			continue
		}

		progress := int(float64(i+1) / float64(len(titles.Titles)) * 100)
		s.updateStatus(fmt.Sprintf("Importing titles (%d/%d)", i+1, len(titles.Titles)), progress, "")
	}

	log.Printf("Successfully imported %d titles", len(titles.Titles))
	s.markStepComplete("titles")
	
	// Import content immediately after titles in the same thread
	log.Println("Starting content import...")
	s.setOverallStep(3, "Importing content")
	s.updateStatus("Preparing content download", 0, "")

	// Get non-reserved titles using raw SQL to avoid GORM boolean issues
	var activeTitles []models.Title
	if err := database.DB.Raw("SELECT * FROM titles WHERE reserved = false").Scan(&activeTitles).Error; err != nil {
		s.updateStatus("Failed to fetch titles", 0, err.Error())
		return err
	}

	s.mutex.Lock()
	s.status.TotalTitles = len(activeTitles)
	s.status.CurrentTitle = 0
	s.mutex.Unlock()

	// Use worker pool pattern with 5 workers
	titleChan := make(chan models.Title, len(activeTitles))
	var wg sync.WaitGroup
	
	log.Printf("Starting content import with %d workers for %d titles", 5, len(activeTitles))
	
	// Start 5 concurrent workers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer func() {
				log.Printf("Worker %d finished", workerID)
				wg.Done()
			}()
			log.Printf("Worker %d started", workerID)
			for title := range titleChan {
				log.Printf("Worker %d processing title %d: %s", workerID, title.Number, title.Name)
				s.downloadAndProcessTitle(title)
				s.incrementProgress()
				log.Printf("Worker %d completed title %d", workerID, title.Number)
			}
		}(i)
	}

	// Queue all titles
	log.Printf("Queuing %d titles for processing", len(activeTitles))
	for _, title := range activeTitles {
		titleChan <- title
	}
	close(titleChan)
	log.Printf("All titles queued, waiting for workers to complete")

	// Wait for all workers to complete
	wg.Wait()

	log.Printf("All workers completed. Successfully processed %d title contents", len(activeTitles))
	s.updateStatus("Content import completed", 100, "")
	s.markStepComplete("content")
	
	// Import historical data after content is complete
	log.Println("Starting historical snapshots import...")
	s.setOverallStep(4, "Creating historical snapshots")
	s.updateStatus("Creating historical snapshots", 0, "")
	
	// Use the historical service to capture current snapshot and import historical data
	historicalService := NewHistoricalService()
	
	// First capture current snapshot
	if err := historicalService.CaptureSnapshot(); err != nil {
		s.updateStatus("Failed to create current snapshot", 0, err.Error())
		log.Printf("Warning: Failed to create current snapshot: %v", err)
	}
	
	s.updateStatus("Importing historical data from eCFR API", 50, "")
	
	// Then import historical data from eCFR API
	if err := historicalService.ImportHistoricalData(); err != nil {
		log.Printf("Warning: Failed to import historical data: %v", err)
		// Don't fail the entire import if historical data fails
	}
	
	s.updateStatus("Historical snapshots completed", 100, "")
	s.markStepComplete("historical")
	
	return nil
}


func (s *ImportService) downloadAndProcessTitle(title models.Title) {
	log.Printf("Starting download for title %d: %s", title.Number, title.Name)
	
	// Download XML content using the modular content downloader (tries bulk first, then API)
	content, err := s.contentDownloader.DownloadTitleContent(title.Number)
	if err != nil {
		log.Printf("FAILED to download title %d (%s): %s", title.Number, title.Name, err.Error())
		return
	}
	
	log.Printf("Successfully downloaded title %d (%s), size: %d bytes", title.Number, title.Name, len(content))

	// Calculate word count
	wordCount := s.calculateWordCount(content)
	log.Printf("Title %d word count: %d", title.Number, wordCount)
	
	// Calculate checksum
	checksum := s.calculateChecksum(content)
	log.Printf("Title %d checksum: %s", title.Number, checksum[:8]+"...")
	
	// Store in database
	titleContent := &models.TitleContent{
		TitleID:     title.ID,
		ContentDate: time.Now().UTC().Truncate(24 * time.Hour), // Store as date only
		XMLContent:  content,
		WordCount:   &wordCount,
		Checksum:    &checksum,
	}

	log.Printf("Storing title %d content to database...", title.Number)
	// Upsert content (update if exists for same title and date)
	err = database.DB.Where("title_id = ? AND content_date = ?", 
		titleContent.TitleID, titleContent.ContentDate).
		FirstOrCreate(titleContent).Error
	if err != nil {
		log.Printf("FAILED to store content for title %d (%s): %s", title.Number, title.Name, err.Error())
		return
	}
	
	log.Printf("Successfully stored title %d (%s) content to database", title.Number, title.Name)
}

func (s *ImportService) incrementProgress() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.status.CurrentTitle++
	if s.status.TotalTitles > 0 {
		s.status.Progress = int(float64(s.status.CurrentTitle) / float64(s.status.TotalTitles) * 100)
	}
	s.status.CurrentStep = fmt.Sprintf("Downloading Title %d of %d", s.status.CurrentTitle, s.status.TotalTitles)
	s.status.LastUpdated = time.Now()
	
	log.Printf("Progress update: %d/%d titles completed (%d%%)", 
		s.status.CurrentTitle, s.status.TotalTitles, s.status.Progress)
}

func (s *ImportService) setOverallStep(step int, description string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.status.OverallStep = step
	s.status.CurrentStep = description
	s.status.IsLoading = true
	s.status.LastUpdated = time.Now()
}

func (s *ImportService) markStepComplete(stepName string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	switch stepName {
	case "agencies":
		s.status.AgenciesDone = true
	case "titles":
		s.status.TitlesDone = true
	case "references":
		s.status.ReferencesDone = true
	case "content":
		s.status.ContentDone = true
	case "historical":
		s.status.HistoricalDone = true
	}
	
	// Update overall progress based on completed steps
	completedSteps := 0
	if s.status.AgenciesDone { completedSteps++ }
	if s.status.TitlesDone { completedSteps++ }
	if s.status.ReferencesDone { completedSteps++ }
	if s.status.ContentDone { completedSteps++ }
	if s.status.HistoricalDone { completedSteps++ }
	
	s.status.Progress = int(float64(completedSteps) / float64(s.status.TotalSteps) * 100)
	s.status.LastUpdated = time.Now()
	
	// If all steps complete, mark as not loading
	if completedSteps == s.status.TotalSteps {
		s.status.IsLoading = false
		s.status.CurrentStep = "All imports completed"
	}
}

func (s *ImportService) calculateWordCount(xmlContent string) int {
	// Strip XML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(xmlContent, " ")
	
	// Normalize whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(text), " ")
	
	// Count words
	if text == "" {
		return 0
	}
	return len(strings.Fields(text))
}

func (s *ImportService) calculateChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}




func (s *ImportService) LoadAllData() error {
	log.Println("[SERVICE] Starting LoadAllData process")
	s.setLoading(true)
	defer s.setLoading(false)

	// Import in sequence: agencies (with CFR refs) -> titles (with content + historical data)
	log.Println("[SERVICE] Starting agency import")
	if err := s.ImportAgencies(); err != nil {
		log.Printf("[SERVICE] Agency import failed: %v", err)
		return err
	}
	log.Println("[SERVICE] Agency import completed successfully")

	log.Println("[SERVICE] Starting title import")
	if err := s.ImportTitles(); err != nil {
		log.Printf("[SERVICE] Title import failed: %v", err)
		return err
	}
	log.Println("[SERVICE] Title import completed successfully")

	s.updateStatus("All data loaded successfully", 100, "")
	log.Println("[SERVICE] LoadAllData process completed successfully")
	return nil
}