package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pranshuj73/oni/logger"
)

const anilistAPIURL = "https://graphql.anilist.co"

// Client represents an AniList API client
type Client struct {
	httpClient  *http.Client
	accessToken string
	userID      int
}

// NewClient creates a new AniList client
func NewClient() (*Client, error) {
	logger.Debug("Creating new AniList client", nil)

	client := &Client{
		httpClient: &http.Client{},
	}

	// Try to load existing token
	token, err := LoadToken()
	if err != nil {
		logger.Error("Failed to load AniList token", err, nil)
		return nil, fmt.Errorf("failed to load token: %w", err)
	}

	if token == "" {
		logger.Warn("No AniList token found", nil)
		// Prompt for token
		token, err = PromptForToken()
		if err != nil {
			logger.Error("Failed to get AniList token", err, nil)
			return nil, fmt.Errorf("failed to get token: %w", err)
		}

		// Save token
		if err := SaveToken(token); err != nil {
			logger.Error("Failed to save AniList token", err, nil)
			return nil, fmt.Errorf("failed to save token: %w", err)
		}
	}

	client.accessToken = token

	// Try to load user ID
	userID, err := LoadUserID()
	if err != nil {
		logger.Error("Failed to load user ID", err, nil)
		return nil, fmt.Errorf("failed to load user ID: %w", err)
	}

	if userID == 0 {
		logger.Debug("User ID not found, fetching from API", nil)
		// Fetch user ID from API
		userID, err = client.fetchUserID(context.Background())
		if err != nil {
			logger.Error("Failed to fetch user ID from API", err, nil)
			return nil, fmt.Errorf("failed to fetch user ID: %w", err)
		}

		// Save user ID
		if err := SaveUserID(userID); err != nil {
			logger.Error("Failed to save user ID", err, nil)
			return nil, fmt.Errorf("failed to save user ID: %w", err)
		}
	}

	client.userID = userID

	logger.Info("AniList client created successfully", map[string]interface{}{
		"userID": userID,
	})

	return client, nil
}

// NewClientWithToken creates a new AniList client with the given token
func NewClientWithToken(token string) (*Client, error) {
	logger.Debug("Creating new AniList client with provided token", nil)

	client := &Client{
		httpClient:  &http.Client{},
		accessToken: token,
	}

	// Fetch user ID
	userID, err := client.fetchUserID(context.Background())
	if err != nil {
		logger.Error("Failed to fetch user ID with provided token", err, nil)
		return nil, fmt.Errorf("failed to fetch user ID: %w", err)
	}

	client.userID = userID

	logger.Info("AniList client created with token successfully", map[string]interface{}{
		"userID": userID,
	})

	return client, nil
}

// graphqlRequest represents a GraphQL request
type graphqlRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// graphqlResponse represents a GraphQL response
type graphqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// query executes a GraphQL query
func (c *Client) query(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	// Extract query name for logging (first line typically contains operation name)
	queryName := "unknown"
	if lines := strings.Split(query, "\n"); len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		if len(firstLine) > 50 {
			firstLine = firstLine[:50] + "..."
		}
		queryName = firstLine
	}

	logger.Debug("Executing AniList GraphQL query", map[string]interface{}{
		"query":     queryName,
		"variables": variables,
	})

	reqBody := graphqlRequest{
		Query:     query,
		Variables: variables,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		logger.Error("Failed to marshal GraphQL request", err, map[string]interface{}{
			"query": queryName,
		})
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anilistAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("Failed to create HTTP request", err, map[string]interface{}{
			"query": queryName,
		})
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.accessToken != "" {
		// Use token exactly as provided, just trim whitespace (like jerry.sh)
		token := strings.TrimSpace(c.accessToken)
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error("Failed to execute GraphQL request", err, map[string]interface{}{
			"query": queryName,
			"url":   anilistAPIURL,
		})
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read GraphQL response", err, map[string]interface{}{
			"query":      queryName,
			"statusCode": resp.StatusCode,
		})
		return fmt.Errorf("failed to read response: %w", err)
	}

	var gqlResp graphqlResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		logger.Error("Failed to unmarshal GraphQL response", err, map[string]interface{}{
			"query":      queryName,
			"statusCode": resp.StatusCode,
			"response":   string(body),
		})
		// If JSON unmarshal fails, return the raw response for debugging
		return fmt.Errorf("failed to unmarshal response (status %d): %s", resp.StatusCode, string(body))
	}

	if len(gqlResp.Errors) > 0 {
		// Return more detailed error info
		errMsg := gqlResp.Errors[0].Message
		if len(gqlResp.Errors) > 1 {
			errMsg += fmt.Sprintf(" (and %d more errors)", len(gqlResp.Errors)-1)
		}
		// Include HTTP status code if available
		if resp.StatusCode != 200 {
			errMsg += fmt.Sprintf(" [HTTP %d]", resp.StatusCode)
		}
		logger.Error("GraphQL query returned errors", nil, map[string]interface{}{
			"query":      queryName,
			"error":      errMsg,
			"statusCode": resp.StatusCode,
		})
		return fmt.Errorf("GraphQL error: %s", errMsg)
	}
	
	// Check if data is null/empty (might indicate auth failure)
	if len(gqlResp.Data) == 0 || string(gqlResp.Data) == "null" {
		logger.Error("Empty GraphQL response", nil, map[string]interface{}{
			"query":      queryName,
			"statusCode": resp.StatusCode,
		})
		return fmt.Errorf("empty response from API - token may be invalid [HTTP %d]", resp.StatusCode)
	}

	if err := json.Unmarshal(gqlResp.Data, result); err != nil {
		logger.Error("Failed to unmarshal GraphQL data", err, map[string]interface{}{
			"query": queryName,
		})
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	logger.Debug("GraphQL query successful", map[string]interface{}{
		"query":      queryName,
		"statusCode": resp.StatusCode,
	})

	return nil
}

