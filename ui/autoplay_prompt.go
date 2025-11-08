package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pranshuj73/oni/config"
)

// AutoplayPrompt displays a prompt asking if the user wants to enable autoplay
type AutoplayPrompt struct {
	cfg         *config.Config
	styles      Styles
	help        help.Model
	animeTitle  string
	nextEpisode int
	selected    int // 0 = Yes (autoplay), 1 = No (return to menu)
	universalKeys UniversalKeys
}

// AutoplayPromptMsg is sent when user makes a choice
type AutoplayPromptMsg struct {
	EnableAutoplay bool
}

// NewAutoplayPrompt creates a new autoplay prompt
func NewAutoplayPrompt(cfg *config.Config, animeTitle string, nextEpisode int) *AutoplayPrompt {
	m := &AutoplayPrompt{
		cfg:         cfg,
		styles:      DefaultStyles(),
		help:        help.New(),
		animeTitle:  animeTitle,
		nextEpisode: nextEpisode,
		selected:    0,
		universalKeys: DefaultUniversalKeys(),
	}
	m.help.ShowAll = false
	return m
}

func (m *AutoplayPrompt) Init() tea.Cmd {
	return nil
}

func (m *AutoplayPrompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle universal keys
		switch {
		case key.Matches(msg, m.universalKeys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		case key.Matches(msg, m.universalKeys.Quit):
			return m, func() tea.Msg { return BackMsg{} }
		}

		// Handle prompt-specific keys
		switch msg.String() {
		case "up", "k", "left", "h":
			m.selected = 0
		case "down", "j", "right", "l":
			m.selected = 1
		case "enter":
			return m, func() tea.Msg {
				return AutoplayPromptMsg{
					EnableAutoplay: m.selected == 0,
				}
			}
		case "y", "Y":
			return m, func() tea.Msg {
				return AutoplayPromptMsg{EnableAutoplay: true}
			}
		case "n", "N":
			return m, func() tea.Msg {
				return AutoplayPromptMsg{EnableAutoplay: false}
			}
		case "esc":
			return m, func() tea.Msg { return BackMsg{} }
		}

	case tea.WindowSizeMsg:
		m.help.Width = msg.Width
	}

	return m, nil
}

func (m *AutoplayPrompt) View() string {
	s := "\n"
	s += m.styles.Title.Render(fmt.Sprintf("Continue watching %s?", m.animeTitle)) + "\n\n"
	s += m.styles.Info.Render(fmt.Sprintf("Episode %d completed!", m.nextEpisode-1)) + "\n\n"

	// Options
	yesStyle := m.styles.MenuItem
	noStyle := m.styles.MenuItem
	if m.selected == 0 {
		yesStyle = m.styles.SelectedItem
	} else {
		noStyle = m.styles.SelectedItem
	}

	s += yesStyle.Render("  Yes - Start autoplay (continue watching)") + "\n"
	s += noStyle.Render("  No  - Return to main menu") + "\n\n"

	// Help
	helpKeys := autoplayPromptKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Yes: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "yes"),
		),
		No: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "no"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
	}

	extendedKeys := ExtendedKeyMap{
		Universal: m.universalKeys,
		ViewKeys:  helpKeys.ShortHelp(),
		ViewFull:  helpKeys.FullHelp(),
	}

	s += "\n" + m.help.View(extendedKeys)
	return s
}

// autoplayPromptKeyMap defines the keybindings for the autoplay prompt
type autoplayPromptKeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Yes   key.Binding
	No    key.Binding
	Back  key.Binding
}

func (k autoplayPromptKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Yes, k.No, k.Enter, k.Back}
}

func (k autoplayPromptKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Yes, k.No, k.Back},
	}
}

