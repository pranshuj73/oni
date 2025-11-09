package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pranshuj73/oni/anilist"
	"github.com/pranshuj73/oni/config"
	"github.com/pranshuj73/oni/discord"
	"github.com/pranshuj73/oni/logger"
	"github.com/pranshuj73/oni/player"
	"github.com/pranshuj73/oni/providers"
	"github.com/pranshuj73/oni/ui"
)

const version = "2.0.0"

// AppState represents the current application state
type AppState int

const (
	StateMainMenu AppState = iota
	StateUpdateProgress
	StateEditConfig
	StateAnimeList
	StateEpisodeSelect
	StateAniListAuth
)

// App represents the main application model
type App struct {
	cfg            *config.Config
	client         *anilist.Client
	discordMgr     *discord.PresenceManager
	state          AppState
	currentModel   tea.Model
	mainMenu       *ui.MainMenu // Keep reference to main menu to preserve cursor
	selectedAnime  *anilist.Anime
	selectedEntry  *anilist.MediaListEntry
	selectedEp     int
	subOrDub       string
	err            error
	loadingMsg     string        // Central loading message
	spinner        spinner.Model // Central spinner
	width          int           // Terminal width
	height         int           // Terminal height
	autoplayMode   bool          // Whether we're in autoplay/binge mode
	lastAnimeID    int           // Track the last anime watched for session detection
	lastWatchTime  time.Time     // Track when the last episode was watched
	incognitoMode  bool          // Runtime incognito mode state
}

func main() {
	// Parse command-line flags
	var (
		showVersion    = flag.Bool("v", false, "Show version")
		showHelp       = flag.Bool("h", false, "Show help")
		editConfig     = flag.Bool("e", false, "Edit configuration")
		quality        = flag.String("q", "", "Video quality")
		provider       = flag.String("w", "", "Provider")
		subOrDub       = flag.String("sub-or-dub", "", "Sub or dub")
		discordPresence = flag.Bool("d", false, "Enable Discord presence")
	)

	flag.Parse()

	// Initialize logger
	if err := logger.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Application started", map[string]interface{}{
		"version": version,
	})

	if *showVersion {
		fmt.Printf("oni version %s\n", version)
		logger.Info("Version displayed", nil)
		os.Exit(0)
	}

	if *showHelp {
		showUsage()
		logger.Info("Help displayed", nil)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", err, nil)
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Configuration loaded", nil)

	// Apply command-line overrides
	if *quality != "" {
		cfg.Provider.Quality = *quality
		logger.Debug("Quality override applied", map[string]interface{}{
			"quality": *quality,
		})
	}
	if *provider != "" {
		cfg.Provider.Provider = *provider
		logger.Debug("Provider override applied", map[string]interface{}{
			"provider": *provider,
		})
	}
	if *subOrDub != "" {
		cfg.Playback.SubOrDub = *subOrDub
		logger.Debug("SubOrDub override applied", map[string]interface{}{
			"subOrDub": *subOrDub,
		})
	}
	if *discordPresence {
		cfg.Discord.DiscordPresence = true
		logger.Debug("Discord presence enabled via flag", nil)
	}

	// Handle config edit mode
	if *editConfig {
		logger.Info("Entering config edit mode", nil)
		p := tea.NewProgram(ui.NewConfigEditor(cfg))
		if _, err := p.Run(); err != nil {
			logger.Error("Config editor error", err, nil)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		logger.Info("Config editor closed", nil)
		os.Exit(0)
	}

	// Try to load existing AniList token
	var client *anilist.Client
	var needsAuth bool
	if !cfg.AniList.NoAniList {
		logger.Debug("Attempting to load AniList token", nil)
		token, err := anilist.LoadToken()
		if err == nil && token != "" {
			logger.Debug("AniList token found, creating client", nil)
			// Token exists, try to create client
			client, err = anilist.NewClient()
			if err != nil {
				// Token might be invalid, need re-auth
				logger.Warn("AniList client creation failed, auth required", map[string]interface{}{
					"error": err.Error(),
				})
				needsAuth = true
			} else {
				logger.Info("AniList client created successfully", nil)
			}
		} else {
			// No token, need auth
			logger.Debug("No AniList token found, auth required", nil)
			needsAuth = true
		}
	} else {
		logger.Info("AniList integration disabled", nil)
	}

	// Create Discord presence manager
	discordMgr := discord.NewPresenceManager(cfg.Discord.DiscordPresence)
	if cfg.Discord.DiscordPresence {
		logger.Debug("Attempting to connect to Discord", nil)
		if err := discordMgr.Connect(); err != nil {
			logger.Warn("Failed to connect to Discord", map[string]interface{}{
				"error": err.Error(),
			})
			fmt.Fprintf(os.Stderr, "Warning: Failed to connect to Discord: %v\n", err)
		} else {
			logger.Info("Discord connected successfully", nil)
		}
	}

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))

	// Create and run the app
	mainMenu := ui.NewMainMenuWithClient(cfg, client)
	initialState := StateMainMenu
	var initialModel tea.Model = mainMenu
	
	// If we need auth and not using NoAniList, show auth screen first
	if needsAuth && !cfg.AniList.NoAniList {
		logger.Info("Starting with AniList auth screen", nil)
		initialState = StateAniListAuth
		initialModel = ui.NewAniListAuth(cfg)
	} else {
		logger.Info("Starting with main menu", nil)
	}
	
	app := &App{
		cfg:          cfg,
		client:       client,
		discordMgr:   discordMgr,
		state:        initialState,
		currentModel: initialModel,
		mainMenu:     mainMenu,
		spinner:      s,
	}

	logger.Info("Starting TUI application", nil)

	// Use alternate screen buffer for fullscreen app experience
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		logger.Fatal("TUI application error", err, nil)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	logger.Info("TUI application closed", nil)

	// Cleanup
	if cfg.Discord.DiscordPresence {
		logger.Debug("Clearing Discord presence", nil)
		discordMgr.Clear()
	}

	logger.Info("Application shutdown complete", nil)
}

