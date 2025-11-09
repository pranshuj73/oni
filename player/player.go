package player

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pranshuj73/oni/config"
	"github.com/pranshuj73/oni/logger"
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
	logger.Debug("Getting player", map[string]interface{}{
		"player": cfg.Player.Player,
	})

	switch cfg.Player.Player {
	case "mpv", "mpv.exe":
		logger.Info("Using MPV player", nil)
		return NewMPVPlayer(cfg), nil
	case "vlc":
		logger.Info("Using VLC player", nil)
		return NewVLCPlayer(cfg), nil
	case "iina":
		logger.Info("Using IINA player", nil)
		return NewIINAPlayer(cfg), nil
	default:
		logger.Error("Unknown player", nil, map[string]interface{}{
			"player": cfg.Player.Player,
		})
		return nil, fmt.Errorf("unknown player: %s", cfg.Player.Player)
	}
}

// GetHistoryPath returns the path to the history file
func GetHistoryPath() (string, error) {
	return GetHistoryPathWithIncognito(false)
}

// GetHistoryPathWithConfig returns the path to the history file (incognito or normal)
// Note: This is kept for compatibility but incognito mode is now runtime-only
func GetHistoryPathWithConfig(cfg *config.Config) (string, error) {
	// Incognito mode is runtime-only, so this always returns normal history path
	return GetHistoryPathWithIncognito(false)
}

// GetHistoryPathWithIncognito returns the path to the history file (incognito or normal)
func GetHistoryPathWithIncognito(incognito bool) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	dataDir := filepath.Join(homeDir, ".oni")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create data directory: %w", err)
	}

	// Use incognito history if incognito mode is enabled
	if incognito {
		return filepath.Join(dataDir, "incognito_history.txt"), nil
	}

	return filepath.Join(dataDir, "history.txt"), nil
}

// DeleteIncognitoHistory deletes the incognito history file
func DeleteIncognitoHistory() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	incognitoPath := filepath.Join(homeDir, ".oni", "incognito_history.txt")
	if err := os.Remove(incognitoPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete incognito history: %w", err)
	}

	return nil
}

