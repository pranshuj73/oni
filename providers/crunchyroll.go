package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// CrunchyrollProvider implements the crunchyroll provider
type CrunchyrollProvider struct {
	client      *http.Client
	accessToken string
}

// NewCrunchyrollProvider creates a new Crunchyroll provider
func NewCrunchyrollProvider() *CrunchyrollProvider {
	return &CrunchyrollProvider{
		client: &http.Client{},
	}
}

// Name returns the provider name
func (p *CrunchyrollProvider) Name() string {
	return "crunchyroll"
}

// getAccessToken fetches an access token from Crunchyroll
func (p *CrunchyrollProvider) getAccessToken(ctx context.Context) error {
	if p.accessToken != "" {
		return nil
	}

	tokenURL := "https://www.crunchyroll.com/auth/v1/token"

	data := "grant_type=client_id&scope=offline_access"

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Basic Y3Jfd2ViOg==")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	p.accessToken = tokenResp.AccessToken

	return nil
}

// GetEpisodeInfo fetches episode information from crunchyroll
func (p *CrunchyrollProvider) GetEpisodeInfo(ctx context.Context, mediaID int, episodeNum int, title string) (*EpisodeInfo, error) {
	// Fetch crunchyroll URL from mal-backup
	backupURL := fmt.Sprintf("https://raw.githubusercontent.com/bal-mackup/mal-backup/master/anilist/anime/%d.json", mediaID)

	req, err := http.NewRequestWithContext(ctx, "GET", backupURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Extract crunchyroll URL
	re := regexp.MustCompile(`"Crunchyroll".*?"url":"([^"]*)"`)
	matches := re.FindStringSubmatch(string(body))

	if len(matches) < 2 {
		return nil, fmt.Errorf("crunchyroll URL not found for media ID %d", mediaID)
	}

	crURL := matches[1]

	// Extract series ID from URL
	reID := regexp.MustCompile(`.*/([^/]*)/.*`)
	matchesID := reID.FindStringSubmatch(crURL)

	if len(matchesID) < 2 {
		return nil, fmt.Errorf("failed to extract series ID")
	}

	seriesID := matchesID[1]

	// Get access token
	if err := p.getAccessToken(ctx); err != nil {
		return nil, err
	}

	// Fetch season info
	seasonsURL := fmt.Sprintf("https://www.crunchyroll.com/content/v2/cms/series/%s/seasons", seriesID)

	req, err = http.NewRequestWithContext(ctx, "GET", seasonsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.accessToken)

	resp, err = p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Extract first season ID
	reSeasonID := regexp.MustCompile(`"id":"([^"]*)"`)
	matchesSeasonID := reSeasonID.FindStringSubmatch(string(body))

	if len(matchesSeasonID) < 2 {
		return nil, fmt.Errorf("season ID not found")
	}

	seasonID := matchesSeasonID[1]

	// Fetch episodes
	episodesURL := fmt.Sprintf("https://www.crunchyroll.com/content/v2/cms/seasons/%s/episodes", seasonID)

	req, err = http.NewRequestWithContext(ctx, "GET", episodesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.accessToken)

	resp, err = p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse episodes (simplified)
	// Split by },{ to handle multiple episodes
	episodes := strings.Split(string(body), "},{")
	if len(episodes) < episodeNum {
		return nil, fmt.Errorf("episode %d not found", episodeNum)
	}

	// Extract episode ID from the desired episode
	epData := episodes[episodeNum-1]
	reEpID := regexp.MustCompile(`"id":"([^"]*)"`)
	matchesEpID := reEpID.FindStringSubmatch(epData)

	if len(matchesEpID) < 2 {
		return nil, fmt.Errorf("episode ID not found")
	}

	episodeID := matchesEpID[1]

	// Extract title
	reEpTitle := regexp.MustCompile(`"title":"([^"]*)"`)
	matchesEpTitle := reEpTitle.FindStringSubmatch(epData)

	epTitle := fmt.Sprintf("Episode %d", episodeNum)
	if len(matchesEpTitle) >= 2 {
		epTitle = matchesEpTitle[1]
	}

	return &EpisodeInfo{
		EpisodeID:    episodeID,
		EpisodeTitle: epTitle,
	}, nil
}

// GetVideoLink extracts video links from crunchyroll
func (p *CrunchyrollProvider) GetVideoLink(ctx context.Context, episodeInfo *EpisodeInfo, quality string, subOrDub string) (*VideoData, error) {
	// Crunchyroll requires DRM and complex authentication
	// This is a simplified stub - full implementation would require DRM handling
	return nil, fmt.Errorf("crunchyroll provider requires DRM handling - not yet fully implemented")
}

