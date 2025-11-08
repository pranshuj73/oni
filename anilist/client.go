package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
	client := &Client{
		httpClient: &http.Client{},
	}

	// Try to load existing token
	token, err := LoadToken()
	if err != nil {
		return nil, fmt.Errorf("failed to load token: %w", err)
	}

	if token == "" {
		// Prompt for token
		token, err = PromptForToken()
		if err != nil {
			return nil, fmt.Errorf("failed to get token: %w", err)
		}

		// Save token
		if err := SaveToken(token); err != nil {
			return nil, fmt.Errorf("failed to save token: %w", err)
		}
	}

	client.accessToken = token

	// Try to load user ID
	userID, err := LoadUserID()
	if err != nil {
		return nil, fmt.Errorf("failed to load user ID: %w", err)
	}

	if userID == 0 {
		// Fetch user ID from API
		userID, err = client.fetchUserID(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user ID: %w", err)
		}

		// Save user ID
		if err := SaveUserID(userID); err != nil {
			return nil, fmt.Errorf("failed to save user ID: %w", err)
		}
	}

	client.userID = userID

	return client, nil
}

// NewClientWithToken creates a new AniList client with the given token
func NewClientWithToken(token string) (*Client, error) {
	client := &Client{
		httpClient:  &http.Client{},
		accessToken: token,
	}

	// Fetch user ID
	userID, err := client.fetchUserID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user ID: %w", err)
	}

	client.userID = userID

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
	reqBody := graphqlRequest{
		Query:     query,
		Variables: variables,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anilistAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
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
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var gqlResp graphqlResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
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
		return fmt.Errorf("GraphQL error: %s", errMsg)
	}
	
	// Check if data is null/empty (might indicate auth failure)
	if len(gqlResp.Data) == 0 || string(gqlResp.Data) == "null" {
		return fmt.Errorf("empty response from API - token may be invalid [HTTP %d]", resp.StatusCode)
	}

	if err := json.Unmarshal(gqlResp.Data, result); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

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

	return result.Page.Media, nil
}

// GetAnimeList gets the user's anime list by status
func (c *Client) GetAnimeList(ctx context.Context, status string) ([]MediaListEntry, error) {
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

	return entries, nil
}

// UpdateProgress updates the watch progress for an anime
func (c *Client) UpdateProgress(ctx context.Context, mediaID, progress int, status string) error {
	variables := map[string]interface{}{
		"mediaId":  mediaID,
		"progress": progress,
		"status":   status,
	}

	var result UpdateResponse
	return c.query(ctx, UpdateProgressMutation, variables, &result)
}

// UpdateScore updates the score for an anime
func (c *Client) UpdateScore(ctx context.Context, mediaID int, score float64) error {
	variables := map[string]interface{}{
		"mediaId": mediaID,
		"score":   score,
	}

	var result UpdateResponse
	return c.query(ctx, UpdateScoreMutation, variables, &result)
}

// UpdateStatus updates the status for an anime
func (c *Client) UpdateStatus(ctx context.Context, mediaID int, status string) error {
	variables := map[string]interface{}{
		"mediaId": mediaID,
		"status":  status,
	}

	var result UpdateResponse
	return c.query(ctx, UpdateStatusMutation, variables, &result)
}

// GetAnimeInfo gets detailed information about an anime
func (c *Client) GetAnimeInfo(ctx context.Context, mediaID int) (*Anime, error) {
	variables := map[string]interface{}{
		"id": mediaID,
	}

	var result struct {
		Media Anime `json:"Media"`
	}

	if err := c.query(ctx, GetAnimeInfoQuery, variables, &result); err != nil {
		return nil, err
	}

	return &result.Media, nil
}

// GetCurrentUserID returns the current user's ID (synchronous, no API call)
func (c *Client) GetCurrentUserID() int {
	return c.userID
}

