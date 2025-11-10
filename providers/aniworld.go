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

// AniWorldProvider implements the aniworld provider
type AniWorldProvider struct {
	client *http.Client
}

// NewAniWorldProvider creates a new AniWorld provider
func NewAniWorldProvider() *AniWorldProvider {
	return &AniWorldProvider{
		client: &http.Client{},
	}
}

// Name returns the provider name
func (p *AniWorldProvider) Name() string {
	return "aniworld"
}

// GetEpisodeInfo fetches episode information from aniworld
func (p *AniWorldProvider) GetEpisodeInfo(ctx context.Context, mediaID int, episodeNum int, title string) (*EpisodeInfo, error) {
	// Fetch title from mal-backup
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

	// Extract title
	reTitle := regexp.MustCompile(`"title":\s*"([^"]*)"`)
	matchesTitle := reTitle.FindStringSubmatch(string(body))

	if len(matchesTitle) < 2 {
		return nil, fmt.Errorf("title not found in backup")
	}

	searchTitle := strings.ReplaceAll(matchesTitle[1], " ", "+")

	// Check cache first
	cached, err := LoadProviderMapping("aniworld", mediaID)
	if err == nil && cached != nil {
		// Use cached provider ID (anime link)
		return &EpisodeInfo{
			EpisodeID:    cached.ProviderID,
			EpisodeTitle: fmt.Sprintf("Episode %d", episodeNum),
		}, nil
	}

	// Search on aniworld
	searchURL := "https://aniworld.to/ajax/search"

	data := fmt.Sprintf("keyword=%s", searchTitle)

	req, err = http.NewRequestWithContext(ctx, "POST", searchURL, strings.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse search results
	var results []struct {
		Title string `json:"title"`
		Link  string `json:"link"`
	}

	// The response is multiple JSON objects separated by {}
	parts := strings.Split(string(body), "{")
	for _, part := range parts {
		if !strings.Contains(part, "title") {
			continue
		}

		var result struct {
			Title string `json:"title"`
			Link  string `json:"link"`
		}

		jsonStr := "{" + part
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			results = append(results, result)
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results found on aniworld")
	}

	// Use first result
	animeLink := results[0].Link

	// Save to cache
	SaveProviderMapping("aniworld", mediaID, animeLink, title)

	return &EpisodeInfo{
		EpisodeID:    animeLink,
		EpisodeTitle: fmt.Sprintf("Episode %d", episodeNum),
	}, nil
}

// GetVideoLink extracts video links from aniworld
func (p *AniWorldProvider) GetVideoLink(ctx context.Context, episodeInfo *EpisodeInfo, quality string, subOrDub string) (*VideoData, error) {
	// Fetch anime page
	animeURL := fmt.Sprintf("https://aniworld.to%s", episodeInfo.EpisodeID)

	req, err := http.NewRequestWithContext(ctx, "GET", animeURL, nil)
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

	// Extract episode href
	reEp := regexp.MustCompile(`href="([^"]*-\d+)".*?data-episode-id="[0-9]*"`)
	matchesEp := reEp.FindStringSubmatch(string(body))

	if len(matchesEp) < 2 {
		return nil, fmt.Errorf("episode not found")
	}

	episodeHref := matchesEp[1]

	// Fetch episode page
	episodeURL := fmt.Sprintf("https://aniworld.to%s", episodeHref)

	req, err = http.NewRequestWithContext(ctx, "GET", episodeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err = p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Extract redirect URL
	reRedirect := regexp.MustCompile(`href="(/redirect[^"]*)"`)
	matchesRedirect := reRedirect.FindStringSubmatch(string(body))

	if len(matchesRedirect) < 2 {
		return nil, fmt.Errorf("redirect URL not found")
	}

	redirectURL := fmt.Sprintf("https://aniworld.to%s", matchesRedirect[1])

	// Follow redirect to get video link
	req, err = http.NewRequestWithContext(ctx, "GET", redirectURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Don't follow redirects automatically
	p.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err = p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Extract m3u8 link
	reM3u8 := regexp.MustCompile(`"(.*\.m3u8[^"]*)"`)
	matchesM3u8 := reM3u8.FindStringSubmatch(string(body))

	if len(matchesM3u8) < 2 {
		return nil, fmt.Errorf("video link not found")
	}

	return &VideoData{
		VideoURL: matchesM3u8[1],
	}, nil
}

