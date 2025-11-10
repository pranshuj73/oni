package providers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
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

	// Check cache first
	cached, err := LoadProviderMapping("hdrezka", mediaID)
	if err == nil && cached != nil {
		// Parse cached provider ID (format: "mediaType|episodeID")
		parts := strings.Split(cached.ProviderID, "|")
		if len(parts) == 2 {
			return &EpisodeInfo{
				EpisodeID:    parts[1],
				EpisodeTitle: fmt.Sprintf("Episode %d", episodeNum),
				MediaType:    parts[0],
			}, nil
		}
	}

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

	// Save to cache (store as "mediaType|episodeID" for easy parsing)
	cacheValue := fmt.Sprintf("%s|%s", mediaType, episodeID)
	SaveProviderMapping("hdrezka", mediaID, cacheValue, title)

	return &EpisodeInfo{
		EpisodeID:    episodeID,
		EpisodeTitle: fmt.Sprintf("Episode %d", episodeNum),
		MediaType:    mediaType,
	}, nil
}

// GetVideoLink extracts video links from hdrezka
func (p *HDRezkaProvider) GetVideoLink(ctx context.Context, episodeInfo *EpisodeInfo, quality string, subOrDub string) (*VideoData, error) {
	// Extract data_id from episode_id (format: "series_id/episode_id" or just episode_id)
	dataID := episodeInfo.EpisodeID
	if strings.Contains(dataID, "/") {
		parts := strings.Split(dataID, "/")
		if len(parts) >= 2 {
			dataID = parts[0]
		}
	}

	// Get episode page to extract data_id and translator_id
	episodeURL := fmt.Sprintf("https://hdrezka.website/%s/%s.html", episodeInfo.MediaType, strings.ReplaceAll(episodeInfo.EpisodeID, "=", "/"))
	
	req, err := http.NewRequestWithContext(ctx, "GET", episodeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	bodyStr := string(body)
	
	// Extract data_id from the page
	var dataIDStr string
	if episodeInfo.MediaType == "films" {
		reDataID := regexp.MustCompile(fmt.Sprintf(`initCDNMoviesEvents\(%s, ([0-9]*),`, regexp.QuoteMeta(dataID)))
		matches := reDataID.FindStringSubmatch(bodyStr)
		if len(matches) >= 2 {
			dataIDStr = matches[1]
		}
	} else {
		reDataID := regexp.MustCompile(fmt.Sprintf(`initCDNSeriesEvents\(%s, ([0-9]*),`, regexp.QuoteMeta(dataID)))
		matches := reDataID.FindStringSubmatch(bodyStr)
		if len(matches) >= 2 {
			dataIDStr = matches[1]
		}
	}
	
	if dataIDStr == "" {
		// Try to extract from episode_id directly
		reDataID := regexp.MustCompile(`([0-9]+)`)
		matches := reDataID.FindStringSubmatch(dataID)
		if len(matches) >= 2 {
			dataIDStr = matches[1]
		}
	}
	
	if dataIDStr == "" {
		return nil, fmt.Errorf("could not extract data_id")
	}
	
	// Extract default translator_id
	var defaultTranslatorID string
	if episodeInfo.MediaType == "films" {
		reTranslator := regexp.MustCompile(fmt.Sprintf(`initCDNMoviesEvents\(%s, %s, ([0-9]*)`, regexp.QuoteMeta(dataID), regexp.QuoteMeta(dataIDStr)))
		matches := reTranslator.FindStringSubmatch(bodyStr)
		if len(matches) >= 2 {
			defaultTranslatorID = matches[1]
		}
	} else {
		reTranslator := regexp.MustCompile(fmt.Sprintf(`initCDNSeriesEvents\(%s, %s, ([0-9]*)`, regexp.QuoteMeta(dataID), regexp.QuoteMeta(dataIDStr)))
		matches := reTranslator.FindStringSubmatch(bodyStr)
		if len(matches) >= 2 {
			defaultTranslatorID = matches[1]
		}
	}
	
	translatorID := defaultTranslatorID
	
	// Extract available translations (optional - for now use default)
	// This would allow user to select translation, but for now we use default
	
	// Extract season_id if it's a series
	var seasonID string
	if episodeInfo.MediaType != "films" {
		reSeason := regexp.MustCompile(`data-tab_id="([0-9]+)">`)
		matches := reSeason.FindStringSubmatch(bodyStr)
		if len(matches) >= 2 {
			seasonID = matches[1]
		} else {
			seasonID = "1" // Default to season 1
		}
	}
	
	// Extract episode number from episode_id
	episodeNum := 1
	reEpNum := regexp.MustCompile(`-([0-9]+)`)
	matches := reEpNum.FindStringSubmatch(episodeInfo.EpisodeID)
	if len(matches) >= 2 {
		if num, err := strconv.Atoi(matches[1]); err == nil {
			episodeNum = num
		}
	}
	
	// Make POST request to get CDN data
	postURL := "https://hdrezka.website/ajax/get_cdn_series/"
	postData := url.Values{}
	postData.Set("id", dataIDStr)
	postData.Set("translator_id", translatorID)
	if episodeInfo.MediaType != "films" {
		postData.Set("season", seasonID)
		postData.Set("episode", strconv.Itoa(episodeNum))
	}
	postData.Set("action", "get_stream")
	
	req, err = http.NewRequestWithContext(ctx, "POST", postURL, strings.NewReader(postData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
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
	
	var jsonResp struct {
		Success bool   `json:"success"`
		URL     string `json:"url"`
		Subtitle string `json:"subtitle"`
	}
	
	if err := json.Unmarshal(body, &jsonResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	if !jsonResp.Success || jsonResp.URL == "" {
		return nil, fmt.Errorf("failed to get video URL from hdrezka")
	}
	
	// Decrypt the video URL
	encryptedURL := jsonResp.URL
	
	// Remove the decryption table strings
	table := []string{
		"ISE=", "IUA=", "IV4=", "ISM=", "ISQ=", "QCE=", "QEA=", "QF4=", "QCM=", "QCQ=",
		"XiE=", "XkA=", "Xl4=", "XiM=", "XiQ=", "IyE=", "I0A=", "I14=", "IyM=", "IyQ=",
		"JCE=", "JEA=", "JF4=", "JCM=", "JCQ=", "ISEh", "ISFA", "ISFe", "ISEj", "ISEk",
		"IUAh", "IUBA", "IUBe", "IUAj", "IUAk", "IV4h", "IV5A", "IV5e", "IV4j", "IV4k",
		"ISMh", "ISNA", "ISNe", "ISMj", "ISMk", "ISQh", "ISRA", "ISRe", "ISQj", "ISQk",
		"QCEh", "QCFA", "QCFe", "QCEj", "QCEk", "QEAh", "QEBA", "QEBe", "QEAj", "QEAk",
		"QF4h", "QF5A", "QF5e", "QF4j", "QF4k", "QCMh", "QCNA", "QCNe", "QCMj", "QCMk",
		"QCQh", "QCRA", "QCRe", "QCQj", "QCQk", "XiEh", "XiFA", "XiFe", "XiEj", "XiEk",
		"XkAh", "XkBA", "XkBe", "XkAj", "XkAk", "Xl4h", "Xl5A", "Xl5e", "Xl4j", "Xl4k",
		"XiMh", "XiNA", "XiNe", "XiMj", "XiMk", "XiQh", "XiRA", "XiRe", "XiQj", "XiQk",
		"IyEh", "IyFA", "IyFe", "IyEj", "IyEk", "I0Ah", "I0BA", "I0Be", "I0Aj", "I0Ak",
		"I14h", "I15A", "I15e", "I14j", "I14k", "IyMh", "IyNA", "IyNe", "IyMj", "IyMk",
		"IyQh", "IyRA", "IyRe", "IyQj", "IyQk", "JCEh", "JCFA", "JCFe", "JCEj", "JCEk",
		"JEAh", "JEBA", "JEBe", "JEAj", "JEAk", "JF4h", "JF5A", "JF5e", "JF4j", "JF4k",
		"JCMh", "JCNA", "JCNe", "JCMj", "JCMk", "JCQh", "JCRA", "JCRe", "JCQj", "JCQk",
	}
	
	for _, t := range table {
		encryptedURL = strings.ReplaceAll(encryptedURL, t, "")
	}
	
	// Remove underscores and decode base64
	encryptedURL = strings.ReplaceAll(encryptedURL, "_", "")
	
	decoded, err := base64.StdEncoding.DecodeString(encryptedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted URL: %w", err)
	}
	
	// Parse the decoded data to extract video links
	decodedStr := string(decoded)
	
	// The decoded string contains quality-specific links in format: [quality]url
	// Parse it to extract links
	reVideoLinks := regexp.MustCompile(`\[([^\]]+)\]([^,\[]+)`)
	videoMatches := reVideoLinks.FindAllStringSubmatch(decodedStr, -1)
	
	if len(videoMatches) == 0 {
		return nil, fmt.Errorf("no video links found in decoded data")
	}
	
	// Find the best matching quality
	videoURL := ""
	if quality == "" || quality == "best" {
		// Use the last (usually best) quality
		videoURL = videoMatches[len(videoMatches)-1][2]
	} else if quality == "worst" {
		// Use the first (usually worst) quality
		videoURL = videoMatches[0][2]
	} else {
		// Find matching quality
		for _, match := range videoMatches {
			if strings.Contains(match[1], quality) {
				videoURL = match[2]
				break
			}
		}
		// If no match found, use best quality
		if videoURL == "" {
			videoURL = videoMatches[len(videoMatches)-1][2]
		}
	}
	
	// Clean up video URL (remove " or " patterns)
	videoURL = strings.Split(videoURL, " or ")[0]
	videoURL = strings.TrimSpace(videoURL)
	
	// Extract subtitles
	var subtitles []string
	if jsonResp.Subtitle != "" {
		// Parse subtitle JSON array
		subtitleStr := jsonResp.Subtitle
		// Remove brackets and split by comma
		subtitleStr = strings.Trim(subtitleStr, "[]")
		subtitleParts := strings.Split(subtitleStr, ",")
		for _, part := range subtitleParts {
			part = strings.Trim(part, `" `)
			if part != "" {
				subtitles = append(subtitles, part)
			}
		}
	}
	
	return &VideoData{
		VideoURL:     videoURL,
		SubtitleURLs: subtitles,
		Referer:      "https://hdrezka.website/",
	}, nil
}

