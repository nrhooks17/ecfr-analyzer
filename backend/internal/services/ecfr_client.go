package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	BaseURL = "https://www.ecfr.gov"
	Timeout = 30 * time.Second
)

type ECFRClient struct {
	client *http.Client
}

type AgencyData struct {
	Name      string `json:"name"`
	ShortName string `json:"short_name"`
	Slug      string `json:"slug"`
	ParentID  string `json:"parent_id"`
	Children  []AgencyData `json:"children"`
	CFRReferences []struct {
		Title   int    `json:"title"`
		Chapter string `json:"chapter"`
	} `json:"cfr_references"`
}

type AgencyResponse struct {
	Agencies []AgencyData `json:"agencies"`
}

type TitleResponse struct {
	Titles []struct {
		Number           int    `json:"number"`
		Name             string `json:"name"`
		LatestAmendedOn  string `json:"latest_amended_on"`
		LatestIssueDate  string `json:"latest_issue_date"`
		UpToDateAsOf     string `json:"up_to_date_as_of"`
		Reserved         bool   `json:"reserved"`
	} `json:"titles"`
}

type TitleStructure struct {
	Identifier string `json:"identifier"`
	Label      string `json:"label"`
	Size       int    `json:"size"`
	Type       string `json:"type"`
	Reserved   bool   `json:"reserved"`
}

func NewECFRClient() *ECFRClient {
	return &ECFRClient{
		client: &http.Client{
			Timeout: Timeout,
		},
	}
}

func (c *ECFRClient) FetchAgencies() (*AgencyResponse, error) {
	url := fmt.Sprintf("%s/api/admin/v1/agencies.json", BaseURL)
	log.Printf("[ECFR_CLIENT] Fetching agencies from: %s", url)
	
	resp, err := c.client.Get(url)
	if err != nil {
		log.Printf("[ECFR_CLIENT] Failed to fetch agencies: %v", err)
		return nil, fmt.Errorf("failed to fetch agencies: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[ECFR_CLIENT] Unexpected status code for agencies: %d", resp.StatusCode)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ECFR_CLIENT] Failed to read agencies response body: %v", err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var agencies AgencyResponse
	if err := json.Unmarshal(body, &agencies); err != nil {
		log.Printf("[ECFR_CLIENT] Failed to unmarshal agencies: %v", err)
		return nil, fmt.Errorf("failed to unmarshal agencies: %w", err)
	}

	log.Printf("[ECFR_CLIENT] Successfully fetched %d agencies", len(agencies.Agencies))
	return &agencies, nil
}

func (c *ECFRClient) FetchTitles() (*TitleResponse, error) {
	url := fmt.Sprintf("%s/api/versioner/v1/titles.json", BaseURL)
	log.Printf("[ECFR_CLIENT] Fetching titles from: %s", url)
	
	resp, err := c.client.Get(url)
	if err != nil {
		log.Printf("[ECFR_CLIENT] Failed to fetch titles: %v", err)
		return nil, fmt.Errorf("failed to fetch titles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[ECFR_CLIENT] Unexpected status code for titles: %d", resp.StatusCode)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ECFR_CLIENT] Failed to read titles response body: %v", err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var titles TitleResponse
	if err := json.Unmarshal(body, &titles); err != nil {
		log.Printf("[ECFR_CLIENT] Failed to unmarshal titles: %v", err)
		return nil, fmt.Errorf("failed to unmarshal titles: %w", err)
	}

	log.Printf("[ECFR_CLIENT] Successfully fetched %d titles", len(titles.Titles))
	return &titles, nil
}

func (c *ECFRClient) FetchTitleContent(titleNumber int, date string) (string, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	
	url := fmt.Sprintf("%s/api/versioner/v1/full/%s/title-%d.xml", BaseURL, date, titleNumber)
	
	resp, err := c.client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch title %d content: %w", titleNumber, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code for title %d: %d", titleNumber, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read title %d content: %w", titleNumber, err)
	}

	return string(body), nil
}

func (c *ECFRClient) FetchTitleStructure(titleNumber int, date string) (*TitleStructure, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	
	url := fmt.Sprintf("%s/api/versioner/v1/structure/%s/title-%d.json", BaseURL, date, titleNumber)
	
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch title %d structure: %w", titleNumber, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code for title %d structure: %d", titleNumber, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read title %d structure: %w", titleNumber, err)
	}

	var structure TitleStructure
	if err := json.Unmarshal(body, &structure); err != nil {
		return nil, fmt.Errorf("failed to unmarshal title %d structure: %w", titleNumber, err)
	}

	return &structure, nil
}