func (a *App) Init() tea.Cmd {
	// Get initial window size
	return tea.Batch(
		a.currentModel.Init(),
		tea.WindowSize(),
		a.spinner.Tick,
	)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		return a, cmd

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

		// Handle navigation from error state
		if a.err != nil {
			switch msg.String() {
			case "q":
				return a, tea.Quit
			case "enter":
				// Go to Watch Anime menu
				a.err = nil
				a.state = StateAnimeList
				a.currentModel = ui.NewAnimeList(a.cfg, a.client)
				return a, a.currentModel.Init()
			case "esc", "backspace", "m":
				// Go back to main menu
				a.err = nil
				a.state = StateMainMenu
				a.currentModel = a.mainMenu
				return a, a.currentModel.Init()
			}
			return a, nil
		}

	case ui.MenuSelectionMsg:
		return a.handleMenuSelection(msg.Selection, msg.ShowEpisodeSelect)

	case ui.AnimeSelectedMsg:
		a.selectedAnime = &msg.Anime
		a.selectedEntry = msg.Entry
		return a.handleAnimeSelected(msg.ShowEpisodeSelect)

	case ui.EpisodeReadyMsg:
		a.selectedEp = msg.Episode
		a.subOrDub = msg.SubOrDub
		a.loadingMsg = "Fetching episode info..."
		return a, a.fetchAndPlayEpisode()

	case ui.BackMsg:
		return a.handleBack()

	case ContinueWatchingResultMsg:
		a.loadingMsg = "" // Clear loading
		if msg.Err != nil {
			a.err = msg.Err
			a.state = StateMainMenu
			a.currentModel = a.mainMenu
			return a, a.currentModel.Init() // Re-initialize to refresh continue watching anime
		}
		if msg.Entry != nil {
			return a.continueFromEntry(*msg.Entry, msg.ShowEpisodeSelect)
		}

	case PlayEpisodeResultMsg:
		if msg.Err != nil {
			a.err = msg.Err
			a.loadingMsg = ""
			return a, nil
		}
		// Video links fetched, now preparing to play
		a.loadingMsg = "Preparing video player..."
		return a.handlePlayEpisode(msg.VideoData)
	
	case ui.AutoplayPromptMsg:
		// User chose to enable/disable autoplay
		a.autoplayMode = msg.EnableAutoplay
		if a.autoplayMode {
			// Continue to next episode
			return a.playNextEpisode()
		} else {
			// Return to main menu
			a.state = StateMainMenu
			a.currentModel = a.mainMenu
			return a, a.currentModel.Init() // Re-initialize to refresh continue watching anime
		}
	
	case ui.AniListAuthSuccessMsg:
		// Authentication successful, store client and go to main menu
		a.client = msg.Client
		a.mainMenu.SetClient(msg.Client)
		a.state = StateMainMenu
		a.currentModel = a.mainMenu
		return a, a.currentModel.Init() // Re-initialize to fetch continue watching anime

	case tea.WindowSizeMsg:
		// Store window size and pass to current model
		a.width = msg.Width
		a.height = msg.Height
	}

	// Delegate to current model
	var cmd tea.Cmd
	a.currentModel, cmd = a.currentModel.Update(msg)
	return a, cmd
}

