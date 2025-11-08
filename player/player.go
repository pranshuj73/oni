package player

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pranshuj73/oni/config"
	"github.com/pranshuj73/oni/providers"
)

// Player defines the interface for video players
type Player interface {
	// Play plays a video with the given data
	Play(ctx context.Context, videoData *providers.VideoData, title string, resumeFrom string) (*PlaybackInfo, error)

	// Name returns the player name
	Name() string
}

// PlaybackInfo contains information about a playback session
type PlaybackInfo struct {
	StoppedAt           string
	PercentageProgress  int
	CompletedSuccessful bool
}

// GetPlayer returns a player by name
func GetPlayer(cfg *config.Config) (Player, error) {
	switch cfg.Player.Player {
	case "mpv", "mpv.exe":
		return NewMPVPlayer(cfg), nil
	case "vlc":
		return NewVLCPlayer(cfg), nil
	case "iina":
		return NewIINAPlayer(cfg), nil
	default:
		return nil, fmt.Errorf("unknown player: %s", cfg.Player.Player)
	}
}

// GetHistoryPath returns the path to the history file
func GetHistoryPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	dataDir := filepath.Join(homeDir, ".oni")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create data directory: %w", err)
	}

	return filepath.Join(dataDir, "history.txt"), nil
}

