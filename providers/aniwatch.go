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

// AniWatchProvider implements the aniwatch provider
type AniWatchProvider struct {
	client *http.Client
}

// NewAniWatchProvider creates a new AniWatch provider
func NewAniWatchProvider() *AniWatchProvider {
	return &AniWatchProvider{
		client: &http.Client{},
	}
}

// Name returns the provider name
func (p *AniWatchProvider) Name() string {
	return "aniwatch"
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

	// Extract aniwatch ID
	re := regexp.MustCompile(`"Zoro".*?"url":".*-([0-9]+)"`)
	matches := re.FindStringSubmatch(string(body))

	if len(matches) < 2 {
		return nil, fmt.Errorf("aniwatch ID not found for media ID %d", mediaID)
	}

	aniwatchID := matches[1]

	// Fetch episode list
	episodeListURL := fmt.Sprintf("https://hianime.to/ajax/v2/episode/list/%s", aniwatchID)

	req, err = http.NewRequestWithContext(ctx, "GET", episodeListURL, nil)
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

	// Parse episode data
	reEp := regexp.MustCompile(fmt.Sprintf(`a title="([^"]*)".*?data-id="([0-9]*)".*?Episode %d`, episodeNum))
	matchesEp := reEp.FindStringSubmatch(string(body))

	if len(matchesEp) < 3 {
		return nil, fmt.Errorf("episode %d not found", episodeNum)
	}

	return &EpisodeInfo{
		EpisodeID:    matchesEp[2],
		EpisodeTitle: matchesEp[1],
	}, nil
}

// GetVideoLink extracts video links from aniwatch
func (p *AniWatchProvider) GetVideoLink(ctx context.Context, episodeInfo *EpisodeInfo, quality string, subOrDub string) (*VideoData, error) {
	// Get server ID
	serverURL := fmt.Sprintf("https://hianime.to/ajax/v2/episode/servers?episodeId=%s", episodeInfo.EpisodeID)

	req, err := http.NewRequestWithContext(ctx, "GET", serverURL, nil)
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

	// Extract server ID for requested type
	reServer := regexp.MustCompile(fmt.Sprintf(`data-type="%s" data-id="([0-9]*)"`, subOrDub))
	matchesServer := reServer.FindStringSubmatch(string(body))

	if len(matchesServer) < 2 {
		// Fallback to raw
		reServer = regexp.MustCompile(`data-type="raw" data-id="([0-9]*)"`)
		matchesServer = reServer.FindStringSubmatch(string(body))
		if len(matchesServer) < 2 {
			return nil, fmt.Errorf("no server found")
		}
	}

	sourceID := matchesServer[1]

	// Get embed link
	embedURL := fmt.Sprintf("https://hianime.to/ajax/v2/episode/sources?id=%s", sourceID)

	req, err = http.NewRequestWithContext(ctx, "GET", embedURL, nil)
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
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
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
	sourceURL := fmt.Sprintf("%s/embed-%s/ajax/e-%s/getSources?id=%s", providerLink, embedType, eNumber, embedSourceID)

	req, err = http.NewRequestWithContext(ctx, "GET", sourceURL, nil)
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

	// Parse video link
	reVideo := regexp.MustCompile(`"file":"(.*\.m3u8)"`)
	matchesVideo := reVideo.FindStringSubmatch(string(body))

	if len(matchesVideo) < 2 {
		return nil, fmt.Errorf("video link not found")
	}

	videoURL := matchesVideo[1]

	// Apply quality if specified
	if quality != "" {
		videoURL = strings.Replace(videoURL, "/playlist.m3u8", fmt.Sprintf("/%s/index.m3u8", quality), 1)
	}

	// Extract subtitles
	reSubs := regexp.MustCompile(`"file":"([^"]*\.vtt)"`)
	matchesSubs := reSubs.FindAllStringSubmatch(string(body), -1)

	var subtitles []string
	for _, match := range matchesSubs {
		if len(match) >= 2 {
			subtitles = append(subtitles, match[1])
		}
	}

	return &VideoData{
		VideoURL:     videoURL,
		SubtitleURLs: subtitles,
	}, nil
}