func (a *App) View() string {
	if a.err != nil {
		styles := ui.DefaultStyles()
		s := ui.GetBannerGradient() + "\n"
		s += styles.Subtitle.Render("Oni — Anime Streaming Client") + "\n\n"
		
		s += styles.Error.Render("⚠ Error") + "\n\n"
		s += styles.Info.Render(a.err.Error()) + "\n\n"
		
		// Add log file path reference
		logPath := logger.GetLogFilePath()
		if logPath != "" {
			s += styles.Help.Render(fmt.Sprintf("Details saved to: %s", logPath)) + "\n\n"
		}
		
		s += styles.Prompt.Render("Options:") + "\n"
		s += styles.MenuItem.Render("  Enter") + " " + styles.Help.Render("→ Go to Watch Anime menu") + "\n"
		s += styles.MenuItem.Render("  Esc/Backspace/m") + " " + styles.Help.Render("→ Go back to main menu") + "\n"
		s += styles.MenuItem.Render("  q") + " " + styles.Help.Render("→ Quit") + "\n"
		
		return s
	}

	view := a.currentModel.View()

	// If loading, replace the last line (footer/help) with loading message
	if a.loadingMsg != "" {
		lines := strings.Split(view, "\n")
		if len(lines) > 0 {
			// Remove the last line (help text)
			lines = lines[:len(lines)-1]
			view = strings.Join(lines, "\n")
		}
		// Add loading message
		view += "\n" + a.spinner.View() + " " + a.loadingMsg
	}

	return view
}

func (a *App) handleMenuSelection(selection string, showEpisodeSelect bool) (tea.Model, tea.Cmd) {
	logger.Debug("Menu selection", map[string]interface{}{
		"selection":        selection,
		"showEpisodeSelect": showEpisodeSelect,
	})

	switch selection {
	case "Continue Watching":
		logger.Info("User selected Continue Watching", nil)
		a.loadingMsg = "Finding your next episode..."
		return a, a.fetchContinueWatching(showEpisodeSelect)

	case "Watch Anime":
		logger.Info("User selected Watch Anime", nil)
		a.state = StateAnimeList
		a.currentModel = ui.NewAnimeList(a.cfg, a.client)
		return a, a.currentModel.Init()

	case "Update Progress/Status/Score":
		logger.Info("User selected Update Progress/Status/Score", nil)
		a.state = StateUpdateProgress
		a.currentModel = ui.NewUpdateProgress(a.cfg, a.client)
		return a, a.currentModel.Init()

	case "Settings":
		logger.Info("User selected Settings", nil)
		a.state = StateEditConfig
		a.currentModel = ui.NewConfigEditor(a.cfg)
		return a, a.currentModel.Init()

	case "Quit":
		logger.Info("User selected Quit", nil)
		return a, tea.Quit
	}

	return a, nil
}

// ContinueWatchingResultMsg is sent when continue watching fetch is complete
type ContinueWatchingResultMsg struct {
	Entry            *anilist.MediaListEntry
	ShowEpisodeSelect bool
	Err              error
}