// fetchUserID fetches the user ID from the API
func (c *Client) fetchUserID(ctx context.Context) (int, error) {
	var result UserResponse
	if err := c.query(ctx, GetUserIDQuery, nil, &result); err != nil {
		return 0, err
	}

	return result.Viewer.ID, nil
}

// GetUserID returns the user ID for the authenticated user
func (c *Client) GetUserID(ctx context.Context) (int, error) {
	return c.fetchUserID(ctx)
}

// SearchAnime searches for anime by name
func (c *Client) SearchAnime(ctx context.Context, search string, showAdult bool) ([]Anime, error) {
	logger.Info("Searching anime on AniList", map[string]interface{}{
		"search":    search,
		"showAdult": showAdult,
	})

	variables := map[string]interface{}{
		"search":  search,
		"page":    1,
		"perPage": 20,
	}

	if !showAdult {
		variables["isAdult"] = false
	}

	var result SearchResponse
	if err := c.query(ctx, SearchAnimeQuery, variables, &result); err != nil {
		return nil, err
	}

	logger.Info("Anime search completed", map[string]interface{}{
		"search":      search,
		"resultsCount": len(result.Page.Media),
	})

	return result.Page.Media, nil
}

// GetAnimeList gets the user's anime list by status
func (c *Client) GetAnimeList(ctx context.Context, status string) ([]MediaListEntry, error) {
	logger.Info("Fetching anime list from AniList", map[string]interface{}{
		"userID": c.userID,
		"status": status,
	})

	variables := map[string]interface{}{
		"userId": c.userID,
		"type":   "ANIME",
	}

	if status != "" {
		variables["status"] = status
	}

	var result ListResponse
	if err := c.query(ctx, GetAnimeListQuery, variables, &result); err != nil {
		return nil, err
	}

	var entries []MediaListEntry
	for _, list := range result.MediaListCollection.Lists {
		entries = append(entries, list.Entries...)
	}

	logger.Info("Anime list fetched", map[string]interface{}{
		"userID":       c.userID,
		"status":       status,
		"entriesCount": len(entries),
	})

	return entries, nil
}

// UpdateProgress updates the watch progress for an anime
func (c *Client) UpdateProgress(ctx context.Context, mediaID, progress int, status string) error {
	logger.Info("Updating anime progress on AniList", map[string]interface{}{
		"mediaID":  mediaID,
		"progress": progress,
		"status":   status,
	})

	variables := map[string]interface{}{
		"mediaId":  mediaID,
		"progress": progress,
		"status":   status,
	}

	var result UpdateResponse
	err := c.query(ctx, UpdateProgressMutation, variables, &result)
	if err != nil {
		logger.Error("Failed to update anime progress", err, map[string]interface{}{
			"mediaID":  mediaID,
			"progress": progress,
		})
		return err
	}

	logger.Info("Anime progress updated successfully", map[string]interface{}{
		"mediaID":  mediaID,
		"progress": progress,
	})

	return nil
}

// UpdateScore updates the score for an anime
func (c *Client) UpdateScore(ctx context.Context, mediaID int, score float64) error {
	logger.Info("Updating anime score on AniList", map[string]interface{}{
		"mediaID": mediaID,
		"score":   score,
	})

	variables := map[string]interface{}{
		"mediaId": mediaID,
		"score":   score,
	}

	var result UpdateResponse
	err := c.query(ctx, UpdateScoreMutation, variables, &result)
	if err != nil {
		logger.Error("Failed to update anime score", err, map[string]interface{}{
			"mediaID": mediaID,
		})
		return err
	}

	logger.Info("Anime score updated successfully", map[string]interface{}{
		"mediaID": mediaID,
		"score":   score,
	})

	return nil
}

// UpdateStatus updates the status for an anime
func (c *Client) UpdateStatus(ctx context.Context, mediaID int, status string) error {
	logger.Info("Updating anime status on AniList", map[string]interface{}{
		"mediaID": mediaID,
		"status":  status,
	})

	variables := map[string]interface{}{
		"mediaId": mediaID,
		"status":  status,
	}

	var result UpdateResponse
	err := c.query(ctx, UpdateStatusMutation, variables, &result)
	if err != nil {
		logger.Error("Failed to update anime status", err, map[string]interface{}{
			"mediaID": mediaID,
		})
		return err
	}

	logger.Info("Anime status updated successfully", map[string]interface{}{
		"mediaID": mediaID,
		"status":  status,
	})

	return nil
}

// GetAnimeInfo gets detailed information about an anime
func (c *Client) GetAnimeInfo(ctx context.Context, mediaID int) (*Anime, error) {
	logger.Debug("Fetching anime info from AniList", map[string]interface{}{
		"mediaID": mediaID,
	})

	variables := map[string]interface{}{
		"id": mediaID,
	}

	var result struct {
		Media Anime `json:"Media"`
	}

	if err := c.query(ctx, GetAnimeInfoQuery, variables, &result); err != nil {
		logger.Error("Failed to fetch anime info", err, map[string]interface{}{
			"mediaID": mediaID,
		})
		return nil, err
	}

	logger.Debug("Anime info fetched successfully", map[string]interface{}{
		"mediaID": mediaID,
		"title":   result.Media.Title.UserPreferred,
	})

	return &result.Media, nil
}

// GetCurrentUserID returns the current user's ID (synchronous, no API call)
func (c *Client) GetCurrentUserID() int {
	return c.userID
}

