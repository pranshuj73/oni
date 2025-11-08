package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	allAnimeBase   = "allanime.day"
	allAnimeRefr   = "https://allanime.to"
	allAnimeAPIURL = "https://api.allanime.day/api"
)

// AllAnimeProvider implements the allanime provider
type AllAnimeProvider struct {
	client *http.Client
}

// NewAllAnimeProvider creates a new AllAnime provider
func NewAllAnimeProvider() *AllAnimeProvider {
	return &AllAnimeProvider{
		client: &http.Client{},
	}
}

// Name returns the provider name
func (p *AllAnimeProvider) Name() string {
	return "allanime"
}

// GetEpisodeInfo searches for anime and returns episode info
func (p *AllAnimeProvider) GetEpisodeInfo(ctx context.Context, mediaID int, episodeNum int, title string) (*EpisodeInfo, error) {
	// Search for the anime
	queryTitle := strings.ReplaceAll(title, " ", "+")

	searchQuery := `query($search: SearchInput, $limit: Int, $page: Int, $translationType: VaildTranslationTypeEnumType, $countryOrigin: VaildCountryOriginEnumType) {
		shows(search: $search, limit: $limit, page: $page, translationType: $translationType, countryOrigin: $countryOrigin) {
			edges {
				_id
				name
				availableEpisodes
				__typename
			}
		}
	}`

	variables := map[string]interface{}{
		"search": map[string]interface{}{
			"allowAdult":   false,
			"allowUnknown": false,
			"query":        queryTitle,
		},
		"limit":           40,
		"page":            1,
		"translationType": "sub",
		"countryOrigin":   "ALL",
	}

	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal variables: %w", err)
	}

	params := url.Values{}
	params.Add("variables", string(variablesJSON))
	params.Add("query", searchQuery)

	reqURL := fmt.Sprintf("%s?%s", allAnimeAPIURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Referer", allAnimeRefr)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var searchResp struct {
		Data struct {
			Shows struct {
				Edges []struct {
					ID                string `json:"_id"`
					Name              string `json:"name"`
					AvailableEpisodes struct {
						Sub int `json:"sub"`
						Dub int `json:"dub"`
					} `json:"availableEpisodes"`
				} `json:"edges"`
			} `json:"shows"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(searchResp.Data.Shows.Edges) == 0 {
		return nil, fmt.Errorf("no results found for: %s", title)
	}

	// Use the first result
	show := searchResp.Data.Shows.Edges[0]

	return &EpisodeInfo{
		EpisodeID:    fmt.Sprintf("%d", episodeNum),
		EpisodeTitle: fmt.Sprintf("Episode %d", episodeNum),
		ShowID:       show.ID,
	}, nil
}

// GetVideoLink extracts video links from allanime
func (p *AllAnimeProvider) GetVideoLink(ctx context.Context, episodeInfo *EpisodeInfo, quality string, subOrDub string) (*VideoData, error) {
	// Fetch episode sources
	episodeQuery := `query($showId: String!, $translationType: VaildTranslationTypeEnumType!, $episodeString: String!) {
		episode(showId: $showId, translationType: $translationType, episodeString: $episodeString) {
			episodeString
			sourceUrls
		}
	}`

	variables := map[string]interface{}{
		"showId":          episodeInfo.ShowID,
		"translationType": subOrDub,
		"episodeString":   episodeInfo.EpisodeID,
	}

	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal variables: %w", err)
	}

	params := url.Values{}
	params.Add("variables", string(variablesJSON))
	params.Add("query", episodeQuery)

	reqURL := fmt.Sprintf("%s?%s", allAnimeAPIURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Referer", allAnimeRefr)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var episodeResp struct {
		Data struct {
			Episode struct {
				SourceUrls json.RawMessage `json:"sourceUrls"`
			} `json:"episode"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &episodeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check if SourceUrls is empty or null
	if len(episodeResp.Data.Episode.SourceUrls) == 0 || string(episodeResp.Data.Episode.SourceUrls) == "null" {
		return nil, fmt.Errorf("no video links found: episode may not exist or source URLs are empty")
	}

	// Parse source URLs
	links, err := p.extractLinks(ctx, episodeResp.Data.Episode.SourceUrls)
	if err != nil {
		return nil, fmt.Errorf("failed to extract links: %w", err)
	}

	if len(links) == 0 {
		return nil, fmt.Errorf("no video links found: could not extract links from source URLs")
	}

	// Select quality
	videoURL := p.selectQuality(links, quality)

	return &VideoData{
		VideoURL: videoURL,
		Referer:  allAnimeRefr,
	}, nil
}

// extractLinks extracts video links from source URLs - matches jerry.sh exactly
func (p *AllAnimeProvider) extractLinks(ctx context.Context, sourceURLs json.RawMessage) (map[string]string, error) {
	// Convert JSON to string and process like jerry.sh line 944
	jsonStr := string(sourceURLs)
	// Replace {} with newlines, fix unicode escapes, extract sourceName:sourceUrl pairs
	jsonStr = strings.ReplaceAll(jsonStr, "{", "\n")
	jsonStr = strings.ReplaceAll(jsonStr, "}", "\n")
	jsonStr = strings.ReplaceAll(jsonStr, "\\u002F", "/")
	jsonStr = strings.ReplaceAll(jsonStr, "\\", "")
	
	// Extract sourceName : sourceUrl pairs (jerry.sh: sed -nE 's|.*sourceUrl":"--([^"]*)".*sourceName":"([^"]*)".*|\2 :\1|p')
	re := regexp.MustCompile(`sourceUrl":"--([^"]*)".*sourceName":"([^"]*)"`)
	matches := re.FindAllStringSubmatch(jsonStr, -1)
	
	if len(matches) == 0 {
		return nil, fmt.Errorf("no source URLs found in response")
	}
	
	// Build resp string with sourceName :sourceUrl format (matches jerry.sh exactly)
	var resp strings.Builder
	for _, match := range matches {
		if len(match) >= 3 {
			sourceName := match[2]
			sourceURL := match[1]
			// Format: "sourceName :sourceUrl" (space before colon, no space after)
			resp.WriteString(fmt.Sprintf("%s :%s\n", sourceName, sourceURL))
		}
	}
	respStr := resp.String()

	// Try all 5 providers in parallel (like jerry.sh does)
	type providerResult struct {
		links map[string]string
		err   error
	}
	
	results := make(chan providerResult, 5)
	
	// Try each provider (1-5) like jerry.sh
	for providerNum := 1; providerNum <= 5; providerNum++ {
		go func(num int) {
			links, err := p.generateLinksForProvider(ctx, respStr, num)
			results <- providerResult{links: links, err: err}
		}(providerNum)
	}
	
	// Collect all results
	allLinks := make(map[string]string)
	for i := 0; i < 5; i++ {
		result := <-results
		if result.err == nil && len(result.links) > 0 {
			for quality, link := range result.links {
				allLinks[quality] = link
			}
		}
	}
	
	if len(allLinks) == 0 {
		return nil, fmt.Errorf("no video links found: all providers failed")
	}
	
	return allLinks, nil
}

// generateLinksForProvider generates links for a specific provider (matches jerry.sh generate_links)
func (p *AllAnimeProvider) generateLinksForProvider(ctx context.Context, respStr string, providerNum int) (map[string]string, error) {
	var providerID string
	var providerName string
	
	// Match jerry.sh provider_init logic - uses sed patterns to find matching line
	// jerry.sh: sed -n "$2" | head -1 | cut -d':' -f2
	switch providerNum {
	case 1: // gogoanime - pattern: /Luf-mp4 :/p
		providerName = "gogoanime"
		re := regexp.MustCompile(`Luf-mp4\s*:([^\n]+)`)
		match := re.FindStringSubmatch(respStr)
		if len(match) < 2 {
			return nil, nil // Provider not found, return nil (not an error)
		}
		// Extract the part after the colon (cut -d':' -f2)
		providerID = strings.TrimSpace(match[1])
		providerID = p.decodeProviderID(providerID)
	case 2: // wixmp - pattern: /Default :/p
		providerName = "wixmp"
		re := regexp.MustCompile(`Default\s*:([^\n]+)`)
		match := re.FindStringSubmatch(respStr)
		if len(match) < 2 {
			return nil, nil
		}
		providerID = strings.TrimSpace(match[1])
		providerID = p.decodeProviderID(providerID)
	case 3: // dropbox - pattern: /Sak :/p
		providerName = "dropbox"
		re := regexp.MustCompile(`Sak\s*:([^\n]+)`)
		match := re.FindStringSubmatch(respStr)
		if len(match) < 2 {
			return nil, nil
		}
		providerID = strings.TrimSpace(match[1])
		providerID = p.decodeProviderID(providerID)
	case 4: // wetransfer - pattern: /Kir :/p
		providerName = "wetransfer"
		re := regexp.MustCompile(`Kir\s*:([^\n]+)`)
		match := re.FindStringSubmatch(respStr)
		if len(match) < 2 {
			return nil, nil
		}
		providerID = strings.TrimSpace(match[1])
		providerID = p.decodeProviderID(providerID)
	case 5: // sharepoint - pattern: /S-mp4 :/p
		providerName = "sharepoint"
		re := regexp.MustCompile(`S-mp4\s*:([^\n]+)`)
		match := re.FindStringSubmatch(respStr)
		if len(match) < 2 {
			return nil, nil
		}
		providerID = strings.TrimSpace(match[1])
		providerID = p.decodeProviderID(providerID)
	}
	
	if providerID == "" {
		return nil, fmt.Errorf("provider %d ID not found", providerNum)
	}
	
	// Get links from provider (matches jerry.sh get_links)
	return p.getLinksFromProviderID(ctx, providerID, providerName)
}

// decodeProviderID decodes the hex-encoded provider ID (matches jerry.sh provider_init)
func (p *AllAnimeProvider) decodeProviderID(encoded string) string {
	// This matches jerry.sh's complex hex decoding
	// jerry.sh: sed 's/../&\n/g' | sed 's/^01$/9/g;s/^08$/0/g;...' | tr -d '\n'
	decoded := ""
	for i := 0; i < len(encoded); i += 2 {
		if i+2 > len(encoded) {
			break
		}
		hex := encoded[i : i+2]
		var char string
		switch hex {
		case "01": char = "9"
		case "08": char = "0"
		case "05": char = "="
		case "0a": char = "2"
		case "0b": char = "3"
		case "0c": char = "4"
		case "07": char = "?"
		case "00": char = "8"
		case "5c": char = "d"
		case "0f": char = "7"
		case "5e": char = "f"
		case "17": char = "/"
		case "54": char = "l"
		case "09": char = "1"
		case "48": char = "p"
		case "4f": char = "w"
		case "0e": char = "6"
		case "5b": char = "c"
		case "5d": char = "e"
		case "0d": char = "5"
		case "53": char = "k"
		case "1e": char = "&"
		case "5a": char = "b"
		case "59": char = "a"
		case "4a": char = "r"
		case "4c": char = "t"
		case "4e": char = "v"
		case "57": char = "o"
		case "51": char = "i"
		default: char = hex
		}
		decoded += char
	}
	// Replace /clock with /clock.json
	decoded = strings.ReplaceAll(decoded, "/clock", "/clock.json")
	return decoded
}

// getLinksFromProviderID fetches links from a provider ID (matches jerry.sh get_links exactly)
func (p *AllAnimeProvider) getLinksFromProviderID(ctx context.Context, providerID, providerName string) (map[string]string, error) {
	fullURL := fmt.Sprintf("https://%s%s", allAnimeBase, providerID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Referer", allAnimeRefr)
	req.Header.Set("User-Agent", "uwu")
	
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	bodyStr := string(body)
	// Replace },{ with newlines like jerry.sh does
	bodyStr = strings.ReplaceAll(bodyStr, "},{", "\n")
	links := make(map[string]string)

	// Match jerry.sh get_links logic exactly (line 164-184)
	if strings.Contains(fullURL, "repackager.wixmp.com") {
		// Handle wixmp repackager (jerry.sh lines 167-171)
		re := regexp.MustCompile(`link":"([^"]*)".*resolutionStr":"([^"]*)"`)
		matches := re.FindAllStringSubmatch(bodyStr, -1)
		if len(matches) > 0 {
			extractLink := strings.ReplaceAll(matches[0][1], `\/`, `/`)
			extractLink = strings.ReplaceAll(extractLink, "repackager.wixmp.com/", "")
			extractLink = strings.Split(extractLink, ".urlset")[0]
			
			// Extract quality numbers from the link
			qualityRe := regexp.MustCompile(`/,([^/]*),/mp4`)
			qualityMatch := qualityRe.FindStringSubmatch(matches[0][1])
			if len(qualityMatch) >= 2 {
				qualities := strings.Split(qualityMatch[1], ",")
				for _, q := range qualities {
					link := strings.ReplaceAll(extractLink, ",", q)
					links[q] = link
				}
			}
		}
	} else if strings.Contains(fullURL, "vipanicdn") || strings.Contains(fullURL, "anifastcdn") {
		// Handle vipanicdn/anifastcdn (jerry.sh lines 173-180)
		if strings.Contains(bodyStr, "original.m3u") {
			re := regexp.MustCompile(`link":"([^"]*)".*resolutionStr":"([^"]*)"`)
			matches := re.FindAllStringSubmatch(bodyStr, -1)
			for _, match := range matches {
				if len(match) >= 3 {
					link := strings.ReplaceAll(match[1], `\/`, `/`)
					quality := match[2]
					links[quality] = link
				}
			}
		} else {
			// Parse m3u8 playlist (jerry.sh lines 177-179)
			re := regexp.MustCompile(`link":"([^"]*)"`)
			match := re.FindStringSubmatch(bodyStr)
			if len(match) >= 2 {
				baseURL := strings.ReplaceAll(match[1], `\/`, `/`)
				relativeLink := baseURL[:strings.LastIndex(baseURL, "/")+1]
				
				// Fetch and parse m3u8 (jerry.sh line 179)
				m3u8Req, _ := http.NewRequestWithContext(ctx, "GET", baseURL, nil)
				m3u8Req.Header.Set("Referer", allAnimeRefr)
				m3u8Req.Header.Set("User-Agent", "uwu")
				m3u8Resp, err := p.client.Do(m3u8Req)
				if err == nil {
					defer m3u8Resp.Body.Close()
					m3u8Body, _ := io.ReadAll(m3u8Resp.Body)
					m3u8Str := string(m3u8Body)
					// Process m3u8: sed 's|^#.*x||g; s|,.*|p|g; /^#/d; $!N; s|\n| >|'
					// This extracts quality from lines with 'x' (like "1080x720") and pairs with next line
					lines := strings.Split(m3u8Str, "\n")
					for i := 0; i < len(lines)-1; i++ {
						line := lines[i]
						// Remove comment lines starting with #
						if strings.HasPrefix(line, "#") {
							// Extract resolution if present (like #EXT-X-STREAM-INF:RESOLUTION=1920x1080)
							if strings.Contains(line, "RESOLUTION=") {
								resRe := regexp.MustCompile(`RESOLUTION=(\d+)x(\d+)`)
								if resMatch := resRe.FindStringSubmatch(line); len(resMatch) >= 3 {
									quality := resMatch[2] // Height
									nextLine := strings.TrimSpace(lines[i+1])
									if nextLine != "" && !strings.HasPrefix(nextLine, "#") {
										links[quality] = relativeLink + nextLine
									}
								}
							}
			continue
		}
					}
				}
			}
		}
	} else {
		// Default case (jerry.sh line 182)
		// Pattern: resolutionStr >link or hls url
		re := regexp.MustCompile(`link":"([^"]*)".*resolutionStr":"([^"]*)"`)
		matches := re.FindAllStringSubmatch(bodyStr, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				link := strings.ReplaceAll(match[1], `\/`, `/`)
				quality := match[2]
				links[quality] = link
			}
		}
		// Also check for hls (jerry.sh: s|.*hls","url":"([^"]*)".*|\1|p)
		reHLS := regexp.MustCompile(`hls","url":"([^"]*)"`)
		matchesHLS := reHLS.FindAllStringSubmatch(bodyStr, -1)
		for _, match := range matchesHLS {
			if len(match) >= 2 {
				link := strings.ReplaceAll(match[1], `\/`, `/`)
				links["1080"] = link
			break
			}
		}
	}

	return links, nil
}

// getLinksFromProvider fetches links from a provider URL
func (p *AllAnimeProvider) getLinksFromProvider(ctx context.Context, providerURL string) (map[string]string, error) {
	fullURL := fmt.Sprintf("https://%s%s", allAnimeBase, providerURL)

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Referer", allAnimeRefr)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	links := make(map[string]string)

	// Parse response for video links
	re := regexp.MustCompile(`"link":"([^"]*)".*?"resolutionStr":"([^"]*)"`)
	matches := re.FindAllStringSubmatch(string(body), -1)

	for _, match := range matches {
		if len(match) >= 3 {
			link := strings.ReplaceAll(match[1], `\/`, `/`)
			quality := match[2]
			links[quality] = link
		}
	}

	// Also check for m3u8 links
	reM3u8 := regexp.MustCompile(`"hls","url":"([^"]*)"`)
	matchesM3u8 := reM3u8.FindAllStringSubmatch(string(body), -1)

	for _, match := range matchesM3u8 {
		if len(match) >= 2 {
			link := strings.ReplaceAll(match[1], `\/`, `/`)
			links["1080"] = link
			break
		}
	}

	return links, nil
}

// selectQuality selects the best quality link
func (p *AllAnimeProvider) selectQuality(links map[string]string, preferredQuality string) string {
	// Try preferred quality
	if link, ok := links[preferredQuality]; ok {
		return link
	}

	// Sort qualities in descending order
	qualities := make([]string, 0, len(links))
	for q := range links {
		qualities = append(qualities, q)
	}

	sort.Slice(qualities, func(i, j int) bool {
		qi, _ := strconv.Atoi(qualities[i])
		qj, _ := strconv.Atoi(qualities[j])
		return qi > qj
	})

	// Return best quality
	if len(qualities) > 0 {
		return links[qualities[0]]
	}

	return ""
}

