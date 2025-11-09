package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pranshuj73/oni/anilist"
	"github.com/pranshuj73/oni/config"
	"github.com/pranshuj73/oni/discord"
	"github.com/pranshuj73/oni/player"
	"github.com/pranshuj73/oni/providers"
	"github.com/pranshuj73/oni/ui"
)

const version = "0.1.1"

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

	if *showVersion {
		fmt.Printf("oni version %s\n", version)
		os.Exit(0)
	}

	if *showHelp {
		showUsage()
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Apply command-line overrides
	if *quality != "" {
		cfg.Provider.Quality = *quality
	}
	if *provider != "" {
		cfg.Provider.Provider = *provider
	}
	if *subOrDub != "" {
		cfg.Playback.SubOrDub = *subOrDub
	}
	if *discordPresence {
		cfg.Discord.DiscordPresence = true
	}

	// Handle config edit mode
	if *editConfig {
		p := tea.NewProgram(ui.NewConfigEditor(cfg))
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Try to load existing AniList token
	var client *anilist.Client
	var needsAuth bool
	if !cfg.AniList.NoAniList {
		token, err := anilist.LoadToken()
		if err == nil && token != "" {
			// Token exists, try to create client
			client, err = anilist.NewClient()
			if err != nil {
				// Token might be invalid, need re-auth
				needsAuth = true
			}
		} else {
			// No token, need auth
			needsAuth = true
		}
	}

	// Create Discord presence manager
	discordMgr := discord.NewPresenceManager(cfg.Discord.DiscordPresence)
	if cfg.Discord.DiscordPresence {
		if err := discordMgr.Connect(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to connect to Discord: %v\n", err)
		}
	}

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))

	// Start refreshing cache in background if client is available
	if client != nil && !cfg.AniList.NoAniList {
		ui.RefreshCacheInBackground(cfg, client)
	}

	// Create and run the app
	mainMenu := ui.NewMainMenuWithClient(cfg, client)
	initialState := StateMainMenu
	var initialModel tea.Model = mainMenu
	
	// If we need auth and not using NoAniList, show auth screen first
	if needsAuth && !cfg.AniList.NoAniList {
		initialState = StateAniListAuth
		initialModel = ui.NewAniListAuth(cfg)
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

	// Use alternate screen buffer for fullscreen app experience
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Cleanup
	if cfg.Discord.DiscordPresence {
		discordMgr.Clear()
	}
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
		a.loadingMsg = "Fetching Episode Info"
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
			return a.continueFromEntry(*msg.Entry, msg.Episode, msg.ShowEpisodeSelect)
		}

	case PlayEpisodeResultMsg:
		if msg.Err != nil {
			a.err = msg.Err
			a.loadingMsg = ""
			return a, nil
		}
		// Video links fetched, now loading episode
		a.loadingMsg = "Loading Episode"
		// Trigger play in next update cycle so UI can render "Loading Episode"
		return a, func() tea.Msg {
			return PlayVideoMsg{VideoData: msg.VideoData}
		}

	case PlayVideoMsg:
		// Now actually play the video (UI has rendered "Loading Episode")
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
		// Add loading message in green
		styles := ui.DefaultStyles()
		view += "\n" + a.spinner.View() + " " + styles.Success.Render(a.loadingMsg)
	}

	return view
}

func (a *App) handleMenuSelection(selection string, showEpisodeSelect bool) (tea.Model, tea.Cmd) {
	switch selection {
	case "Continue Watching":
		a.loadingMsg = "Finding your next episode..."
		return a, a.fetchContinueWatching(showEpisodeSelect)

	case "Watch Anime":
		a.state = StateAnimeList
		a.currentModel = ui.NewAnimeList(a.cfg, a.client)
		return a, a.currentModel.Init()

	case "Update Progress/Status/Score":
		a.state = StateUpdateProgress
		a.currentModel = ui.NewUpdateProgress(a.cfg, a.client)
		return a, a.currentModel.Init()

	case "Settings":
		a.state = StateEditConfig
		a.currentModel = ui.NewConfigEditor(a.cfg)
		return a, a.currentModel.Init()

	case "Quit":
		return a, tea.Quit
	}

	return a, nil
}

// ContinueWatchingResultMsg is sent when continue watching fetch is complete
type ContinueWatchingResultMsg struct {
	Entry            *anilist.MediaListEntry
	Episode          int // The episode number to play (calculated based on 95% completion)
	ShowEpisodeSelect bool
	Err              error
}

