package player

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pranshuj73/oni/config"
	"github.com/pranshuj73/oni/logger"
)

// HistoryEntry represents a watch history entry
type HistoryEntry struct {
	MediaID       int    `json:"media_id"`
	Progress      int    `json:"progress"`
	EpisodesTotal int    `json:"episodes_total"`
	Timestamp     string `json:"timestamp"`      // Resume timestamp (where you stopped watching)
	Duration      string `json:"duration"`       // Total duration of the episode (HH:MM:SS format)
	LastWatched   string `json:"last_watched"`   // Last watched timestamp (when you last completed an episode)
	Title         string `json:"title"`
}

// HistoryFile represents the JSON history file structure
type HistoryFile struct {
	Version int            `json:"version"` // File format version for future migrations
	Entries []HistoryEntry `json:"entries"`
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

	data, err := os.ReadFile(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("History file does not exist", map[string]interface{}{
				"path":      historyPath,
				"incognito": incognito,
			})
			return []HistoryEntry{}, nil
		}
		logger.Error("Failed to read history file", err, map[string]interface{}{
			"path":      historyPath,
			"incognito": incognito,
		})
		return nil, fmt.Errorf("failed to read history file: %w", err)
	}

	// Try to parse as JSON first (new format)
	var historyFile HistoryFile
	if err := json.Unmarshal(data, &historyFile); err == nil {
		logger.Info("Watch history loaded (JSON format)", map[string]interface{}{
			"path":         historyPath,
			"incognito":    incognito,
			"version":      historyFile.Version,
			"entriesCount": len(historyFile.Entries),
		})
		return historyFile.Entries, nil
	}

	// Fallback: Try to parse as old tab-separated format and migrate
	logger.Info("Migrating history from old tab-separated format to JSON", map[string]interface{}{
		"path":      historyPath,
		"incognito": incognito,
	})

	entries, err := migrateOldHistoryFormat(string(data))
	if err != nil {
		logger.Error("Failed to migrate old history format", err, map[string]interface{}{
			"path":      historyPath,
			"incognito": incognito,
		})
		return nil, fmt.Errorf("failed to parse history file: %w", err)
	}

	// Save migrated data in JSON format
	if len(entries) > 0 {
		if err := saveHistoryToFile(historyPath, entries); err != nil {
			logger.Warn("Failed to save migrated history", map[string]interface{}{
				"path":  historyPath,
				"error": err.Error(),
			})
		} else {
			logger.Info("Successfully migrated history to JSON format", map[string]interface{}{
				"path":         historyPath,
				"entriesCount": len(entries),
			})
		}
	}

	return entries, nil
}

// migrateOldHistoryFormat migrates old tab-separated format to HistoryEntry slice
func migrateOldHistoryFormat(data string) ([]HistoryEntry, error) {
	var entries []HistoryEntry
	scanner := bufio.NewScanner(strings.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

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
			duration = parts[3]
			lastWatched = parts[4]
			title = strings.Join(parts[5:], "\t")
		} else if len(parts) >= 5 {
			duration = ""
			lastWatched = parts[3]
			title = strings.Join(parts[4:], "\t")
		} else {
			duration = ""
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
		return nil, fmt.Errorf("failed to scan old format: %w", err)
	}

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

	// Save to file
	if err := saveHistoryToFile(historyPath, entries); err != nil {
		logger.Error("Failed to save history file", err, map[string]interface{}{
			"path":      historyPath,
			"incognito": incognito,
		})
		return err
	}

	logger.Info("History entry saved successfully", map[string]interface{}{
		"mediaID":   entry.MediaID,
		"progress":  entry.Progress,
		"incognito": incognito,
		"path":      historyPath,
	})

	return nil
}

// saveHistoryToFile saves history entries to a JSON file with atomic write
func saveHistoryToFile(historyPath string, entries []HistoryEntry) error {
	historyFile := HistoryFile{
		Version: 1,
		Entries: entries,
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(historyFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	// Atomic write: write to temp file, then rename
	dir := filepath.Dir(historyPath)
	tmpFile, err := os.CreateTemp(dir, "history-*.json.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up temp file if rename fails

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, historyPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// DeleteHistoryEntry deletes a history entry
func DeleteHistoryEntry(mediaID int) error {
	logger.Debug("Deleting history entry", map[string]interface{}{
		"mediaID": mediaID,
	})

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
	deleted := false
	for _, e := range entries {
		if e.MediaID != mediaID {
			newEntries = append(newEntries, e)
		} else {
			deleted = true
		}
	}

	if !deleted {
		logger.Debug("History entry not found", map[string]interface{}{
			"mediaID": mediaID,
		})
		return nil // Entry not found, nothing to delete
	}

	// Save updated history
	if err := saveHistoryToFile(historyPath, newEntries); err != nil {
		logger.Error("Failed to save history after deletion", err, map[string]interface{}{
			"mediaID": mediaID,
			"path":    historyPath,
		})
		return fmt.Errorf("failed to save history: %w", err)
	}

	logger.Info("History entry deleted successfully", map[string]interface{}{
		"mediaID": mediaID,
		"path":    historyPath,
	})

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

