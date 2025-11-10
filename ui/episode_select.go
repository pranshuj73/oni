package ui

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pranshuj73/oni/anilist"
	"github.com/pranshuj73/oni/config"
)

// EpisodeSelectState represents the episode selection state
type EpisodeSelectState int

const (
	EpisodeSubDubSelect EpisodeSelectState = iota
	EpisodeNumberInput
	EpisodeReady
)

// EpisodeSelect represents the episode selection model
type EpisodeSelect struct {
	cfg             *config.Config
	styles          Styles
	state           EpisodeSelectState
	anime           anilist.Anime
	progress        int
	episodesTotal   int
	episodeInput    string
	selectedEpisode int
	subOrDub        string
	subDubCursor    int
	err             error
	spinner         spinner.Model
	help            help.Model
}

// episodeSelectKeyMap defines the keybindings for episode select
type episodeSelectKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Back   key.Binding
}

func (k episodeSelectKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.Back}
}

func (k episodeSelectKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up, k.Down, k.Select, k.Back}}
}

// episodeInputKeyMap defines the keybindings for episode input
type episodeInputKeyMap struct {
	Play key.Binding
	Back key.Binding
}

func (k episodeInputKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Play, k.Back}
}

func (k episodeInputKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Play, k.Back}}
}

// NewEpisodeSelect creates a new episode selector
func NewEpisodeSelect(cfg *config.Config, anime anilist.Anime, progress int) *EpisodeSelect {
	episodesTotal := 9999
	if anime.Episodes != nil {
		episodesTotal = *anime.Episodes
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))

	h := help.New()
	h.ShowAll = false

	return &EpisodeSelect{
		cfg:           cfg,
		styles:        DefaultStyles(),
		state:         EpisodeSubDubSelect,
		anime:         anime,
		progress:      progress,
		episodesTotal: episodesTotal,
		subOrDub:      cfg.Playback.SubOrDub,
		subDubCursor:  0,
		spinner:       s,
		help:          h,
	}
}

// Init initializes the episode selector
func (m *EpisodeSelect) Init() tea.Cmd {
	// If sub_or_dub is already set, skip selection
	if m.cfg.Playback.SubOrDub != "" {
		m.subOrDub = m.cfg.Playback.SubOrDub
		m.state = EpisodeNumberInput
		
		// If progress is available (> 0), pre-select next episode
		// User can press enter to continue or type a different number
		if m.progress > 0 {
			m.selectedEpisode = m.progress + 1
		}
	}
	// Don't auto-play here - let user press Enter to play
	return m.spinner.Tick
}

// EpisodeReadyMsg is sent when episode selection is complete
type EpisodeReadyMsg struct {
	Episode  int
	SubOrDub string
}

// Update handles messages
func (m *EpisodeSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch m.state {
		case EpisodeSubDubSelect:
			switch msg.String() {
			case "ctrl+c", "esc", "q", "backspace":
				return m, func() tea.Msg { return BackMsg{} }

			case "up", "k":
				m.subDubCursor = 0

			case "down", "j":
				m.subDubCursor = 1

			case "enter":
				if m.subDubCursor == 0 {
					m.subOrDub = "sub"
				} else {
					m.subOrDub = "dub"
				}
				m.state = EpisodeNumberInput
			}

		case EpisodeNumberInput:
			switch msg.String() {
			case "ctrl+c", "esc", "q":
				return m, func() tea.Msg { return BackMsg{} }

			case "backspace":
				// Check if we should go back or delete character
				if len(m.episodeInput) == 0 {
					return m, func() tea.Msg { return BackMsg{} }
				}
				if len(m.episodeInput) > 0 {
					m.episodeInput = m.episodeInput[:len(m.episodeInput)-1]
				}

			case "enter":
				if m.episodeInput == "" {
					// If we already have a selected episode from progress, use it
					// Otherwise default to next episode after progress
					if m.selectedEpisode == 0 {
					m.selectedEpisode = m.progress + 1
					}
				} else {
					ep, err := strconv.Atoi(m.episodeInput)
					if err != nil || ep < 1 || ep > m.episodesTotal {
						m.err = fmt.Errorf("invalid episode number")
						return m, nil
					}
					m.selectedEpisode = ep
				}

				m.state = EpisodeReady
				return m, func() tea.Msg {
					return EpisodeReadyMsg{
						Episode:  m.selectedEpisode,
						SubOrDub: m.subOrDub,
					}
				}

			default:
				// Only accept numeric input
				if msg.String() >= "0" && msg.String() <= "9" {
					m.episodeInput += msg.String()
				}
			}
		}
	}

	return m, nil
}

// View renders the episode selector
func (m *EpisodeSelect) View() string {
	switch m.state {
	case EpisodeSubDubSelect:
		s := m.styles.Title.Render("Select Audio Type") + "\n\n"

		options := []string{"Sub", "Dub"}
		for i, opt := range options {
			cursor := " "
			if m.subDubCursor == i {
				cursor = ">"
				s += m.styles.SelectedItem.Render(cursor + " " + opt) + "\n"
			} else {
				s += m.styles.MenuItem.Render(cursor + " " + opt) + "\n"
			}
		}

		keys := episodeSelectKeyMap{
			Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
			Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
			Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
			Back:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		}
		s += "\n" + m.help.View(keys)
		return s

	case EpisodeNumberInput:
		s := m.styles.Title.Render(m.anime.Title.UserPreferred) + "\n\n"
		s += m.styles.Info.Render(fmt.Sprintf("Current progress: %d/%d episodes", m.progress, m.episodesTotal)) + "\n\n"
		nextEp := m.progress + 1
		if m.selectedEpisode > 0 {
			nextEp = m.selectedEpisode
		}
		if m.episodeInput == "" && m.progress > 0 {
			s += m.styles.Prompt.Render(fmt.Sprintf("Press enter to continue with episode %d (or type a different number):", nextEp)) + "\n"
		} else {
		s += m.styles.Prompt.Render("Enter episode number (or press enter for next):") + "\n"
		}
		s += m.styles.MenuItem.Render(m.episodeInput + "█") + "\n\n"

		if m.err != nil {
			s += m.styles.Error.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n"
		}

		keys := episodeInputKeyMap{
			Play: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "play")),
			Back: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		}
		s += m.help.View(keys)
		return s

	case EpisodeReady:
		// Loading is handled by main app, just return empty to avoid duplicate loaders
		return ""
	}

	return ""
}

// GetSelectedEpisode returns the selected episode number
func (m *EpisodeSelect) GetSelectedEpisode() int {
	return m.selectedEpisode
}

// GetSubOrDub returns the selected audio type
func (m *EpisodeSelect) GetSubOrDub() string {
	return m.subOrDub
}

