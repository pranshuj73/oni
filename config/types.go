package config

import (
	"fmt"
	"strings"
)

// Config represents the complete application configuration
type Config struct {
	Player   PlayerConfig   `ini:"player"`
	Provider ProviderConfig `ini:"provider"`
	AniList  AniListConfig  `ini:"anilist"`
	UI       UIConfig       `ini:"ui"`
	Playback PlaybackConfig `ini:"playback"`
	Discord  DiscordConfig  `ini:"discord"`
	Advanced AdvancedConfig `ini:"advanced"`
}

// PlayerConfig contains player-related settings
type PlayerConfig struct {
	Player          string `ini:"player"`
	PlayerArguments string `ini:"player_arguments"`
}

// ProviderConfig contains provider-related settings
type ProviderConfig struct {
	Provider     string `ini:"provider"`
	DownloadDir  string `ini:"download_dir"`
	Quality      string `ini:"quality"`
}

// AniListConfig contains AniList integration settings
type AniListConfig struct {
	NoAniList          bool `ini:"no_anilist"`
	ScoreOnCompletion  bool `ini:"score_on_completion"`
}

// UIConfig contains UI-related settings
type UIConfig struct {
	UseExternalMenu bool `ini:"use_external_menu"`
	ImagePreview    bool `ini:"image_preview"`
	JSONOutput      bool `ini:"json_output"`
}

// PlaybackConfig contains playback-related settings
type PlaybackConfig struct {
	SubOrDub              string `ini:"sub_or_dub"`
	SubsLanguage          string `ini:"subs_language"`
	PersistIncognitoSessions bool `ini:"persist_incognito_sessions"`
}

// DiscordConfig contains Discord presence settings
type DiscordConfig struct {
	DiscordPresence bool `ini:"discord_presence"`
}

// AdvancedConfig contains advanced settings
type AdvancedConfig struct {
	ShowAdultContent bool `ini:"show_adult_content"`
}

// Validate validates all configuration values
func (c *Config) Validate() error {
	// Validate player
	validPlayers := []string{"mpv", "vlc", "iina"}
	if !contains(validPlayers, c.Player.Player) {
		return fmt.Errorf("invalid player '%s': must be one of [%s]",
			c.Player.Player, strings.Join(validPlayers, ", "))
	}

	// Validate provider
	validProviders := []string{"allanime", "aniwatch", "yugen", "hdrezka", "aniworld"}
	if !contains(validProviders, c.Provider.Provider) {
		return fmt.Errorf("invalid provider '%s': must be one of [%s]",
			c.Provider.Provider, strings.Join(validProviders, ", "))
	}

	// Validate quality
	validQualities := []string{"1080", "720", "480", "360"}
	if !contains(validQualities, c.Provider.Quality) {
		return fmt.Errorf("invalid quality '%s': must be one of [%s]",
			c.Provider.Quality, strings.Join(validQualities, ", "))
	}

	// Validate sub_or_dub
	validSubOrDub := []string{"sub", "dub"}
	if !contains(validSubOrDub, c.Playback.SubOrDub) {
		return fmt.Errorf("invalid sub_or_dub '%s': must be one of [%s]",
			c.Playback.SubOrDub, strings.Join(validSubOrDub, ", "))
	}

	return nil
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

