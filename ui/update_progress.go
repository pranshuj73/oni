package ui

import (
	"context"
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pranshuj73/oni/anilist"
	"github.com/pranshuj73/oni/config"
)

// UpdateType represents the type of update
type UpdateType int

const (
	UpdateEpisode UpdateType = iota
	UpdateStatus
	UpdateScore
)

// UpdateProgressState represents the update state
type UpdateProgressState int

const (
	UpdateTypeSelection UpdateProgressState = iota
	UpdateAnimeSelection
	UpdateInputEntry
	UpdateProcessing
	UpdateComplete
)

// UpdateProgress represents the update progress model
type UpdateProgress struct {
	cfg           *config.Config
	client        *anilist.Client
	styles        Styles
	state         UpdateProgressState
	updateType    UpdateType
	typeCursor    int
	animeList     *AnimeList
	selectedEntry *anilist.MediaListEntry
	inputValue    string
	statusCursor  int
	statuses      []string
	err           error
	successMsg    string
	spinner       spinner.Model
}

// NewUpdateProgress creates a new update progress UI
func NewUpdateProgress(cfg *config.Config, client *anilist.Client) *UpdateProgress {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))

	return &UpdateProgress{
		cfg:        cfg,
		client:     client,
		styles:     DefaultStyles(),
		state:      UpdateTypeSelection,
		typeCursor: 0,
		statuses: []string{
			"CURRENT",
			"REPEATING",
			"COMPLETED",
			"PAUSED",
			"DROPPED",
			"PLANNING",
		},
		spinner: s,
	}
}

// Init initializes the update progress UI
func (m *UpdateProgress) Init() tea.Cmd {
	return m.spinner.Tick
}

// UpdateCompleteMsg is sent when update is complete
type UpdateCompleteMsg struct {
	Success bool
	Message string
	Err     error
}

// performUpdate performs the actual update
func (m *UpdateProgress) performUpdate() tea.Msg {
	ctx := context.Background()

	switch m.updateType {
	case UpdateEpisode:
		episode, err := strconv.Atoi(m.inputValue)
		if err != nil {
			return UpdateCompleteMsg{Success: false, Err: fmt.Errorf("invalid episode number")}
		}

		status := "CURRENT"
		if m.selectedEntry.Media.Episodes != nil && episode >= *m.selectedEntry.Media.Episodes {
			status = "COMPLETED"
		}

		err = m.client.UpdateProgress(ctx, m.selectedEntry.MediaID, episode, status)
		if err != nil {
			return UpdateCompleteMsg{Success: false, Err: err}
		}

		return UpdateCompleteMsg{
			Success: true,
			Message: fmt.Sprintf("Updated progress to %d episodes", episode),
		}

	case UpdateStatus:
		newStatus := m.statuses[m.statusCursor]
		err := m.client.UpdateStatus(ctx, m.selectedEntry.MediaID, newStatus)
		if err != nil {
			return UpdateCompleteMsg{Success: false, Err: err}
		}

		return UpdateCompleteMsg{
			Success: true,
			Message: fmt.Sprintf("Updated status to %s", newStatus),
		}

	case UpdateScore:
		score, err := strconv.ParseFloat(m.inputValue, 64)
		if err != nil {
			return UpdateCompleteMsg{Success: false, Err: fmt.Errorf("invalid score")}
		}

		err = m.client.UpdateScore(ctx, m.selectedEntry.MediaID, score)
		if err != nil {
			return UpdateCompleteMsg{Success: false, Err: err}
		}

		return UpdateCompleteMsg{
			Success: true,
			Message: fmt.Sprintf("Updated score to %.1f", score),
		}
	}

	return UpdateCompleteMsg{Success: false, Err: fmt.Errorf("unknown update type")}
}

