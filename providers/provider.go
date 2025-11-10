package providers

import (
	"context"
	"fmt"

	"github.com/pranshuj73/oni/logger"
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
	logger.Debug("Getting provider", map[string]interface{}{
		"provider": name,
	})

	switch name {
	case "allanime":
		logger.Info("Using AllAnime provider", nil)
		return NewAllAnimeProvider(), nil
	case "aniwatch":
		logger.Info("Using AniWatch provider", nil)
		return NewAniWatchProvider(), nil
	case "yugen":
		logger.Info("Using Yugen provider", nil)
		return NewYugenProvider(), nil
	case "hdrezka":
		logger.Info("Using HDRezka provider", nil)
		return NewHDRezkaProvider(), nil
	case "aniworld":
		logger.Info("Using AniWorld provider", nil)
		return NewAniWorldProvider(), nil
	default:
		logger.Error("Unknown provider", nil, map[string]interface{}{
			"provider": name,
		})
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}

