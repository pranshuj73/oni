package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pranshuj73/oni/anilist"
	"github.com/pranshuj73/oni/config"
	"github.com/pranshuj73/oni/player"
)

// MainMenu represents the main menu model
type MainMenu struct {
	cfg           *config.Config
	client        *anilist.Client
	styles        Styles
	cursor        int
	options       []string
	selected      string
	err           error
	help          help.Model
	keys          mainMenuKeyMap
	universalKeys UniversalKeys
	loadingMsg    string
	spinner       spinner.Model
	fetchingAnime bool
	incognitoMode bool // Runtime incognito mode (not persisted)
}

// mainMenuKeyMap defines the keybindings for the main menu
type mainMenuKeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Select        key.Binding
	SelectEpisode key.Binding
	Incognito     key.Binding
	Quit          key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k mainMenuKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.Incognito, k.Quit}
}

// FullHelp returns keybindings for the full help view
func (k mainMenuKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select, k.SelectEpisode},
		{k.Incognito, k.Quit},
	}
}

// DefaultMainMenuKeyMap returns the default keybindings
func DefaultMainMenuKeyMap() mainMenuKeyMap {
	return mainMenuKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		SelectEpisode: key.NewBinding(
			key.WithKeys("s", "shift+enter"),
			key.WithHelp("s", "select episode"),
		),
		Incognito: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "toggle incognito"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// NewMainMenu creates a new main menu
func NewMainMenu(cfg *config.Config) *MainMenu {
	return NewMainMenuWithClient(cfg, nil)
}

// NewMainMenuWithClient creates a new main menu with an AniList client
func NewMainMenuWithClient(cfg *config.Config, client *anilist.Client) *MainMenu {
	options := []string{
		"Continue Watching",
		"Watch Anime",
		"Update Progress/Status/Score",
		"Settings",
		"Quit",
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))

	mm := &MainMenu{
		cfg:           cfg,
		client:        client,
		styles:        DefaultStyles(),
		cursor:        0,
		options:       options,
		help:          help.New(),
		keys:          DefaultMainMenuKeyMap(),
		universalKeys: DefaultUniversalKeys(),
		spinner:       s,
	}
	// Start with short help by default
	mm.help.ShowAll = false
	return mm
}

// SetClient sets the AniList client and fetches continue watching anime
func (m *MainMenu) SetClient(client *anilist.Client) {
	m.client = client
}

// ContinueWatchingAnimeMsg is sent when the continue watching anime is fetched
type ContinueWatchingAnimeMsg struct {
	AnimeName string
	Episode   int
}

// Init initializes the main menu
func (m *MainMenu) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick}
	m.fetchingAnime = true
	cmds = append(cmds, m.fetchContinueWatchingAnime())
	return tea.Batch(cmds...)
}

// shortenTitle shortens an anime title by:
// 1. Using the part before ":" if it exists
// 2. Otherwise using the original title
func shortenTitle(title string) string {
	if idx := strings.Index(title, ":"); idx > 0 {
		short := strings.TrimSpace(title[:idx])
		if len(short) > 0 {
			return short
		}
	}
	return title
}

// fetchContinueWatchingAnime fetches the anime name for continue watching from local history
func (m *MainMenu) fetchContinueWatchingAnime() tea.Cmd {
	return func() tea.Msg {
		// Use incognito or normal history based on current mode
		history, err := player.LoadHistoryWithIncognito(m.incognitoMode)
		if err == nil && len(history) > 0 {
			// Find the entry with the most recent LastWatched timestamp
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
					// We can't determine when it was last watched without a proper timestamp
					continue
				}
				
				// Check if this is the most recent
				if lastEntry == nil || watchedTime.After(latestTime) {
					lastEntry = entry
					latestTime = watchedTime
				}
			}
			
			if lastEntry != nil {
				// Just shorten the stored title by splitting on colon
				shortTitle := shortenTitle(lastEntry.Title)
				
				// Show next episode (progress + 1) in the menu
				nextEpisode := lastEntry.Progress + 1
				// Don't exceed total episodes
				if lastEntry.EpisodesTotal > 0 && nextEpisode > lastEntry.EpisodesTotal {
					nextEpisode = lastEntry.EpisodesTotal
				}
				
				return ContinueWatchingAnimeMsg{
					AnimeName: shortTitle,
					Episode:   nextEpisode,
				}
			}
		}

		// No anime found
		return ContinueWatchingAnimeMsg{AnimeName: "", Episode: 0}
	}
}