// Update handles messages
func (m *UpdateProgress) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch m.state {
		case UpdateTypeSelection:
			switch msg.String() {
			case "ctrl+c", "esc", "q", "backspace":
				return m, func() tea.Msg { return BackMsg{} }

			case "up", "k":
				if m.typeCursor > 0 {
					m.typeCursor--
				}

			case "down", "j":
				if m.typeCursor < 2 {
					m.typeCursor++
				}

		case "enter":
			m.updateType = UpdateType(m.typeCursor)
			m.animeList = NewAnimeList(m.cfg, m.client)
			m.state = UpdateAnimeSelection
			return m, m.animeList.Init()
			}

		case UpdateAnimeSelection:
			// Handle back navigation
			if msg.String() == "ctrl+c" || msg.String() == "esc" || msg.String() == "q" || msg.String() == "backspace" {
				m.state = UpdateTypeSelection
				m.animeList = nil
				return m, nil
			}

			// Check if user pressed Enter to select
			if msg.String() == "enter" || msg.String() == "p" {
				selectedEntry := m.animeList.GetSelectedEntry()
				if selectedEntry != nil {
					m.selectedEntry = selectedEntry
					m.state = UpdateInputEntry
					m.statusCursor = 0
					return m, nil
				}
				}

			// Delegate to anime list for navigation
			if m.animeList != nil {
				var cmd tea.Cmd
				_, cmd = m.animeList.Update(msg)
				return m, cmd
			}

		case UpdateInputEntry:
			switch m.updateType {
			case UpdateStatus:
				switch msg.String() {
				case "ctrl+c", "esc", "q", "backspace":
					return m, func() tea.Msg { return BackMsg{} }

				case "up", "k":
					if m.statusCursor > 0 {
						m.statusCursor--
					}

				case "down", "j":
					if m.statusCursor < len(m.statuses)-1 {
						m.statusCursor++
					}

				case "enter":
					m.state = UpdateProcessing
					return m, m.performUpdate
				}

			default: // UpdateEpisode or UpdateScore
				switch msg.String() {
				case "ctrl+c", "esc", "q":
					return m, func() tea.Msg { return BackMsg{} }

				case "backspace":
					// Check if we should go back or delete character
					if len(m.inputValue) == 0 {
						return m, func() tea.Msg { return BackMsg{} }
					}
					if len(m.inputValue) > 0 {
						m.inputValue = m.inputValue[:len(m.inputValue)-1]
					}

				case "enter":
					if m.inputValue != "" {
						m.state = UpdateProcessing
						return m, m.performUpdate
					}

				default:
					// Accept numeric input and decimal point for score
					if (msg.String() >= "0" && msg.String() <= "9") ||
						(msg.String() == "." && m.updateType == UpdateScore) {
						m.inputValue += msg.String()
					}
				}
			}

		case UpdateComplete:
			switch msg.String() {
			case "enter", "esc", "ctrl+c", "q", "backspace":
				return m, func() tea.Msg { return BackMsg{} }
			}
		}

	case UpdateCompleteMsg:
		m.state = UpdateComplete
		if msg.Success {
			m.successMsg = msg.Message
			m.err = nil
			// Trigger background cache refresh after successful update
			// Use ForceRefreshCacheInBackground to bypass 5-minute check
			if m.client != nil && !m.cfg.AniList.NoAniList {
				ForceRefreshCacheInBackground(m.cfg, m.client)
			}
		} else {
			m.err = msg.Err
		}
	}

	return m, nil
}

// View renders the update progress UI
func (m *UpdateProgress) View() string {
	switch m.state {
	case UpdateTypeSelection:
		s := m.styles.Title.Render("Update Anime") + "\n\n"

		options := []string{"Update Episodes Watched", "Update Status", "Update Score"}
		for i, opt := range options {
			cursor := " "
			if m.typeCursor == i {
				cursor = ">"
				s += m.styles.SelectedItem.Render(cursor + " " + opt) + "\n"
			} else {
				s += m.styles.MenuItem.Render(cursor + " " + opt) + "\n"
			}
		}

		s += "\n" + m.styles.Help.Render("↑/↓: navigate • enter: select • esc: back")
		return s

	case UpdateAnimeSelection:
		if m.animeList != nil {
			return m.animeList.View()
		}
		return ""

	case UpdateInputEntry:
		if m.selectedEntry == nil {
			return ""
		}

		s := m.styles.Title.Render(m.selectedEntry.Media.Title.UserPreferred) + "\n\n"

		switch m.updateType {
		case UpdateEpisode:
			s += m.styles.Info.Render(fmt.Sprintf("Current progress: %d episodes", m.selectedEntry.Progress)) + "\n\n"
			s += m.styles.Prompt.Render("Enter new episode number:") + "\n"
			s += m.styles.MenuItem.Render(m.inputValue + "█") + "\n\n"
			s += m.styles.Help.Render("enter: update • esc: back")

		case UpdateStatus:
			s += m.styles.Info.Render(fmt.Sprintf("Current status: %s", m.selectedEntry.Status)) + "\n\n"
			s += m.styles.Prompt.Render("Select new status:") + "\n"

			for i, status := range m.statuses {
				cursor := " "
				if m.statusCursor == i {
					cursor = ">"
					s += m.styles.SelectedItem.Render(cursor + " " + status) + "\n"
				} else {
					s += m.styles.MenuItem.Render(cursor + " " + status) + "\n"
				}
			}

			s += "\n" + m.styles.Help.Render("↑/↓: navigate • enter: update • esc: back")

		case UpdateScore:
			currentScore := "N/A"
			if m.selectedEntry.Score != nil {
				currentScore = fmt.Sprintf("%.1f", *m.selectedEntry.Score)
			}
			s += m.styles.Info.Render(fmt.Sprintf("Current score: %s", currentScore)) + "\n\n"
			s += m.styles.Prompt.Render("Enter new score (0-100):") + "\n"
			s += m.styles.MenuItem.Render(m.inputValue + "█") + "\n\n"
			s += m.styles.Help.Render("enter: update • esc: back")
		}

		return s

	case UpdateProcessing:
		return fmt.Sprintf("%s %s\n", m.spinner.View(), m.styles.Info.Render("Updating..."))

	case UpdateComplete:
		if m.err != nil {
			s := m.styles.Error.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n"
			s += m.styles.Help.Render("press any key to continue")
			return s
		}

		s := m.styles.Success.Render(m.successMsg) + "\n\n"
		s += m.styles.Help.Render("press any key to continue")
		return s
	}

	return ""
}

