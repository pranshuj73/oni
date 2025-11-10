package player

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pranshuj73/oni/config"
	"github.com/pranshuj73/oni/logger"
	"github.com/pranshuj73/oni/providers"
)

// MPVPlayer implements MPV player
type MPVPlayer struct {
	cfg *config.Config
}

// NewMPVPlayer creates a new MPV player
func NewMPVPlayer(cfg *config.Config) *MPVPlayer {
	return &MPVPlayer{
		cfg: cfg,
	}
}

// Name returns the player name
func (p *MPVPlayer) Name() string {
	return "mpv"
}

// Play plays a video using MPV
func (p *MPVPlayer) Play(ctx context.Context, videoData *providers.VideoData, title string, resumeFrom string) (*PlaybackInfo, error) {
	logger.Info("Starting MPV player", map[string]interface{}{
		"title":             title,
		"resumeFrom":        resumeFrom,
		"hasSubtitles":      len(videoData.SubtitleURLs) > 0,
		"subtitlesCount":    len(videoData.SubtitleURLs),
		"hasReferer":        videoData.Referer != "",
		"hasCustomArgs":     p.cfg.Player.PlayerArguments != "",
	})

	// Create temp file for output
	tmpFile := filepath.Join(os.TempDir(), "oni_mpv_output.txt")

	args := []string{videoData.VideoURL}

	// Add custom player arguments
	if p.cfg.Player.PlayerArguments != "" {
		customArgs := strings.Fields(p.cfg.Player.PlayerArguments)
		args = append(args, customArgs...)
		logger.Debug("Added custom player arguments", map[string]interface{}{
			"args": p.cfg.Player.PlayerArguments,
		})
	}

	// Add referer if needed
	if videoData.Referer != "" {
		args = append(args, fmt.Sprintf("--http-header-fields-append=Referer:%s", videoData.Referer))
		logger.Debug("Added referer header", map[string]interface{}{
			"referer": videoData.Referer,
		})
	}

	// Add resume position
	if resumeFrom != "" && resumeFrom != "00:00:00" {
		args = append(args, fmt.Sprintf("--start=%s", resumeFrom))
		logger.Debug("Added resume position", map[string]interface{}{
			"position": resumeFrom,
		})
	}

	// Add title
	args = append(args, fmt.Sprintf("--force-media-title=%s", title))

	// Add subtitles if available
	if len(videoData.SubtitleURLs) > 0 {
		if len(videoData.SubtitleURLs) == 1 {
			args = append(args, "--sub-file="+videoData.SubtitleURLs[0])
		} else {
			subFiles := strings.Join(videoData.SubtitleURLs, ":")
			args = append(args, "--sub-files="+subFiles)
		}
		logger.Debug("Added subtitles", map[string]interface{}{
			"count": len(videoData.SubtitleURLs),
		})
	}

	// Reduce output verbosity
	args = append(args, "--msg-level=ffmpeg/demuxer=error")

	// Create command
	cmd := exec.CommandContext(ctx, p.cfg.Player.Player, args...)

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error("Failed to create stdout pipe", err, nil)
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		logger.Error("Failed to create stderr pipe", err, nil)
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		logger.Error("Failed to start MPV", err, map[string]interface{}{
			"player": p.cfg.Player.Player,
			"title":  title,
		})
		return nil, fmt.Errorf("failed to start mpv: %w", err)
	}

	logger.Debug("MPV process started successfully", map[string]interface{}{
		"pid": cmd.Process.Pid,
	})

	// Create file to write output
	outFile, err := os.Create(tmpFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Read output in goroutines
	done := make(chan bool)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			outFile.WriteString(line + "\n")
		}
		done <- true
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			outFile.WriteString(line + "\n")
		}
		done <- true
	}()

	// Wait for command to finish
	<-done
	<-done

	if err := cmd.Wait(); err != nil {
		// MPV might exit with non-zero even on normal close
		// Don't treat this as an error
		logger.Debug("MPV exited with error (may be normal)", map[string]interface{}{
			"error": err.Error(),
		})
	}

	logger.Debug("MPV process ended", nil)

	// Parse output file for playback position
	playbackInfo, err := p.parseOutput(tmpFile)
	if err != nil {
		logger.Warn("Failed to parse MPV output", map[string]interface{}{
			"error": err.Error(),
		})
		return &PlaybackInfo{
			StoppedAt:          "00:00:00",
			PercentageProgress: 0,
		}, nil
	}

	logger.Info("MPV playback completed", map[string]interface{}{
		"stoppedAt":           playbackInfo.StoppedAt,
		"percentageProgress":  playbackInfo.PercentageProgress,
		"completedSuccessful": playbackInfo.CompletedSuccessful,
	})

	return playbackInfo, nil
}

// parseOutput parses MPV output to extract playback information
func (p *MPVPlayer) parseOutput(filePath string) (*PlaybackInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open output file: %w", err)
	}
	defer file.Close()

	var lastPosition string
	var lastTotalDuration string
	var lastPercentage int

	// Regular expression to match: AV: 00:01:23 / 00:24:56 (5%)
	re := regexp.MustCompile(`AV:\s+([0-9:]+)\s+/\s+([0-9:]+)\s+\(([0-9]+)%\)`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 4 {
			lastPosition = matches[1]
			lastTotalDuration = matches[2] // Extract total duration
			lastPercentage, _ = strconv.Atoi(matches[3])
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan output file: %w", err)
	}

	if lastPosition == "" {
		return &PlaybackInfo{
			StoppedAt:          "00:00:00",
			TotalDuration:      "",
			PercentageProgress: 0,
		}, nil
	}

	return &PlaybackInfo{
		StoppedAt:           lastPosition,
		TotalDuration:       lastTotalDuration,
		PercentageProgress:  lastPercentage,
		CompletedSuccessful: lastPercentage >= 85,
	}, nil
}

