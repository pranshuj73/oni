package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles contains all the lipgloss styles used in the UI
type Styles struct {
	Title          lipgloss.Style
	Subtitle       lipgloss.Style
	MenuItem       lipgloss.Style
	SelectedItem   lipgloss.Style
	Info           lipgloss.Style
	Error          lipgloss.Style
	Success        lipgloss.Style
	Prompt         lipgloss.Style
	Border         lipgloss.Style
	Help           lipgloss.Style
	StatusBar      lipgloss.Style
	AnimeTitle     lipgloss.Style
	EpisodeInfo    lipgloss.Style
}

// DefaultStyles returns the default styles with enhanced colors
func DefaultStyles() Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#4A90E2")). // Darker blue
			Padding(0, 1),

		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5B9BD5")). // Medium blue
			Padding(0, 1),

		MenuItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D0D0D0")). // Light gray
			Padding(0, 2),

		SelectedItem: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")). // White text
			Background(lipgloss.Color("#4A90E2")).  // Darker blue
			Padding(0, 2),

		Info: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5B9BD5")). // Medium blue
			Padding(0, 1),

		Error: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#E06C75")). // Soft red
			Padding(0, 1),

		Success: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#98C379")). // Green
			Padding(0, 1),

		Prompt: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5C07B")). // Gold
			Padding(0, 1),

		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#4A90E2")). // Darker blue
			Padding(1, 2),

		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")). // Medium gray
			Padding(0, 1),

		StatusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).  // White
			Background(lipgloss.Color("#4A90E2")). // Darker blue
			Padding(0, 1),

		AnimeTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#A78BFA")). // Lighter purple
			Padding(0, 1),

		EpisodeInfo: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D0D0D0")). // Light gray
			Padding(0, 1),
	}
}

