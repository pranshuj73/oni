package player

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pranshuj73/oni/config"
	"github.com/pranshuj73/oni/logger"
)

// HistoryEntry represents a watch history entry
type HistoryEntry struct {
	MediaID       int
	Progress      int
	EpisodesTotal int
	Timestamp     string // Resume timestamp (where you stopped watching)
	Duration      string // Total duration of the episode (HH:MM:SS format)
	LastWatched   string // Last watched timestamp (when you last completed an episode)
	Title         string
}

// LoadHistory loads the watch history
func LoadHistory() ([]HistoryEntry, error) {
	return LoadHistoryWithConfig(nil)
}

// LoadHistoryWithConfig loads the watch history (incognito or normal)
func LoadHistoryWithConfig(cfg *config.Config) ([]HistoryEntry, error) {
	incognito := false
	if cfg != nil {
		// Try to extract incognito state from config if available
		// This is a fallback for compatibility
	}
	return LoadHistoryWithIncognito(incognito)
}

// LoadHistoryWithIncognito loads the watch history (incognito or normal)
func LoadHistoryWithIncognito(incognito bool) ([]HistoryEntry, error) {
	logger.Debug("Loading watch history", map[string]interface{}{
		"incognito": incognito,
	})

	historyPath, err := GetHistoryPathWithIncognito(incognito)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("History file does not exist", map[string]interface{}{
				"path":      historyPath,
				"incognito": incognito,
			})
			return []HistoryEntry{}, nil
		}
		logger.Error("Failed to open history file", err, map[string]interface{}{
			"path":      historyPath,
			"incognito": incognito,
		})
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}
	defer file.Close()

	var entries []HistoryEntry

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")

		if len(parts) < 4 {
			continue
		}

		mediaID, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		progressParts := strings.Split(parts[1], "/")
		if len(progressParts) < 2 {
			continue
		}

		progress, err := strconv.Atoi(progressParts[0])
		if err != nil {
			continue
		}

		episodesTotal, err := strconv.Atoi(progressParts[1])
		if err != nil {
			continue
		}

		// Parse timestamp (resume point)
		timestamp := parts[2]
		
		// Parse Duration, LastWatched and Title
		// Format: MediaID\tProgress/EpisodesTotal\tTimestamp\tDuration\tLastWatched\tTitle
		// For backward compatibility:
		// - Old format (4 parts): MediaID\tProgress/EpisodesTotal\tTimestamp\tTitle
		// - Old format (5 parts): MediaID\tProgress/EpisodesTotal\tTimestamp\tLastWatched\tTitle
		// - New format (6+ parts): MediaID\tProgress/EpisodesTotal\tTimestamp\tDuration\tLastWatched\tTitle
		var duration string
		var lastWatched string
		var title string
		if len(parts) >= 6 {
			// New format with Duration
			duration = parts[3]
			lastWatched = parts[4]
			title = strings.Join(parts[5:], "\t")
		} else if len(parts) >= 5 {
			// Old format with LastWatched but no Duration
			duration = "" // No duration stored
			lastWatched = parts[3]
			title = strings.Join(parts[4:], "\t")
		} else {
			// Oldest format without LastWatched or Duration
			duration = "" // No duration stored
			lastWatched = parts[2]
			title = strings.Join(parts[3:], "\t")
		}

		entry := HistoryEntry{
			MediaID:       mediaID,
			Progress:      progress,
			EpisodesTotal: episodesTotal,
			Timestamp:     timestamp,
			Duration:      duration,
			LastWatched:   lastWatched,
			Title:         title,
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		logger.Error("Failed to scan history file", err, map[string]interface{}{
			"path":      historyPath,
			"incognito": incognito,
		})
		return nil, fmt.Errorf("failed to scan history file: %w", err)
	}

	logger.Info("Watch history loaded", map[string]interface{}{
		"path":         historyPath,
		"incognito":    incognito,
		"entriesCount": len(entries),
	})

	return entries, nil
}

// SaveHistoryEntry saves or updates a history entry
func SaveHistoryEntry(entry HistoryEntry) error {
	return SaveHistoryEntryWithConfig(entry, nil)
}

