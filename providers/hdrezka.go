package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// HDRezkaProvider implements the hdrezka provider
type HDRezkaProvider struct {
	client *http.Client
}

// NewHDRezkaProvider creates a new HDRezka provider
func NewHDRezkaProvider() *HDRezkaProvider {
	return &HDRezkaProvider{
		client: &http.Client{},
	}
}

// Name returns the provider name
func (p *HDRezkaProvider) Name() string {
	return "hdrezka"
}

// GetEpisodeInfo fetches episode information from hdrezka
func (p *HDRezkaProvider) GetEpisodeInfo(ctx context.Context, mediaID int, episodeNum int, title string) (*EpisodeInfo, error) {
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

	// Search on hdrezka
	searchURL := fmt.Sprintf("https://hdrezka.website/search/?do=search&subaction=search&q=%s", searchTitle)

	req, err = http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

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
	reResult := regexp.MustCompile(`src="([^"]*)".*?<a href="https://hdrezka\.website/(.*)/(.*)/(.*)\.html">([^<]*)</a>.*?<div>([0-9]*)`)
	matchesResult := reResult.FindStringSubmatch(string(body))

	if len(matchesResult) < 7 {
		return nil, fmt.Errorf("no results found on hdrezka")
	}

	mediaType := matchesResult[2]
	episodeID := fmt.Sprintf("%s/%s", matchesResult[3], matchesResult[4])

	return &EpisodeInfo{
		EpisodeID:    episodeID,
		EpisodeTitle: fmt.Sprintf("Episode %d", episodeNum),
		MediaType:    mediaType,
	}, nil
}

// GetVideoLink extracts video links from hdrezka
func (p *HDRezkaProvider) GetVideoLink(ctx context.Context, episodeInfo *EpisodeInfo, quality string, subOrDub string) (*VideoData, error) {
	// This is a simplified implementation
	// The original jerry.sh has complex decryption logic
	return nil, fmt.Errorf("hdrezka provider requires complex decryption - not yet fully implemented")
}