// fetchContinueWatching fetches the anime to continue watching from local history
func (a *App) fetchContinueWatching(showEpisodeSelect bool) tea.Cmd {
	return func() tea.Msg {
		// Get current incognito mode state from main menu
		a.incognitoMode = a.mainMenu.GetIncognitoMode()
		
		logger.Debug("Fetching continue watching", map[string]interface{}{
			"incognitoMode": a.incognitoMode,
		})

		// Use incognito or normal history based on current mode
		history, err := player.LoadHistoryWithIncognito(a.incognitoMode)
		if err != nil || len(history) == 0 {
			logger.Warn("No anime found to continue watching", map[string]interface{}{
				"error":         err,
				"historyLength": len(history),
			})
			return ContinueWatchingResultMsg{
				Err: fmt.Errorf("no anime found to continue watching"),
			}
		}

		lastEntry := history[len(history)-1]
		logger.Debug("Found last watched anime", map[string]interface{}{
			"mediaID":  lastEntry.MediaID,
			"title":    lastEntry.Title,
			"progress": lastEntry.Progress,
		})
		
		// If AniList is available, fetch full anime info
		if !a.cfg.AniList.NoAniList && a.client != nil {
			animeInfo, err := a.client.GetAnimeInfo(context.Background(), lastEntry.MediaID)
			if err == nil {
				logger.Info("Fetched anime info from AniList", map[string]interface{}{
					"mediaID": lastEntry.MediaID,
				})
				entry := anilist.MediaListEntry{
					Media:    *animeInfo,
					Progress: lastEntry.Progress,
				}
				return ContinueWatchingResultMsg{
					Entry:            &entry,
					ShowEpisodeSelect: showEpisodeSelect,
				}
			}
			logger.Warn("Failed to fetch anime info from AniList", map[string]interface{}{
				"error":   err.Error(),
				"mediaID": lastEntry.MediaID,
			})
		}

		// If AniList not available or fetch failed, create a minimal entry from history
		// This will require searching by title when playing
		logger.Debug("Using minimal entry from history", nil)
		entry := anilist.MediaListEntry{
			Media: anilist.Anime{
				ID:    lastEntry.MediaID,
				Title: anilist.Title{English: lastEntry.Title},
			},
			Progress: lastEntry.Progress,
		}
		return ContinueWatchingResultMsg{
			Entry:            &entry,
			ShowEpisodeSelect: showEpisodeSelect,
		}
	}
}

func (a *App) handleAnimeSelected(showEpisodeSelect bool) (tea.Model, tea.Cmd) {
	// Determine progress
	progress := 0

	// If this is from continue watching, get progress from list entry
	if a.selectedEntry != nil {
		progress = a.selectedEntry.Progress
	}

	// If showEpisodeSelect is false and we have progress, try to auto-play next episode
	// If auto-play fails, fall back to episode selection
	if !showEpisodeSelect && progress > 0 {
		// Calculate next episode
		nextEp := progress + 1
		if a.selectedAnime.Episodes != nil && nextEp > *a.selectedAnime.Episodes {
			nextEp = progress
		}
		if nextEp < 1 {
			nextEp = 1
		}
		a.selectedEp = nextEp
		a.subOrDub = a.cfg.Playback.SubOrDub
		if a.subOrDub == "" {
			a.subOrDub = "sub" // Default to sub
		}
		
		// Try to auto-play the next episode
		a.loadingMsg = "Fetching episode info..."
		return a, a.fetchAndPlayEpisode()
	}

	// Show episode selection (either requested or no progress available)
	a.state = StateEpisodeSelect
	a.currentModel = ui.NewEpisodeSelect(a.cfg, *a.selectedAnime, progress)
	return a, a.currentModel.Init()
}

// PlayEpisodeResultMsg is sent when episode is ready to play
type PlayEpisodeResultMsg struct {
	VideoData *providers.VideoData
	Err       error
}

