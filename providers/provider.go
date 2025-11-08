package providers

import (
	"context"
	"fmt"
)

// Provider defines the interface for anime providers
type Provider interface {
	// GetEpisodeInfo fetches episode information
	GetEpisodeInfo(ctx context.Context, mediaID int, episodeNum int, title string) (*EpisodeInfo, error)

	// GetVideoLink extracts the video URL and subtitles
	GetVideoLink(ctx context.Context, episodeInfo *EpisodeInfo, quality string, subOrDub string) (*VideoData, error)

	// Name returns the provider name
	Name() string
}

// EpisodeInfo contains information about an episode
type EpisodeInfo struct {
	EpisodeID    string
	EpisodeTitle string
	MediaType    string // For hdrezka
	ShowID       string // For allanime
}

// VideoData contains video and subtitle information
type VideoData struct {
	VideoURL     string
	SubtitleURLs []string
	Referer      string
}

// GetProvider returns a provider by name
func GetProvider(name string) (Provider, error) {
	switch name {
	case "allanime":
		return NewAllAnimeProvider(), nil
	case "aniwatch":
		return NewAniWatchProvider(), nil
	case "yugen":
		return NewYugenProvider(), nil
	case "hdrezka":
		return NewHDRezkaProvider(), nil
	case "aniworld":
		return NewAniWorldProvider(), nil
	case "crunchyroll":
		return NewCrunchyrollProvider(), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}

