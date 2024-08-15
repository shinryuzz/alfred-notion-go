package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

const (
	notionApiUrl     = "https://api.notion.com/v1/pages"
	notionApiVersion = "2021-05-13"
)

var (
	notionDbId  string
	notionToken string
	logger      *log.Logger
)

type NotionClient struct {
	httpClient *http.Client
	token      string
}

func init() {
	logFile := filepath.Join(".", "error.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	logger = log.New(file, "", log.Ldate|log.Ltime|log.Lshortfile)

	err = godotenv.Load()
	if err != nil {
		logger.Printf("Error loading .env file: %v", err)
	}

	notionDbId = os.Getenv("NOTION_DB_ID")
	notionToken = os.Getenv("NOTION_TOKEN")

	if notionDbId == "" || notionToken == "" {
		logger.Fatal("Error: NOTION_DB_ID and NOTION_TOKEN environments must be set")
	}
}

func NewNotionClient(token string) *NotionClient {
	return &NotionClient{
		httpClient: &http.Client{},
		token:      token,
	}
}

func (c *NotionClient) RegisterRecord(databaseId, title string) error {
	payload := map[string]interface{}{
		"parent": map[string]string{
			"database_id": databaseId,
		},
		"properties": map[string]interface{}{
			"Name": map[string]interface{}{
				"title": []map[string]interface{}{
					{
						"text": map[string]string{
							"content": title,
						},
					},
				},
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshalling payload: %w", err)
	}

	req, err := http.NewRequest("POST", notionApiUrl, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", c.token))
	req.Header.Set("Notion-Version", notionApiVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status: %d): %s", resp.StatusCode, body)
	}

	return nil
}

func main() {
	if len(os.Args) < 2 {
		logger.Println("Error: Insufficient arguments")
		fmt.Println("Usage: <TRIGGER_NAME> <TITLE>")
		os.Exit(1)
	}

	title := os.Args[1]

	client := NewNotionClient(notionToken)
	err := client.RegisterRecord(notionDbId, title)
	if err != nil {
		logger.Printf("Error registering record: %v", err)
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully registered: %s\n", title)
}