// fetchAndPlayEpisode fetches episode info and video links, then plays
func (a *App) fetchAndPlayEpisode() tea.Cmd {
	return func() tea.Msg {
		if a.selectedAnime == nil {
			logger.Error("No anime selected for playback", nil, nil)
			return PlayEpisodeResultMsg{Err: fmt.Errorf("no anime selected")}
		}

		logger.Info("Fetching episode", map[string]interface{}{
			"mediaID":  a.selectedAnime.ID,
			"title":    a.selectedAnime.Title.UserPreferred,
			"episode":  a.selectedEp,
			"provider": a.cfg.Provider.Provider,
			"quality":  a.cfg.Provider.Quality,
			"subOrDub": a.subOrDub,
		})

		// Get provider
		prov, err := providers.GetProvider(a.cfg.Provider.Provider)
		if err != nil {
			logger.Error("Failed to get provider", err, map[string]interface{}{
				"provider": a.cfg.Provider.Provider,
			})
			return PlayEpisodeResultMsg{Err: err}
		}

		// Get episode info
		epInfo, err := prov.GetEpisodeInfo(context.Background(), a.selectedAnime.ID, a.selectedEp, a.selectedAnime.Title.UserPreferred)
		if err != nil {
			logger.Error("Failed to get episode info", err, map[string]interface{}{
				"mediaID":  a.selectedAnime.ID,
				"episode":  a.selectedEp,
				"provider": a.cfg.Provider.Provider,
			})
			return PlayEpisodeResultMsg{Err: fmt.Errorf("failed to get episode info: %w", err)}
		}

		logger.Debug("Episode info fetched", map[string]interface{}{
			"episodeID": epInfo.EpisodeID,
		})

		// Get video link
		videoData, err := prov.GetVideoLink(context.Background(), epInfo, a.cfg.Provider.Quality, a.subOrDub)
		if err != nil {
			logger.Error("Failed to get video link", err, map[string]interface{}{
				"episodeID": epInfo.EpisodeID,
				"quality":   a.cfg.Provider.Quality,
				"subOrDub":  a.subOrDub,
			})
			return PlayEpisodeResultMsg{Err: fmt.Errorf("failed to get video link: %w", err)}
		}

		logger.Info("Video link fetched successfully", map[string]interface{}{
			"hasSubtitles": len(videoData.SubtitleURLs) > 0,
		})

		return PlayEpisodeResultMsg{VideoData: videoData}
	}
}

