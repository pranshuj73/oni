package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pranshuj73/oni/anilist"
	"github.com/pranshuj73/oni/config"
)

// AnimeSearchState represents the search state
type AnimeSearchState int

const (
	SearchInput AnimeSearchState = iota
	SearchResults
	SearchLoading
)

// searchInputKeyMap for search input help
type searchInputHelpKeyMap struct {
	Enter key.Binding
	Back  key.Binding
}

func (k searchInputHelpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Enter, k.Back}
}

func (k searchInputHelpKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Enter, k.Back}}
}

// searchResultsHelpKeyMap for search results help
type searchResultsHelpKeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Select        key.Binding
	SelectEpisode key.Binding
	Back          key.Binding
	Quit          key.Binding
}

func (k searchResultsHelpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.SelectEpisode, k.Back, k.Quit}
}

func (k searchResultsHelpKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up, k.Down, k.Select, k.SelectEpisode, k.Back, k.Quit}}
}

// backOnlyHelpKeyMap for back only help
type backOnlyHelpKeyMap struct {
	Back key.Binding
}

func (k backOnlyHelpKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Back}
}

func (k backOnlyHelpKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Back}}
}

// AnimeSearch represents the anime search model
type AnimeSearch struct {
	cfg     *config.Config
	client  *anilist.Client
	styles  Styles
	state   AnimeSearchState
	input   string
	cursor  int
	results []anilist.Anime
	err     error
	spinner spinner.Model
	help    help.Model
}

// NewAnimeSearch creates a new anime search
func NewAnimeSearch(cfg *config.Config, client *anilist.Client) *AnimeSearch {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))

	h := help.New()
	h.ShowAll = false

	return &AnimeSearch{
		cfg:     cfg,
		client:  client,
		styles:  DefaultStyles(),
		state:   SearchInput,
		input:   "",
		cursor:  0,
		results: []anilist.Anime{},
		spinner: s,
		help:    h,
	}
}

// Init initializes the anime search
func (m *AnimeSearch) Init() tea.Cmd {
	return m.spinner.Tick
}

// SearchResultMsg is sent when search results are ready
type SearchResultMsg struct {
	Results []anilist.Anime
	Err     error
}

// AnimeSelectedMsg is sent when an anime is selected
type AnimeSelectedMsg struct {
	Anime            anilist.Anime
	Entry            *anilist.MediaListEntry // Optional: entry from user's list with progress
	ShowEpisodeSelect bool                    // If true, show episode selection; if false, auto-play
}

// searchAnime performs the search
func (m *AnimeSearch) searchAnime() tea.Msg {
	results, err := m.client.SearchAnime(context.Background(), m.input, m.cfg.Advanced.ShowAdultContent)
	return SearchResultMsg{Results: results, Err: err}
}

// Update handles messages
func (m *AnimeSearch) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch m.state {
		case SearchInput:
			switch msg.String() {
			case "ctrl+c", "esc":
				return m, func() tea.Msg { return BackMsg{} }

			case "backspace":
				if len(m.input) > 0 {
					m.input = m.input[:len(m.input)-1]
				}
				return m, nil

			case "enter":
				if m.input != "" {
					m.state = SearchLoading
					return m, m.searchAnime
				}
				return m, nil

			default:
				// Only add printable characters (ignore special keys)
				if len(msg.Runes) > 0 {
					m.input += string(msg.Runes)
				}
				return m, nil
			}

		case SearchResults:
			switch msg.String() {
			case "ctrl+c", "esc":
				return m, func() tea.Msg { return BackMsg{} }

			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}

			case "down", "j":
				if m.cursor < len(m.results)-1 {
					m.cursor++
				}

			case "enter":
				if len(m.results) > 0 {
					return m, func() tea.Msg {
						return AnimeSelectedMsg{
							Anime:            m.results[m.cursor],
							ShowEpisodeSelect: false, // Auto-play (but will show episode select if no progress)
						}
					}
				}

			case "p":
				// Select anime and show episode selection
				if len(m.results) > 0 {
					return m, func() tea.Msg {
						return AnimeSelectedMsg{
							Anime:            m.results[m.cursor],
							ShowEpisodeSelect: true, // Show episode selection
						}
					}
				}

			case "backspace":
				m.state = SearchInput
				m.cursor = 0
				m.results = []anilist.Anime{}
			}
		}

	case SearchResultMsg:
		m.state = SearchResults
		m.results = msg.Results
		m.err = msg.Err
		m.cursor = 0
	}

	return m, nil
}

// View renders the anime search
func (m *AnimeSearch) View() string {
	switch m.state {
	case SearchInput:
		s := m.styles.Title.Render("Search Anime") + "\n\n"
		s += m.styles.Prompt.Render("Enter anime name:") + "\n"
		s += m.styles.MenuItem.Render(m.input + "█") + "\n\n"
		keys := searchInputHelpKeyMap{
			Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "search")),
			Back:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		}
		s += m.help.View(keys)
		return s

	case SearchLoading:
		s := m.styles.Title.Render("Search Anime") + "\n\n"
		s += fmt.Sprintf("%s %s\n", m.spinner.View(), m.styles.Info.Render("Searching..."))
		return s

	case SearchResults:
		s := m.styles.Title.Render("Search Results") + "\n\n"

		backKeys := backOnlyHelpKeyMap{
			Back: key.NewBinding(key.WithKeys("backspace"), key.WithHelp("backspace", "back")),
		}

		if m.err != nil {
			s += m.styles.Error.Render(fmt.Sprintf("Error: %v", m.err)) + "\n"
			s += m.help.View(backKeys)
			return s
		}

		if len(m.results) == 0 {
			s += m.styles.Info.Render("No results found") + "\n"
			s += m.help.View(backKeys)
			return s
		}

		for i, anime := range m.results {
			cursor := " "
			title := anime.Title.UserPreferred

			// Add episode count if available
			if anime.Episodes != nil {
				title = fmt.Sprintf("%s (%d episodes)", title, *anime.Episodes)
			}

			// Add start year if available
			if anime.StartDate.Year != nil {
				title = fmt.Sprintf("%s [%d]", title, *anime.StartDate.Year)
			}

			if m.cursor == i {
				cursor = ">"
				s += m.styles.SelectedItem.Render(cursor + " " + title) + "\n"
			} else {
				s += m.styles.MenuItem.Render(cursor + " " + title) + "\n"
			}
		}

		keys := searchResultsHelpKeyMap{
			Up:            key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
			Down:          key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
			Select:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "auto-play")),
			SelectEpisode: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "select episode")),
			Back:          key.NewBinding(key.WithKeys("backspace"), key.WithHelp("backspace", "back")),
			Quit:          key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "quit")),
		}
		s += "\n" + m.help.View(keys)
		return s
	}

	return ""
}

// BackMsg is sent when the user wants to go back
type BackMsg struct{}

