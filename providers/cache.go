package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

// ProviderCacheEntry represents a cached provider mapping
type ProviderCacheEntry struct {
	ProviderID string
	Title      string
	LastUsed   time.Time
}

var cacheFile *ini.File
var cacheInitialized bool

// getCachePath returns the path to the provider cache file
func getCachePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(homeDir, ".oni")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "provider_cache.ini"), nil
}

// initCache initializes the cache file
func initCache() error {
	if cacheInitialized {
		return nil
	}
	cacheInitialized = true

	cachePath, err := getCachePath()
	if err != nil {
		return err
	}

	// Load existing cache or create new one
	cacheFile, err = ini.Load(cachePath)
	if err != nil {
		// File doesn't exist, create new
		cacheFile = ini.Empty()
	}

	return nil
}

// LoadProviderMapping loads a cached provider mapping
func LoadProviderMapping(provider string, mediaID int) (*ProviderCacheEntry, error) {
	if err := initCache(); err != nil {
		return nil, err
	}

	section, err := cacheFile.GetSection(provider)
	if err != nil {
		// Section doesn't exist
		return nil, nil
	}

	key := fmt.Sprintf("%d", mediaID)
	value := section.Key(key).String()
	if value == "" {
		return nil, nil
	}

	// Parse pipe-separated format: provider_id|title|last_used
	parts := strings.Split(value, "|")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid cache entry format")
	}

	lastUsed, err := time.Parse(time.RFC3339, parts[2])
	if err != nil {
		// Try parsing without timezone for backward compatibility
		lastUsed, err = time.Parse("2006-01-02T15:04:05", parts[2])
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp format: %w", err)
		}
	}

	return &ProviderCacheEntry{
		ProviderID: parts[0],
		Title:      parts[1],
		LastUsed:   lastUsed,
	}, nil
}

// SaveProviderMapping saves a provider mapping to cache
func SaveProviderMapping(provider string, mediaID int, providerID string, title string) error {
	if err := initCache(); err != nil {
		return err
	}

	section, err := cacheFile.GetSection(provider)
	if err != nil {
		// Section doesn't exist, create it
		section, err = cacheFile.NewSection(provider)
		if err != nil {
			return fmt.Errorf("failed to create section: %w", err)
		}
	}

	key := fmt.Sprintf("%d", mediaID)
	timestamp := time.Now().UTC().Format(time.RFC3339)
	value := fmt.Sprintf("%s|%s|%s", providerID, title, timestamp)

	section.Key(key).SetValue(value)

	// Save to file
	cachePath, err := getCachePath()
	if err != nil {
		return err
	}

	return cacheFile.SaveTo(cachePath)
}

// ClearProviderMapping clears a specific provider mapping
func ClearProviderMapping(provider string, mediaID int) error {
	if err := initCache(); err != nil {
		return err
	}

	section, err := cacheFile.GetSection(provider)
	if err != nil {
		// Section doesn't exist, nothing to clear
		return nil
	}

	key := fmt.Sprintf("%d", mediaID)
	section.DeleteKey(key)

	cachePath, err := getCachePath()
	if err != nil {
		return err
	}

	return cacheFile.SaveTo(cachePath)
}

// ClearAllProviderMappings clears all provider mappings
func ClearAllProviderMappings() error {
	cachePath, err := getCachePath()
	if err != nil {
		return err
	}

	// Delete the cache file
	return os.Remove(cachePath)
}