func (a *App) handlePlayEpisode(videoData *providers.VideoData) (tea.Model, tea.Cmd) {
	if a.selectedAnime == nil {
		logger.Error("No anime selected in handlePlayEpisode", nil, nil)
		a.err = fmt.Errorf("no anime selected")
		a.loadingMsg = ""
		return a, nil
	}

	logger.Info("Starting playback", map[string]interface{}{
		"title":   a.selectedAnime.Title.UserPreferred,
		"episode": a.selectedEp,
	})

	// Set Discord presence (only if not in incognito mode)
	a.incognitoMode = a.mainMenu.GetIncognitoMode()
	if a.cfg.Discord.DiscordPresence && a.discordMgr.IsEnabled() && !a.incognitoMode {
		year := 0
		if a.selectedAnime.StartDate.Year != nil {
			year = *a.selectedAnime.StartDate.Year
		}
		logger.Debug("Setting Discord presence", map[string]interface{}{
			"title":   a.selectedAnime.Title.UserPreferred,
			"episode": a.selectedEp,
		})
		a.discordMgr.SetPresence(
			a.selectedAnime.Title.UserPreferred,
			a.selectedEp,
			year,
			a.selectedAnime.CoverImage.Large,
		)
	}

	// Get player
	plyr, err := player.GetPlayer(a.cfg)
	if err != nil {
		logger.Error("Failed to get player", err, map[string]interface{}{
			"player": a.cfg.Player.Player,
		})
		a.err = err
		a.loadingMsg = ""
		return a, nil
	}

	// Check for resume point (only if episode was completed before)
	resumeFrom := "00:00:00"
	historyEntry, _ := player.GetHistoryEntryWithIncognito(a.selectedAnime.ID, a.selectedEp, a.incognitoMode)
	if historyEntry != nil {
		resumeFrom = historyEntry.Timestamp
		logger.Debug("Resume point found", map[string]interface{}{
			"timestamp": resumeFrom,
		})
	}

	// Play video
	a.loadingMsg = "Starting video player..."
	title := fmt.Sprintf("%s - Episode %d", a.selectedAnime.Title.UserPreferred, a.selectedEp)
	playbackInfo, err := plyr.Play(context.Background(), videoData, title, resumeFrom)
	a.loadingMsg = "" // Clear loading after play starts
	if err != nil {
		logger.Error("Failed to play video", err, map[string]interface{}{
			"title":   title,
			"player":  a.cfg.Player.Player,
		})
		a.err = fmt.Errorf("failed to play video: %w", err)
		return a, nil
	}

	logger.Info("Playback completed", map[string]interface{}{
		"completedSuccessful": playbackInfo.CompletedSuccessful,
		"stoppedAt":           playbackInfo.StoppedAt,
		"percentProgress":     playbackInfo.PercentageProgress,
	})

	// Only save to local history if episode was completed successfully
	if playbackInfo.CompletedSuccessful {
		episodesTotal := 9999
		if a.selectedAnime.Episodes != nil {
			episodesTotal = *a.selectedAnime.Episodes
		}

		entry := player.HistoryEntry{
			MediaID:       a.selectedAnime.ID,
			Progress:      a.selectedEp,
			EpisodesTotal: episodesTotal,
			Timestamp:     playbackInfo.StoppedAt,
			Title:         a.selectedAnime.Title.UserPreferred,
		}

		// Save to incognito or normal history based on current mode
		if err := player.SaveHistoryEntryWithIncognito(entry, a.incognitoMode); err != nil {
			logger.Error("Failed to save history", err, map[string]interface{}{
				"mediaID":        a.selectedAnime.ID,
				"episode":        a.selectedEp,
				"incognitoMode": a.incognitoMode,
			})
			fmt.Fprintf(os.Stderr, "Warning: Failed to save history: %v\n", err)
		} else {
			logger.Info("History saved", map[string]interface{}{
				"mediaID":        a.selectedAnime.ID,
				"episode":        a.selectedEp,
				"incognitoMode": a.incognitoMode,
			})
		}
	}

	// Update AniList progress separately (if enabled, episode completed, and NOT in incognito mode)
	if playbackInfo.CompletedSuccessful && !a.cfg.AniList.NoAniList && !a.incognitoMode && a.client != nil {
		status := "CURRENT"
		if a.selectedAnime.Episodes != nil && a.selectedEp >= *a.selectedAnime.Episodes {
			status = "COMPLETED"
		}

		logger.Debug("Updating AniList progress", map[string]interface{}{
			"mediaID": a.selectedAnime.ID,
			"episode": a.selectedEp,
			"status":  status,
		})

		err := a.client.UpdateProgress(context.Background(), a.selectedAnime.ID, a.selectedEp, status)
		if err != nil {
			logger.Error("Failed to update AniList progress", err, map[string]interface{}{
				"mediaID": a.selectedAnime.ID,
				"episode": a.selectedEp,
			})
			fmt.Fprintf(os.Stderr, "Warning: Failed to update AniList progress: %v\n", err)
		} else {
			logger.Info("AniList progress updated", map[string]interface{}{
				"mediaID": a.selectedAnime.ID,
				"episode": a.selectedEp,
				"status":  status,
			})
		}
		// Note: We don't delete from local history even if AniList marks it as completed
		// Local history is independent and preserved at all times
	}

	// Check if episode was completed successfully
	if playbackInfo.CompletedSuccessful {
		// Check if there are more episodes
		hasMoreEpisodes := true
		if a.selectedAnime.Episodes != nil && a.selectedEp >= *a.selectedAnime.Episodes {
			hasMoreEpisodes = false
		}

		if hasMoreEpisodes {
			// Determine if we should prompt for autoplay or continue automatically
			shouldPrompt := a.shouldPromptForAutoplay()
			
			if shouldPrompt {
				// Show autoplay prompt
				a.state = StateMainMenu
				a.currentModel = ui.NewAutoplayPrompt(a.cfg, a.selectedAnime.Title.UserPreferred, a.selectedEp+1)
				return a, a.currentModel.Init()
			} else if a.autoplayMode {
				// Continue to next episode automatically
				return a.playNextEpisode()
			}
		}
	}

	// Clear Discord presence
	if a.cfg.Discord.DiscordPresence {
		a.discordMgr.Clear()
	}

	// Reset autoplay mode when returning to main menu
	a.autoplayMode = false

	// Return to main menu
	a.state = StateMainMenu
	a.currentModel = a.mainMenu
	return a, a.currentModel.Init() // Re-initialize to refresh continue watching anime
}

