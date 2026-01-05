package services

import (
	"log"
)

type ContentDownloadStrategy interface {
	DownloadTitleContent(titleNumber int) (string, error)
	GetStrategyName() string
}

type APIContentStrategy struct {
	client *ECFRClient
}

type BulkContentStrategy struct {
	bulkService *BulkDownloadService
}

func NewAPIContentStrategy(client *ECFRClient) *APIContentStrategy {
	return &APIContentStrategy{client: client}
}

func NewBulkContentStrategy() *BulkContentStrategy {
	return &BulkContentStrategy{
		bulkService: NewBulkDownloadService(),
	}
}

func (a *APIContentStrategy) DownloadTitleContent(titleNumber int) (string, error) {
	return a.client.FetchTitleContent(titleNumber, "")
}

func (a *APIContentStrategy) GetStrategyName() string {
	return "API"
}

func (b *BulkContentStrategy) DownloadTitleContent(titleNumber int) (string, error) {
	return b.bulkService.DownloadTitleXML(titleNumber)
}

func (b *BulkContentStrategy) GetStrategyName() string {
	return "Bulk Repository"
}

// ContentDownloader manages different download strategies
type ContentDownloader struct {
	strategies []ContentDownloadStrategy
}

func NewContentDownloader() *ContentDownloader {
	// Create strategies in preferred order (bulk first, then API fallback)
	strategies := []ContentDownloadStrategy{
		NewBulkContentStrategy(),
		NewAPIContentStrategy(NewECFRClient()),
	}
	
	return &ContentDownloader{
		strategies: strategies,
	}
}

func (cd *ContentDownloader) DownloadTitleContent(titleNumber int) (string, error) {
	var lastErr error
	
	for _, strategy := range cd.strategies {
		log.Printf("Attempting to download title %d using %s strategy", titleNumber, strategy.GetStrategyName())
		
		content, err := strategy.DownloadTitleContent(titleNumber)
		if err == nil {
			log.Printf("Successfully downloaded title %d using %s strategy", titleNumber, strategy.GetStrategyName())
			return content, nil
		}
		
		log.Printf("Failed to download title %d using %s strategy: %s", titleNumber, strategy.GetStrategyName(), err.Error())
		lastErr = err
	}
	
	return "", lastErr
}