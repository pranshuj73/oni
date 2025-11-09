package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

// GetConfigPath returns the path to the configuration file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".oni")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDir, "config.ini"), nil
}

// GetDataDir returns the path to the data directory
func GetDataDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	dataDir := filepath.Join(homeDir, ".oni")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create data directory: %w", err)
	}

	return dataDir, nil
}

// Load reads the configuration from the INI file
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// Create default config
	cfg := &Config{
		Player: PlayerConfig{
			Player:          "mpv",
			PlayerArguments: "",
		},
		Provider: ProviderConfig{
			Provider:    "allanime",
			DownloadDir: "",
			Quality:     "1080",
		},
		AniList: AniListConfig{
			NoAniList:         false,
			ScoreOnCompletion: false,
		},
		UI: UIConfig{
			UseExternalMenu: false,
			ImagePreview:    false,
			JSONOutput:      false,
		},
		Playback: PlaybackConfig{
			SubOrDub:              "sub",
			SubsLanguage:          "english",
			PersistIncognitoSessions: false,
		},
		Discord: DiscordConfig{
			DiscordPresence: false,
		},
		Advanced: AdvancedConfig{
			ShowAdultContent: false,
		},
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config file
		if err := Save(cfg); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return cfg, nil
	}

	// Load existing config
	iniFile, err := ini.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := iniFile.MapTo(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}

// Save writes the configuration to the INI file
func Save(cfg *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	iniFile := ini.Empty()
	if err := iniFile.ReflectFrom(cfg); err != nil {
		return fmt.Errorf("failed to reflect config: %w", err)
	}

	if err := iniFile.SaveTo(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