func (a *App) continueFromEntry(entry anilist.MediaListEntry, showEpisodeSelect bool) (tea.Model, tea.Cmd) {
	a.selectedAnime = &entry.Media
	a.selectedEntry = &entry

	nextEp := entry.Progress + 1
	if entry.Media.Episodes != nil && nextEp > *entry.Media.Episodes {
		nextEp = entry.Progress
	}
	if nextEp < 1 {
		nextEp = 1
	}

	a.selectedEp = nextEp
	a.subOrDub = a.cfg.Playback.SubOrDub
	if a.subOrDub == "" {
		a.subOrDub = "sub"
	}

	if showEpisodeSelect {
		a.state = StateEpisodeSelect
		a.currentModel = ui.NewEpisodeSelect(a.cfg, entry.Media, entry.Progress)
		return a, a.currentModel.Init()
	}

	a.loadingMsg = "Fetching episode info..."
	return a, a.fetchAndPlayEpisode()
}

// shouldPromptForAutoplay determines if we should ask the user about autoplay
func (a *App) shouldPromptForAutoplay() bool {
	// If already in autoplay mode, don't prompt again
	if a.autoplayMode {
		return false
	}

	// If this is a different anime from the last one, prompt
	if a.lastAnimeID != 0 && a.lastAnimeID != a.selectedAnime.ID {
		return true
	}

	// If last watch was more than 1 hour ago, prompt
	if !a.lastWatchTime.IsZero() {
		timeSinceLastWatch := time.Since(a.lastWatchTime)
		if timeSinceLastWatch > time.Hour {
			return true
		}
	}

	// First episode of a session, prompt
	if a.lastAnimeID == 0 {
		return true
	}

	return false
}

// playNextEpisode prepares and plays the next episode
func (a *App) playNextEpisode() (tea.Model, tea.Cmd) {
	// Update tracking
	a.lastAnimeID = a.selectedAnime.ID
	a.lastWatchTime = time.Now()

	// Increment episode
	a.selectedEp++

	// Check if we've reached the end
	if a.selectedAnime.Episodes != nil && a.selectedEp > *a.selectedAnime.Episodes {
		// No more episodes
		a.autoplayMode = false
		a.state = StateMainMenu
		a.currentModel = a.mainMenu
		return a, a.currentModel.Init() // Re-initialize to refresh continue watching anime
	}

	// Fetch and play next episode
	a.loadingMsg = fmt.Sprintf("Loading episode %d...", a.selectedEp)
	return a, a.fetchAndPlayEpisode()
}

func (a *App) handleBack() (tea.Model, tea.Cmd) {
	a.state = StateMainMenu
	a.currentModel = a.mainMenu
	a.selectedAnime = nil
	a.selectedEntry = nil
	a.err = nil
	return a, a.currentModel.Init() // Re-initialize to refresh continue watching anime
}

func showUsage() {
	fmt.Printf(`ONI - Anime Streaming Client

Usage: oni [options] [query]

Options:
  -e             Edit configuration
  -d             Enable Discord presence
  -h             Show this help
  -q <quality>   Video quality (e.g., 1080, 720)
  -v             Show version
  -w <provider>  Provider (allanime, aniwatch, yugen, hdrezka, aniworld, crunchyroll)
  --sub-or-dub   Audio type (sub, dub)

Examples:
  oni                         # Start interactive menu
  oni -q 720                  # Set quality to 720p
  oni -w aniwatch             # Use aniwatch provider

`)
}

