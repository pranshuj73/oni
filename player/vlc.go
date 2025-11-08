package player

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/pranshuj73/oni/config"
	"github.com/pranshuj73/oni/providers"
)

// VLCPlayer implements VLC player
type VLCPlayer struct {
	cfg *config.Config
}

// NewVLCPlayer creates a new VLC player
func NewVLCPlayer(cfg *config.Config) *VLCPlayer {
	return &VLCPlayer{
		cfg: cfg,
	}
}

// Name returns the player name
func (p *VLCPlayer) Name() string {
	return "vlc"
}

// Play plays a video using VLC
func (p *VLCPlayer) Play(ctx context.Context, videoData *providers.VideoData, title string, resumeFrom string) (*PlaybackInfo, error) {
	args := []string{
		"--play-and-exit",
		fmt.Sprintf("--meta-title=%s", title),
		videoData.VideoURL,
	}

	cmd := exec.CommandContext(ctx, "vlc", args...)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run vlc: %w", err)
	}

	// VLC doesn't provide playback position info easily
	// Return as completed for now
	return &PlaybackInfo{
		StoppedAt:           "00:00:00",
		PercentageProgress:  100,
		CompletedSuccessful: true,
	}, nil
}

// IINAPlayer implements IINA player (macOS)
type IINAPlayer struct {
	cfg *config.Config
}

// NewIINAPlayer creates a new IINA player
func NewIINAPlayer(cfg *config.Config) *IINAPlayer {
	return &IINAPlayer{
		cfg: cfg,
	}
}

// Name returns the player name
func (p *IINAPlayer) Name() string {
	return "iina"
}

// Play plays a video using IINA
func (p *IINAPlayer) Play(ctx context.Context, videoData *providers.VideoData, title string, resumeFrom string) (*PlaybackInfo, error) {
	args := []string{
		"--no-stdin",
		"--keep-running",
		fmt.Sprintf("--mpv-force-media-title=%s", title),
		videoData.VideoURL,
	}

	cmd := exec.CommandContext(ctx, "iina", args...)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run iina: %w", err)
	}

	// IINA doesn't provide playback position info easily
	// Return as completed for now
	return &PlaybackInfo{
		StoppedAt:           "00:00:00",
		PercentageProgress:  100,
		CompletedSuccessful: true,
	}, nil
}

