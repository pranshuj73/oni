package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pranshuj73/oni/config"
)

// MainMenu represents the main menu model
type MainMenu struct {
	cfg           *config.Config
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
}

// mainMenuKeyMap defines the keybindings for the main menu
type mainMenuKeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Select        key.Binding
	SelectEpisode key.Binding
	Quit          key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k mainMenuKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.Quit}
}

// FullHelp returns keybindings for the full help view
func (k mainMenuKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select, k.SelectEpisode},
		{k.Quit},
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
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// NewMainMenu creates a new main menu
func NewMainMenu(cfg *config.Config) *MainMenu {
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

// Init initializes the main menu
func (m *MainMenu) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles messages
func (m *MainMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
			m.selected = m.options[m.cursor]
			if m.selected == "Quit" {
				return m, tea.Quit
			}
			return m, func() tea.Msg {
				return MenuSelectionMsg{Selection: m.selected, ShowEpisodeSelect: false}
			}
		
		case key.Matches(msg, m.keys.SelectEpisode):
			// If on "Continue Watching", 's' key or Shift+Enter opens episode selection
			if m.options[m.cursor] == "Continue Watching" {
				m.selected = m.options[m.cursor]
				return m, func() tea.Msg {
					return MenuSelectionMsg{Selection: m.selected, ShowEpisodeSelect: true}
				}
			}
		}

	}

	return m, nil
}

// View renders the main menu
func (m *MainMenu) View() string {
	// Show colorful banner
	s := GetBannerGradient() + "\n"
	s += m.styles.Subtitle.Render("Oni — Anime Streaming Client") + "\n\n"

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
		
		if m.cursor < len(m.options) && m.options[m.cursor] == "Continue Watching" {
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