// Update handles messages
func (m *MainMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ContinueWatchingAnimeMsg:
		m.fetchingAnime = false
		if msg.AnimeName != "" {
			m.options[0] = fmt.Sprintf("Continue Watching (%s • Episode %d)", msg.AnimeName, msg.Episode)
		} else {
			// No anime found, reset to default
			m.options[0] = "Continue Watching"
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.help.Width = msg.Width

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.universalKeys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil

		case key.Matches(msg, m.universalKeys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			} else {
				// Cycle to bottom
				m.cursor = len(m.options) - 1
			}

		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.options)-1 {
				m.cursor++
			} else {
				// Cycle to top
				m.cursor = 0
			}

		case key.Matches(msg, m.keys.Select):
			// Extract base selection name (remove anime name if present)
			selected := m.options[m.cursor]
			if strings.HasPrefix(selected, "Continue Watching") {
				m.selected = "Continue Watching"
			} else {
				m.selected = selected
			}
			if m.selected == "Quit" {
				return m, tea.Quit
			}
			return m, func() tea.Msg {
				return MenuSelectionMsg{Selection: m.selected, ShowEpisodeSelect: false}
			}
		
		case key.Matches(msg, m.keys.SelectEpisode):
			// If on "Continue Watching", 's' key or Shift+Enter opens episode selection
			if strings.HasPrefix(m.options[m.cursor], "Continue Watching") {
				m.selected = "Continue Watching"
				return m, func() tea.Msg {
					return MenuSelectionMsg{Selection: m.selected, ShowEpisodeSelect: true}
				}
			}

		case key.Matches(msg, m.keys.Incognito):
			// Toggle incognito mode
			m.incognitoMode = !m.incognitoMode
			// Update styles based on incognito mode
			if m.incognitoMode {
				m.styles = IncognitoStyles()
			} else {
				m.styles = DefaultStyles()
			}
			// If incognito history is preserved, update continue watching immediately
			if m.cfg.Playback.PersistIncognitoSessions {
				m.fetchingAnime = true
				return m, m.fetchContinueWatchingAnime()
			}
			return m, nil
		}

	}

	return m, nil
}

// View renders the main menu
func (m *MainMenu) View() string {
	// Show colorful banner (incognito or normal)
	var banner string
	var subtitle string
	if m.incognitoMode {
		banner = GetBannerGradientIncognito()
		subtitle = m.styles.Subtitle.Render("Oni — Anime Streaming Client (Private Mode)")
	} else {
		banner = GetBannerGradient()
		subtitle = m.styles.Subtitle.Render("Oni — Anime Streaming Client")
	}
	s := banner + "\n"
	s += subtitle + "\n\n"

	for i, option := range m.options {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
			s += m.styles.SelectedItem.Render(cursor + " " + option) + "\n"
		} else {
			s += m.styles.MenuItem.Render(cursor + " " + option) + "\n"
		}
	}

	if m.err != nil {
		s += "\n\n" + m.styles.Error.Render(fmt.Sprintf("Error: %v", m.err))
	}

	// Add footer at the bottom - show loading message or help
	if m.loadingMsg != "" {
		s += "\n" + m.spinner.View() + " " + m.loadingMsg
	} else {
		// Show different help based on selection
		var viewKeys []key.Binding
		var viewFull [][]key.Binding
		
		if m.cursor < len(m.options) && strings.HasPrefix(m.options[m.cursor], "Continue Watching") {
			// Show help with select episode option
			viewKeys = []key.Binding{m.keys.Up, m.keys.Down, 
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "auto-play")),
				m.keys.SelectEpisode}
			viewFull = [][]key.Binding{
				{m.keys.Up, m.keys.Down},
				{key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "auto-play")), m.keys.SelectEpisode},
			}
		} else {
			viewKeys = []key.Binding{m.keys.Up, m.keys.Down, m.keys.Select}
			viewFull = [][]key.Binding{
				{m.keys.Up, m.keys.Down, m.keys.Select},
			}
		}
		
		helpKeys := ExtendedKeyMap{
			Universal: m.universalKeys,
			ViewKeys:  viewKeys,
			ViewFull:  viewFull,
		}
		helpView := m.help.View(helpKeys)
		s += "\n" + helpView
	}

	return s
}

// MenuSelectionMsg is sent when a menu item is selected
type MenuSelectionMsg struct {
	Selection        string
	ShowEpisodeSelect bool // If true, show episode selection instead of auto-playing
}

// GetSelected returns the selected option
func (m *MainMenu) GetSelected() string {
	return m.selected
}

// SetLoadingMsg sets the loading message
func (m *MainMenu) SetLoadingMsg(msg string) {
	m.loadingMsg = msg
}

// GetIncognitoMode returns the current incognito mode state
func (m *MainMenu) GetIncognitoMode() bool {
	return m.incognitoMode
}

