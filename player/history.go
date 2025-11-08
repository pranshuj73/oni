package player

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// HistoryEntry represents a watch history entry
type HistoryEntry struct {
	MediaID       int
	Progress      int
	EpisodesTotal int
	Timestamp     string
	Title         string
}

// LoadHistory loads the watch history
func LoadHistory() ([]HistoryEntry, error) {
	historyPath, err := GetHistoryPath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []HistoryEntry{}, nil
		}
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

		entry := HistoryEntry{
			MediaID:       mediaID,
			Progress:      progress,
			EpisodesTotal: episodesTotal,
			Timestamp:     parts[2],
			Title:         parts[3],
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan history file: %w", err)
	}

	return entries, nil
}

// SaveHistoryEntry saves or updates a history entry
func SaveHistoryEntry(entry HistoryEntry) error {
	historyPath, err := GetHistoryPath()
	if err != nil {
		return err
	}

	// Load existing history
	entries, err := LoadHistory()
	if err != nil {
		return err
	}

	// Find and update existing entry or append new one
	found := false
	for i, e := range entries {
		if e.MediaID == entry.MediaID {
			entries[i] = entry
			found = true
			break
		}
	}

	if !found {
		entries = append(entries, entry)
	}

	// Write back to file
	file, err := os.Create(historyPath)
	if err != nil {
		return fmt.Errorf("failed to create history file: %w", err)
	}
	defer file.Close()

	for _, e := range entries {
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

// GetHistoryEntry gets a specific history entry
func GetHistoryEntry(mediaID int, episode int) (*HistoryEntry, error) {
	entries, err := LoadHistory()
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

