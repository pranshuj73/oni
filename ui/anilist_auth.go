package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pranshuj73/oni/anilist"
	"github.com/pranshuj73/oni/config"
)

// AniListAuth handles AniList authentication
type AniListAuth struct {
	cfg           *config.Config
	styles        Styles
	help          help.Model
	textInput     textinput.Model
	universalKeys UniversalKeys
	err           string
	verifying     bool
	spinner       spinner.Model
}

// AniListAuthSuccessMsg is sent when authentication succeeds
type AniListAuthSuccessMsg struct {
	Client *anilist.Client
}

// AniListAuthErrorMsg is sent when authentication fails
type AniListAuthErrorMsg struct {
	Err error
}

// NewAniListAuth creates a new AniList authentication screen
func NewAniListAuth(cfg *config.Config) *AniListAuth {
	ti := textinput.New()
	ti.Placeholder = "Paste your AniList access token here"
	ti.Focus()
	ti.CharLimit = 0 // No limit - tokens can be very long
	ti.Width = 80

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#4A90E2"))

	m := &AniListAuth{
		cfg:           cfg,
		styles:        DefaultStyles(),
		help:          help.New(),
		textInput:     ti,
		universalKeys: DefaultUniversalKeys(),
		spinner:       s,
	}
	m.help.ShowAll = false
	return m
}

func (m *AniListAuth) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m *AniListAuth) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		// Handle universal keys
		switch {
		case key.Matches(msg, m.universalKeys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		case key.Matches(msg, m.universalKeys.Quit):
			return m, tea.Quit
		}

		// Handle auth-specific keys
		switch msg.String() {
		case "enter":
			if !m.verifying && m.textInput.Value() != "" {
				token := m.textInput.Value()
				// Clean the token - remove any hidden characters
				token = strings.TrimSpace(token)
				token = strings.Trim(token, "\n\r\t")
				m.verifying = true
				return m, m.verifyToken(token)
			}
		case "esc":
			if !m.verifying {
				return m, tea.Quit
			}
		}

	case AniListAuthSuccessMsg:
		// Authentication successful, don't return here
		// Let main.go handle this message
		return m, func() tea.Msg { return msg }

	case AniListAuthErrorMsg:
		m.verifying = false
		m.err = msg.Err.Error()
		return m, nil

	case tea.WindowSizeMsg:
		m.help.Width = msg.Width
	}

	// Update text input
	if !m.verifying {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *AniListAuth) View() string {
	s := "\n"
	s += GetBannerGradient() + "\n"
	s += m.styles.Subtitle.Render("Oni — Anime Streaming Client") + "\n\n"

	s += m.styles.Title.Render("Welcome to Oni!") + "\n\n"

	if m.verifying {
		s += m.spinner.View() + " " + m.styles.Info.Render("Verifying token...") + "\n\n"
	} else {
		s += m.styles.Info.Render("To use Oni, you need to connect your AniList account.") + "\n\n"

		s += m.styles.Prompt.Render("Step 1:") + " " + m.styles.Info.Render("Open this URL in your browser:") + "\n"
		s += m.styles.AnimeTitle.Render("  https://anilist.co/api/v2/oauth/authorize?client_id=32038&response_type=token") + "\n\n"

		s += m.styles.Prompt.Render("Step 2:") + " " + m.styles.Info.Render("Copy the access token from the page") + "\n\n"

		s += m.styles.Prompt.Render("Step 3:") + " " + m.styles.Info.Render("Paste it below:") + "\n"
		s += m.textInput.View() + "\n\n"

		if m.err != "" {
			s += m.styles.Error.Render("Error: "+m.err) + "\n\n"
		}
	}

	// Help
	helpKeys := anilistAuthKeyMap{
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "submit token"),
		),
		Esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "quit"),
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

// verifyToken verifies the token with AniList API
func (m *AniListAuth) verifyToken(token string) tea.Cmd {
	return func() tea.Msg {
		// Just trim whitespace like jerry.sh does - use token exactly as provided
		token = strings.TrimSpace(token)
		
		if token == "" {
			return AniListAuthErrorMsg{Err: fmt.Errorf("token cannot be empty")}
		}
		
		// Try to create a client with the token (exactly like jerry.sh uses it)
		client, err := anilist.NewClientWithToken(token)
		if err != nil {
			return AniListAuthErrorMsg{Err: fmt.Errorf("invalid token: %w", err)}
		}

		// Token is valid, save it
		if err := anilist.SaveToken(token); err != nil {
			return AniListAuthErrorMsg{Err: fmt.Errorf("failed to save token: %w", err)}
		}

		// Get and save user ID
		userID, err := client.GetUserID(context.Background())
		if err != nil {
			return AniListAuthErrorMsg{Err: fmt.Errorf("failed to get user ID: %w", err)}
		}

		if err := anilist.SaveUserID(userID); err != nil {
			return AniListAuthErrorMsg{Err: fmt.Errorf("failed to save user ID: %w", err)}
		}

		return AniListAuthSuccessMsg{Client: client}
	}
}


// anilistAuthKeyMap defines the keybindings for AniList auth
type anilistAuthKeyMap struct {
	Enter key.Binding
	Esc   key.Binding
}

func (k anilistAuthKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Enter, k.Esc}
}

func (k anilistAuthKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Enter, k.Esc},
	}
}

