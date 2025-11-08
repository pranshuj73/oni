package anilist

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetTokenPath returns the path to the AniList token file
func GetTokenPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	dataDir := filepath.Join(homeDir, ".oni")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create data directory: %w", err)
	}

	return filepath.Join(dataDir, "anilist_token.txt"), nil
}

// GetUserIDPath returns the path to the AniList user ID file
func GetUserIDPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	dataDir := filepath.Join(homeDir, ".oni")
	return filepath.Join(dataDir, "anilist_user_id.txt"), nil
}

// LoadToken loads the AniList access token from file
func LoadToken() (string, error) {
	tokenPath, err := GetTokenPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read token: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

// SaveToken saves the AniList access token to file
func SaveToken(token string) error {
	tokenPath, err := GetTokenPath()
	if err != nil {
		return err
	}

	if err := os.WriteFile(tokenPath, []byte(token), 0600); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	return nil
}

// LoadUserID loads the AniList user ID from file
func LoadUserID() (int, error) {
	userIDPath, err := GetUserIDPath()
	if err != nil {
		return 0, err
	}

	data, err := os.ReadFile(userIDPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read user ID: %w", err)
	}

	var userID int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &userID); err != nil {
		return 0, fmt.Errorf("failed to parse user ID: %w", err)
	}

	return userID, nil
}

// SaveUserID saves the AniList user ID to file
func SaveUserID(userID int) error {
	userIDPath, err := GetUserIDPath()
	if err != nil {
		return err
	}

	if err := os.WriteFile(userIDPath, []byte(fmt.Sprintf("%d", userID)), 0600); err != nil {
		return fmt.Errorf("failed to save user ID: %w", err)
	}

	return nil
}

// PromptForToken prompts the user to enter their AniList token (deprecated, use TUI version)
func PromptForToken() (string, error) {
	// This is now handled by the TUI
	return "", fmt.Errorf("please use the TUI authentication flow")
}

