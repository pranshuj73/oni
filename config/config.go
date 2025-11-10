package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pranshuj73/oni/logger"
	"gopkg.in/ini.v1"
)

// GetConfigPath returns the path to the configuration file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("Failed to get home directory", err, nil)
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".oni")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		logger.Error("Failed to create config directory", err, map[string]interface{}{
			"path": configDir,
		})
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.ini")
	logger.Debug("Config path resolved", map[string]interface{}{
		"path": configPath,
	})

	return configPath, nil
}

// GetDataDir returns the path to the data directory
func GetDataDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error("Failed to get home directory", err, nil)
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	dataDir := filepath.Join(homeDir, ".oni")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Error("Failed to create data directory", err, map[string]interface{}{
			"path": dataDir,
		})
		return "", fmt.Errorf("failed to create data directory: %w", err)
	}

	logger.Debug("Data directory resolved", map[string]interface{}{
		"path": dataDir,
	})

	return dataDir, nil
}

// Load reads the configuration from the INI file
func Load() (*Config, error) {
	logger.Debug("Loading configuration", nil)

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
		logger.Info("Config file does not exist, creating default", map[string]interface{}{
			"path": configPath,
		})
		// Create default config file
		if err := Save(cfg); err != nil {
			logger.Error("Failed to create default config", err, map[string]interface{}{
				"path": configPath,
			})
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		logger.Info("Default config created", map[string]interface{}{
			"path": configPath,
		})
		return cfg, nil
	}

	logger.Debug("Loading existing config file", map[string]interface{}{
		"path": configPath,
	})

	// Load existing config
	iniFile, err := ini.Load(configPath)
	if err != nil {
		logger.Error("Failed to load config file", err, map[string]interface{}{
			"path": configPath,
		})
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := iniFile.MapTo(cfg); err != nil {
		logger.Error("Failed to parse config", err, map[string]interface{}{
			"path": configPath,
		})
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	logger.Info("Configuration loaded successfully", map[string]interface{}{
		"path":     configPath,
		"player":   cfg.Player.Player,
		"provider": cfg.Provider.Provider,
		"quality":  cfg.Provider.Quality,
	})

	return cfg, nil
}

// Save writes the configuration to the INI file
func Save(cfg *Config) error {
	logger.Debug("Saving configuration", nil)

	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	iniFile := ini.Empty()
	if err := iniFile.ReflectFrom(cfg); err != nil {
		logger.Error("Failed to reflect config to INI", err, map[string]interface{}{
			"path": configPath,
		})
		return fmt.Errorf("failed to reflect config: %w", err)
	}

	if err := iniFile.SaveTo(configPath); err != nil {
		logger.Error("Failed to save config file", err, map[string]interface{}{
			"path": configPath,
		})
		return fmt.Errorf("failed to save config: %w", err)
	}

	logger.Info("Configuration saved successfully", map[string]interface{}{
		"path": configPath,
	})

	return nil
}

