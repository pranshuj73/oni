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

// GetProvider returns a provider by name, wrapped with retry logic
func GetProvider(name string) (Provider, error) {
	logger.Debug("Getting provider", map[string]interface{}{
		"provider": name,
	})

	var baseProvider Provider

	switch name {
	case "allanime":
		logger.Info("Using AllAnime provider", nil)
		baseProvider = NewAllAnimeProvider()
	case "aniwatch":
		logger.Info("Using AniWatch provider", nil)
		baseProvider = NewAniWatchProvider()
	case "yugen":
		logger.Info("Using Yugen provider", nil)
		baseProvider = NewYugenProvider()
	case "hdrezka":
		logger.Info("Using HDRezka provider", nil)
		baseProvider = NewHDRezkaProvider()
	case "aniworld":
		logger.Info("Using AniWorld provider", nil)
		baseProvider = NewAniWorldProvider()
	default:
		logger.Error("Unknown provider", nil, map[string]interface{}{
			"provider": name,
		})
		return nil, fmt.Errorf("unknown provider: %s", name)
	}

	// Wrap provider with retry logic
	retryConfig := DefaultRetryConfig()
	logger.Debug("Wrapping provider with retry logic", map[string]interface{}{
		"provider":   name,
		"maxRetries": retryConfig.MaxRetries,
		"baseDelay":  retryConfig.BaseDelay.String(),
	})

	return NewProviderWithRetry(baseProvider, retryConfig), nil
}

