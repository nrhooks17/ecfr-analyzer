package services

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	BulkRepositoryBaseURL = "https://www.govinfo.gov/bulkdata/ECFR"
	BulkDownloadTimeout   = 60 * time.Second
)

type BulkDownloadService struct {
	client *http.Client
}

func NewBulkDownloadService() *BulkDownloadService {
	return &BulkDownloadService{
		client: &http.Client{
			Timeout: BulkDownloadTimeout,
		},
	}
}

func (b *BulkDownloadService) DownloadTitleXML(titleNumber int) (string, error) {
	url := fmt.Sprintf("%s/title-%d/ECFR-title%d.xml", BulkRepositoryBaseURL, titleNumber, titleNumber)
	log.Printf("[BULK_DOWNLOAD] Downloading title %d XML from: %s", titleNumber, url)
	
	resp, err := b.client.Get(url)
	if err != nil {
		log.Printf("[BULK_DOWNLOAD] Failed to download title %d XML: %v", titleNumber, err)
		return "", fmt.Errorf("failed to download title %d XML: %w", titleNumber, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[BULK_DOWNLOAD] Unexpected status code for title %d: %d", titleNumber, resp.StatusCode)
		return "", fmt.Errorf("unexpected status code for title %d: %d", titleNumber, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[BULK_DOWNLOAD] Failed to read title %d XML content: %v", titleNumber, err)
		return "", fmt.Errorf("failed to read title %d XML content: %w", titleNumber, err)
	}

	log.Printf("[BULK_DOWNLOAD] Successfully downloaded title %d XML (%d bytes)", titleNumber, len(body))
	return string(body), nil
}

func (b *BulkDownloadService) IsAvailable() bool {
	// Test with title 1 which should always exist
	_, err := b.DownloadTitleXML(1)
	return err == nil
}