// fetchContinueWatching fetches the anime to continue watching from local history
func (a *App) fetchContinueWatching(showEpisodeSelect bool) tea.Cmd {
	return func() tea.Msg {
		// Get current incognito mode state from main menu
		a.incognitoMode = a.mainMenu.GetIncognitoMode()
		
		// Use incognito or normal history based on current mode
		history, err := player.LoadHistoryWithIncognito(a.incognitoMode)
		if err != nil || len(history) == 0 {
			return ContinueWatchingResultMsg{
				Err: fmt.Errorf("no anime found to continue watching"),
			}
		}

		// Find the entry with the most recent LastWatched timestamp (same logic as main_menu.go)
		var lastEntry *player.HistoryEntry
		var latestTime time.Time
		
		for i := range history {
			entry := &history[i]
			if entry.Title == "" {
				continue
			}
			
			// Parse LastWatched timestamp (RFC3339 format)
			watchedTime, err := time.Parse(time.RFC3339, entry.LastWatched)
			if err != nil {
				// If LastWatched is missing or invalid (old format), skip this entry
				continue
			}
			
			// Check if this is the most recent
			if lastEntry == nil || watchedTime.After(latestTime) {
				lastEntry = entry
				latestTime = watchedTime
			}
		}
		
		if lastEntry == nil {
			return ContinueWatchingResultMsg{
				Err: fmt.Errorf("no anime found to continue watching"),
			}
		}
		
		// Calculate which episode to play based on 95% completion check (same logic as main_menu.go)
		var episodeToPlay int
		var isComplete bool
		
		if lastEntry.Duration != "" && lastEntry.Timestamp != "" && lastEntry.Timestamp != "00:00:00" {
			// Parse timestamp (current position)
			timestampParts := strings.Split(lastEntry.Timestamp, ":")
			if len(timestampParts) == 3 {
				hours, _ := strconv.Atoi(timestampParts[0])
				minutes, _ := strconv.Atoi(timestampParts[1])
				seconds, _ := strconv.Atoi(timestampParts[2])
				currentSeconds := hours*3600 + minutes*60 + seconds
				
				// Parse duration (total length)
				durationParts := strings.Split(lastEntry.Duration, ":")
				if len(durationParts) == 3 {
					durHours, _ := strconv.Atoi(durationParts[0])
					durMinutes, _ := strconv.Atoi(durationParts[1])
					durSeconds, _ := strconv.Atoi(durationParts[2])
					totalSeconds := durHours*3600 + durMinutes*60 + durSeconds
					
					if totalSeconds > 0 {
						percentage := (float64(currentSeconds) / float64(totalSeconds)) * 100
						isComplete = percentage >= 95.0
					}
				}
			}
		}
		
		if isComplete {
			// Play next episode (progress + 1) if previous was 95%+ complete
			episodeToPlay = lastEntry.Progress + 1
			// Don't exceed total episodes
			if lastEntry.EpisodesTotal > 0 && episodeToPlay > lastEntry.EpisodesTotal {
				episodeToPlay = lastEntry.EpisodesTotal
			}
		} else {
			// Play same episode if not 95% complete
			episodeToPlay = lastEntry.Progress
		}
		
		// If AniList is available, fetch full anime info
		if !a.cfg.AniList.NoAniList && a.client != nil {
			animeInfo, err := a.client.GetAnimeInfo(context.Background(), lastEntry.MediaID)
			if err == nil {
				entry := anilist.MediaListEntry{
					Media:    *animeInfo,
					Progress: lastEntry.Progress,
				}
				return ContinueWatchingResultMsg{
					Entry:            &entry,
					Episode:          episodeToPlay,
					ShowEpisodeSelect: showEpisodeSelect,
				}
			}
		}

		// If AniList not available or fetch failed, create a minimal entry from history
		// This will require searching by title when playing
		entry := anilist.MediaListEntry{
			Media: anilist.Anime{
				ID:    lastEntry.MediaID,
				Title: anilist.Title{English: lastEntry.Title},
			},
			Progress: lastEntry.Progress,
		}
		return ContinueWatchingResultMsg{
			Entry:            &entry,
			Episode:          episodeToPlay,
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
		a.loadingMsg = "Fetching Episode Info"
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

// PlayVideoMsg is sent to trigger actual video playback (after UI renders "Loading Episode")
type PlayVideoMsg struct {
	VideoData *providers.VideoData
}

// fetchAndPlayEpisode fetches episode info and video links, then plays
func (a *App) fetchAndPlayEpisode() tea.Cmd {
	return func() tea.Msg {
		if a.selectedAnime == nil {
			return PlayEpisodeResultMsg{Err: fmt.Errorf("no anime selected")}
		}

		// Get provider
		prov, err := providers.GetProvider(a.cfg.Provider.Provider)
		if err != nil {
			return PlayEpisodeResultMsg{Err: err}
		}

		// Get episode info
		epInfo, err := prov.GetEpisodeInfo(context.Background(), a.selectedAnime.ID, a.selectedEp, a.selectedAnime.Title.UserPreferred)
		if err != nil {
			return PlayEpisodeResultMsg{Err: fmt.Errorf("failed to get episode info: %w", err)}
		}

		// Get video link
		videoData, err := prov.GetVideoLink(context.Background(), epInfo, a.cfg.Provider.Quality, a.subOrDub)
		if err != nil {
			return PlayEpisodeResultMsg{Err: fmt.Errorf("failed to get video link: %w", err)}
		}

		return PlayEpisodeResultMsg{VideoData: videoData}
	}
}

func (a *App) handlePlayEpisode(videoData *providers.VideoData) (tea.Model, tea.Cmd) {
	if a.selectedAnime == nil {
		a.err = fmt.Errorf("no anime selected")
		a.loadingMsg = ""
		return a, nil
	}

	// Set Discord presence (only if not in incognito mode)
	a.incognitoMode = a.mainMenu.GetIncognitoMode()
	if a.cfg.Discord.DiscordPresence && a.discordMgr.IsEnabled() && !a.incognitoMode {
		year := 0
		if a.selectedAnime.StartDate.Year != nil {
			year = *a.selectedAnime.StartDate.Year
		}
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
		a.err = err
		a.loadingMsg = ""
		return a, nil
	}

	// Check for resume point (only if episode was not already completed)
	resumeFrom := "00:00:00"
	historyEntry, _ := player.GetHistoryEntryWithIncognito(a.selectedAnime.ID, a.selectedEp, a.incognitoMode)
	if historyEntry != nil && historyEntry.Timestamp != "" && historyEntry.Timestamp != "00:00:00" {
		// We need the actual duration to calculate time remaining
		// If duration is not available, start from beginning
		if historyEntry.Duration == "" {
			resumeFrom = "00:00:00"
		} else {
			// Parse timestamp to check if it's near the end
			// Format: HH:MM:SS
			parts := strings.Split(historyEntry.Timestamp, ":")
			if len(parts) == 3 {
				hours, _ := strconv.Atoi(parts[0])
				minutes, _ := strconv.Atoi(parts[1])
				seconds, _ := strconv.Atoi(parts[2])
				currentSeconds := hours*3600 + minutes*60 + seconds
				
				// Parse duration from history entry (HH:MM:SS format)
				durationParts := strings.Split(historyEntry.Duration, ":")
				if len(durationParts) == 3 {
					durHours, _ := strconv.Atoi(durationParts[0])
					durMinutes, _ := strconv.Atoi(durationParts[1])
					durSeconds, _ := strconv.Atoi(durationParts[2])
					totalDurationSeconds := durHours*3600 + durMinutes*60 + durSeconds
					
					timeRemaining := totalDurationSeconds - currentSeconds
					
					// If less than 1 minute remaining, start from beginning to avoid immediate completion
					if timeRemaining < 60 && currentSeconds > 0 {
						resumeFrom = "00:00:00"
					} else if currentSeconds > 30 {
						// Resume from the saved timestamp if it's reasonable
						resumeFrom = historyEntry.Timestamp
					}
					// If currentSeconds <= 30, start from beginning (too early to resume)
				} else {
					// Invalid duration format, start from beginning
					resumeFrom = "00:00:00"
				}
			} else {
				// Invalid timestamp format, start from beginning
				resumeFrom = "00:00:00"
			}
		}
	}

	// Play video
	a.loadingMsg = "Playing Episode"
	title := fmt.Sprintf("%s - Episode %d", a.selectedAnime.Title.UserPreferred, a.selectedEp)
	playbackInfo, err := plyr.Play(context.Background(), videoData, title, resumeFrom)
	a.loadingMsg = "" // Clear loading after play starts
	if err != nil {
		a.err = fmt.Errorf("failed to play video: %w", err)
		return a, nil
	}

	// Save history entry when episode starts
	episodesTotal := 9999
	if a.selectedAnime.Episodes != nil {
		episodesTotal = *a.selectedAnime.Episodes
	}

	// Set LastWatched to current time so "Continue Watching" immediately points to this episode
	startLastWatched := time.Now().Format(time.RFC3339)

	// Use duration from previous history entry if available, otherwise empty (will be set on completion)
	startDuration := ""
	if historyEntry != nil && historyEntry.Duration != "" {
		startDuration = historyEntry.Duration
	}

	startEntry := player.HistoryEntry{
		MediaID:       a.selectedAnime.ID,
		Progress:      a.selectedEp,
		EpisodesTotal: episodesTotal,
		Timestamp:     resumeFrom,
		Duration:      startDuration,
		LastWatched:   startLastWatched,
		Title:         a.selectedAnime.Title.UserPreferred,
	}

	// Save to incognito or normal history based on current mode
	if err := player.SaveHistoryEntryWithIncognito(startEntry, a.incognitoMode); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save history on start: %v\n", err)
	}

	// Update history entry with the actual playback position and duration
	// This ensures we can resume from where we stopped, even if not completed
	if playbackInfo.StoppedAt != "" && playbackInfo.StoppedAt != "00:00:00" {
		// Get current timestamp for LastWatched
		lastWatched := time.Now().Format(time.RFC3339)

		// Use the duration from playback info if available, otherwise keep the one from start entry
		duration := playbackInfo.TotalDuration
		if duration == "" && startEntry.Duration != "" {
			duration = startEntry.Duration
		}

		updatedEntry := player.HistoryEntry{
			MediaID:       a.selectedAnime.ID,
			Progress:      a.selectedEp,
			EpisodesTotal: episodesTotal,
			Timestamp:     playbackInfo.StoppedAt,
			Duration:      duration,
			LastWatched:   lastWatched,
			Title:         a.selectedAnime.Title.UserPreferred,
		}

		// Update history entry with actual playback position
		if err := player.SaveHistoryEntryWithIncognito(updatedEntry, a.incognitoMode); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to save history after playback: %v\n", err)
		}
	}

	// Update AniList progress separately (if enabled, episode completed, and NOT in incognito mode)
	if playbackInfo.CompletedSuccessful && !a.cfg.AniList.NoAniList && !a.incognitoMode && a.client != nil {
		status := "CURRENT"
		if a.selectedAnime.Episodes != nil && a.selectedEp >= *a.selectedAnime.Episodes {
			status = "COMPLETED"
		}

		err := a.client.UpdateProgress(context.Background(), a.selectedAnime.ID, a.selectedEp, status)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to update AniList progress: %v\n", err)
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

func (a *App) continueFromEntry(entry anilist.MediaListEntry, episode int, showEpisodeSelect bool) (tea.Model, tea.Cmd) {
	a.selectedAnime = &entry.Media
	a.selectedEntry = &entry

	// Use the episode number calculated in fetchContinueWatching (based on 95% completion)
	if episode < 1 {
		episode = 1
	}
	if entry.Media.Episodes != nil && episode > *entry.Media.Episodes {
		episode = *entry.Media.Episodes
	}

	a.selectedEp = episode
	a.subOrDub = a.cfg.Playback.SubOrDub
	if a.subOrDub == "" {
		a.subOrDub = "sub"
	}

	if showEpisodeSelect {
		a.state = StateEpisodeSelect
		// Use the calculated episode (based on 95% completion) as the initial progress
		// This ensures the episode selection matches what the menu displayed
		a.currentModel = ui.NewEpisodeSelect(a.cfg, entry.Media, episode-1)
		return a, a.currentModel.Init()
	}

	a.loadingMsg = "Fetching Episode Info"
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
  -w <provider>  Provider (allanime, aniwatch, yugen, hdrezka, aniworld)
  --sub-or-dub   Audio type (sub, dub)

Examples:
  oni                         # Start interactive menu
  oni -q 720                  # Set quality to 720p
  oni -w aniwatch             # Use aniwatch provider

`)
}

