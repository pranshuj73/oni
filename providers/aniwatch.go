package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// AniWatchProvider implements the aniwatch provider
type AniWatchProvider struct {
	client *http.Client
}

// NewAniWatchProvider creates a new AniWatch provider
func NewAniWatchProvider() *AniWatchProvider {
	// Configure HTTP client with timeout and connection pooling
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &AniWatchProvider{
		client: &http.Client{
			Timeout:   60 * time.Second,
			Transport: transport,
		},
	}
}

// Name returns the provider name
func (p *AniWatchProvider) Name() string {
	return "aniwatch"
}

// hiAnimeLines extracts the HTML from a hianime AJAX response (which wraps HTML in a JSON
// envelope) and returns it split into one-tag-per-line, matching jerry.sh's approach of
// `sed "s/</\n/g; s/\\\//g"`.
func hiAnimeLines(body []byte) []string {
	// Try to unwrap the JSON envelope {"html":"..."}
	var envelope struct {
		HTML string `json:"html"`
	}
	html := string(body)
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.HTML != "" {
		html = envelope.HTML
	}
	// Unescape JSON-encoded forward slashes and split on "<"
	html = strings.ReplaceAll(html, `\/`, `/`)
	return strings.Split(html, "<")
}

// GetEpisodeInfo fetches episode information from aniwatch
func (p *AniWatchProvider) GetEpisodeInfo(ctx context.Context, mediaID int, episodeNum int, title string) (*EpisodeInfo, error) {
	// Fetch aniwatch ID from mal-backup
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

	// Parse the backup JSON and extract the hianime show ID from known site keys.
	// The URL format is "https://<domain>/<slug>-<id>" — we want the trailing numeric ID.
	var backup struct {
		Sites map[string]map[string]struct {
			URL string `json:"url"`
		} `json:"Sites"`
	}
	if err := json.Unmarshal(body, &backup); err != nil {
		return nil, fmt.Errorf("failed to parse backup JSON: %w", err)
	}

	var aniwatchID string
	reTrailingID := regexp.MustCompile(`-(\d+)$`)
	for _, key := range []string{"Zoro", "Aniwatch", "Zoro-1"} {
		entries, ok := backup.Sites[key]
		if !ok {
			continue
		}
		for _, entry := range entries {
			if m := reTrailingID.FindStringSubmatch(strings.TrimRight(entry.URL, "/")); len(m) >= 2 {
				aniwatchID = m[1]
				break
			}
		}
		if aniwatchID != "" {
			break
		}
	}

	if aniwatchID == "" {
		return nil, fmt.Errorf("aniwatch ID not found for media ID %d", mediaID)
	}

	// Fetch episode list
	req, err = http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("https://hianime.to/ajax/v2/episode/list/%s", aniwatchID), nil)
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

	// Parse episode list: split on "<" (jerry.sh approach) then match per line.
	// Each episode anchor looks like: a title="Ep Title" ... data-number="N" ... data-id="12345"
	reEpLine := regexp.MustCompile(`a\s[^>]*title="([^"]*)"[^>]*data-id="(\d+)"`)
	reDataNum := regexp.MustCompile(`data-number="(\d+)"`)

	var episodeID, episodeTitle string
	for _, line := range hiAnimeLines(body) {
		m := reEpLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		// Prefer data-number attribute for matching the episode
		numMatch := reDataNum.FindStringSubmatch(line)
		if numMatch != nil {
			if numMatch[1] == fmt.Sprintf("%d", episodeNum) {
				episodeTitle = m[1]
				episodeID = m[2]
				break
			}
		}
	}

	// Fallback: take the Nth anchor (1-indexed) if data-number wasn't found
	if episodeID == "" {
		count := 0
		for _, line := range hiAnimeLines(body) {
			if m := reEpLine.FindStringSubmatch(line); m != nil {
				count++
				if count == episodeNum {
					episodeTitle = m[1]
					episodeID = m[2]
					break
				}
			}
		}
	}

	if episodeID == "" {
		return nil, fmt.Errorf("episode %d not found", episodeNum)
	}

	return &EpisodeInfo{
		EpisodeID:    episodeID,
		EpisodeTitle: episodeTitle,
	}, nil
}

// GetVideoLink extracts video links from aniwatch
func (p *AniWatchProvider) GetVideoLink(ctx context.Context, episodeInfo *EpisodeInfo, quality string, subOrDub string) (*VideoData, error) {
	// Get server list
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("https://hianime.to/ajax/v2/episode/servers?episodeId=%s", episodeInfo.EpisodeID), nil)
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

	// Extract server ID — split on "<" then match per line (jerry.sh approach)
	reServerLine := regexp.MustCompile(`data-type="([^"]*)"[^>]*data-id="(\d+)"`)
	var sourceID string
	for _, preferred := range []string{subOrDub, "raw"} {
		for _, line := range hiAnimeLines(body) {
			if m := reServerLine.FindStringSubmatch(line); m != nil && m[1] == preferred {
				sourceID = m[2]
				break
			}
		}
		if sourceID != "" {
			break
		}
	}
	if sourceID == "" {
		return nil, fmt.Errorf("no server found")
	}

	// Get embed link
	req, err = http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("https://hianime.to/ajax/v2/episode/sources?id=%s", sourceID), nil)
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

	var embedResp struct {
		Link string `json:"link"`
	}
	if err := json.Unmarshal(body, &embedResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embed response: %w", err)
	}

	// Parse embed link
	reEmbed := regexp.MustCompile(`(.*)/embed-([246])/e-([0-9])/(.*)\?k=1`)
	matchesEmbed := reEmbed.FindStringSubmatch(embedResp.Link)
	if len(matchesEmbed) < 5 {
		return nil, fmt.Errorf("invalid embed link format")
	}

	providerLink := matchesEmbed[1]
	embedType := matchesEmbed[2]
	eNumber := matchesEmbed[3]
	embedSourceID := matchesEmbed[4]

	// Get actual source
	req, err = http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/embed-%s/ajax/e-%s/getSources?id=%s", providerLink, embedType, eNumber, embedSourceID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err = p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse video link — JSON response, "file" field ending in .m3u8
	reVideo := regexp.MustCompile(`"file"\s*:\s*"([^"]*\.m3u8)"`)
	matchesVideo := reVideo.FindStringSubmatch(string(body))
	if len(matchesVideo) < 2 {
		return nil, fmt.Errorf("video link not found")
	}

	videoURL := strings.ReplaceAll(matchesVideo[1], `\/`, `/`)
	if quality != "" {
		videoURL = strings.Replace(videoURL, "/playlist.m3u8", fmt.Sprintf("/%s/index.m3u8", quality), 1)
	}

	// Extract subtitles
	reSubs := regexp.MustCompile(`"file"\s*:\s*"([^"]*\.vtt)"`)
	var subtitles []string
	for _, m := range reSubs.FindAllStringSubmatch(string(body), -1) {
		if len(m) >= 2 {
			subtitles = append(subtitles, strings.ReplaceAll(m[1], `\/`, `/`))
		}
	}

	return &VideoData{
		VideoURL:     videoURL,
		SubtitleURLs: subtitles,
	}, nil
}
