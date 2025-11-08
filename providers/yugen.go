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

// YugenProvider implements the yugen provider
type YugenProvider struct {
	client *http.Client
}

// NewYugenProvider creates a new Yugen provider
func NewYugenProvider() *YugenProvider {
	return &YugenProvider{
		client: &http.Client{},
	}
}

// Name returns the provider name
func (p *YugenProvider) Name() string {
	return "yugen"
}

// GetEpisodeInfo fetches episode information from yugen
func (p *YugenProvider) GetEpisodeInfo(ctx context.Context, mediaID int, episodeNum int, title string) (*EpisodeInfo, error) {
	// Fetch yugen URL from mal-backup
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

	// Extract yugen URL
	re := regexp.MustCompile(`"YugenAnime".*?"url": *"([^"]*)"`)
	matches := re.FindStringSubmatch(string(body))

	if len(matches) < 2 {
		return nil, fmt.Errorf("yugen URL not found for media ID %d", mediaID)
	}

	yugenURL := strings.Replace(matches[1], "tv/anime", "tv/watch", 1)
	watchURL := fmt.Sprintf("%s%d/", yugenURL, episodeNum)

	// Fetch episode page
	req, err = http.NewRequestWithContext(ctx, "GET", watchURL, nil)
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

	// Extract episode title
	reTitle := regexp.MustCompile(fmt.Sprintf(`%d\s:\s([^<]*)`, episodeNum))
	matchesTitle := reTitle.FindStringSubmatch(string(body))

	epTitle := fmt.Sprintf("Episode %d", episodeNum)
	if len(matchesTitle) >= 2 {
		epTitle = matchesTitle[1]
	}

	// Extract yugen episode ID
	reID := regexp.MustCompile(`id="main-embed" src=".*/e/([^/]*)/?"`)
	matchesID := reID.FindStringSubmatch(string(body))

	if len(matchesID) < 2 {
		return nil, fmt.Errorf("yugen episode ID not found")
	}

	return &EpisodeInfo{
		EpisodeID:    matchesID[1],
		EpisodeTitle: epTitle,
	}, nil
}

// GetVideoLink extracts video links from yugen
func (p *YugenProvider) GetVideoLink(ctx context.Context, episodeInfo *EpisodeInfo, quality string, subOrDub string) (*VideoData, error) {
	// Fetch video data
	embedURL := "https://yugenanime.tv/api/embed/"

	data := fmt.Sprintf("id=%s&ac=0", episodeInfo.EpisodeID)

	req, err := http.NewRequestWithContext(ctx, "POST", embedURL, strings.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var videoResp struct {
		HLS []string `json:"hls"`
	}

	if err := json.Unmarshal(body, &videoResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(videoResp.HLS) == 0 {
		return nil, fmt.Errorf("no HLS links found")
	}

	videoURL := videoResp.HLS[0]

	// Apply quality if specified
	if quality != "" {
		videoURL = strings.Replace(videoURL, ".m3u8", fmt.Sprintf(".%s.m3u8", quality), 1)
	}

	return &VideoData{
		VideoURL: videoURL,
	}, nil
}

