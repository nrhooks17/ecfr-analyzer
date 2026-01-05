package database

import (
	"fmt"
	"log"
	"os"

	"ecfr-analyzer/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() error {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "ecfr"
	}
	
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		return fmt.Errorf("DB_PASSWORD environment variable is required")
	}
	
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "ecfr"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		host, port, user, password, dbname)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Disable SQL query logging
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable UUID extension
	err = DB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error
	if err != nil {
		return fmt.Errorf("failed to create uuid extension: %w", err)
	}

	// Auto-migrate schemas
	err = DB.AutoMigrate(
		&models.Agency{},
		&models.Title{},
		&models.AgencyCFRReference{},
		&models.TitleContent{},
		&models.HistoricalSnapshot{},
		&models.AgencyChecksum{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto-migrate: %w", err)
	}

	// Create performance indexes
	err = createPerformanceIndexes()
	if err != nil {
		return fmt.Errorf("failed to create performance indexes: %w", err)
	}

	log.Println("Database connected and migrated successfully")
	return nil
}

func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func createPerformanceIndexes() error {
	indexes := []string{
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_agency_cfr_references_agency_id ON agency_cfr_references(agency_id)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_agency_cfr_references_title_id ON agency_cfr_references(title_id)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_title_contents_title_id ON title_contents(title_id)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_title_contents_title_id_content_date ON title_contents(title_id, content_date DESC)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_title_contents_word_count ON title_contents(word_count) WHERE word_count IS NOT NULL",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_agencies_parent_id ON agencies(parent_id) WHERE parent_id IS NOT NULL",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_historical_snapshots_agency_title ON historical_snapshots(agency_id, title_id, snapshot_date)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_historical_snapshots_snapshot_date ON historical_snapshots(snapshot_date)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_agency_checksums_agency_id ON agency_checksums(agency_id)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_agency_checksums_updated_at ON agency_checksums(updated_at)",
	}

	for _, indexSQL := range indexes {
		if err := DB.Exec(indexSQL).Error; err != nil {
			log.Printf("Warning: Failed to create index: %s - %v", indexSQL, err)
		}
	}

	return nil
}