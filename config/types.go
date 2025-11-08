package config

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
	SubOrDub     string `ini:"sub_or_dub"`
	SubsLanguage string `ini:"subs_language"`
}

// DiscordConfig contains Discord presence settings
type DiscordConfig struct {
	DiscordPresence bool `ini:"discord_presence"`
}

// AdvancedConfig contains advanced settings
type AdvancedConfig struct {
	ShowAdultContent bool `ini:"show_adult_content"`
}