// SaveHistoryEntryWithConfig saves or updates a history entry (incognito or normal)
func SaveHistoryEntryWithConfig(entry HistoryEntry, cfg *config.Config) error {
	incognito := false
	if cfg != nil {
		// Try to extract incognito state from config if available
		// This is a fallback for compatibility
	}
	return SaveHistoryEntryWithIncognito(entry, incognito)
}

// SaveHistoryEntryWithIncognito saves or updates a history entry (incognito or normal)
func SaveHistoryEntryWithIncognito(entry HistoryEntry, incognito bool) error {
	logger.Debug("Saving history entry", map[string]interface{}{
		"mediaID":   entry.MediaID,
		"progress":  entry.Progress,
		"title":     entry.Title,
		"incognito": incognito,
	})

	historyPath, err := GetHistoryPathWithIncognito(incognito)
	if err != nil {
		return err
	}

	// Load existing history
	entries, err := LoadHistoryWithIncognito(incognito)
	if err != nil {
		return err
	}

	// Find and update existing entry or append new one
	found := false
	for i, e := range entries {
		if e.MediaID == entry.MediaID {
			entries[i] = entry
			found = true
			logger.Debug("Updated existing history entry", map[string]interface{}{
				"mediaID": entry.MediaID,
			})
			break
		}
	}

	if !found {
		entries = append(entries, entry)
		logger.Debug("Added new history entry", map[string]interface{}{
			"mediaID": entry.MediaID,
		})
	}

	// Write back to file
	file, err := os.Create(historyPath)
	if err != nil {
		logger.Error("Failed to create history file", err, map[string]interface{}{
			"path":      historyPath,
			"incognito": incognito,
		})
		return fmt.Errorf("failed to create history file: %w", err)
	}
	defer file.Close()

	for _, e := range entries {
		// Format: MediaID\tProgress/EpisodesTotal\tTimestamp\tDuration\tLastWatched\tTitle
		line := fmt.Sprintf("%d\t%d/%d\t%s\t%s\t%s\t%s\n",
			e.MediaID,
			e.Progress,
			e.EpisodesTotal,
			e.Timestamp,
			e.Duration,
			e.LastWatched,
			e.Title,
		)
		if _, err := file.WriteString(line); err != nil {
			logger.Error("Failed to write history entry", err, map[string]interface{}{
				"mediaID":   e.MediaID,
				"incognito": incognito,
			})
			return fmt.Errorf("failed to write history entry: %w", err)
		}
	}

	logger.Info("History entry saved successfully", map[string]interface{}{
		"mediaID":   entry.MediaID,
		"progress":  entry.Progress,
		"incognito": incognito,
		"path":      historyPath,
	})

	return nil
}

// DeleteHistoryEntry deletes a history entry
func DeleteHistoryEntry(mediaID int) error {
	historyPath, err := GetHistoryPath()
	if err != nil {
		return err
	}

	// Load existing history
	entries, err := LoadHistory()
	if err != nil {
		return err
	}

	// Filter out the entry to delete
	var newEntries []HistoryEntry
	for _, e := range entries {
		if e.MediaID != mediaID {
			newEntries = append(newEntries, e)
		}
	}

	// Write back to file
	file, err := os.Create(historyPath)
	if err != nil {
		return fmt.Errorf("failed to create history file: %w", err)
	}
	defer file.Close()

	for _, e := range newEntries {
		line := fmt.Sprintf("%d\t%d/%d\t%s\t%s\n",
			e.MediaID,
			e.Progress,
			e.EpisodesTotal,
			e.Timestamp,
			e.Title,
		)
		if _, err := file.WriteString(line); err != nil {
			return fmt.Errorf("failed to write history entry: %w", err)
		}
	}

	return nil
}

// GetHistoryEntry gets a specific history entry (defaults to normal history)
func GetHistoryEntry(mediaID int, episode int) (*HistoryEntry, error) {
	return GetHistoryEntryWithIncognito(mediaID, episode, false)
}

// GetHistoryEntryWithIncognito gets a specific history entry (incognito or normal)
func GetHistoryEntryWithIncognito(mediaID int, episode int, incognito bool) (*HistoryEntry, error) {
	entries, err := LoadHistoryWithIncognito(incognito)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.MediaID == mediaID && e.Progress == episode {
			return &e, nil
		}
	}

	return nil, nil
}

