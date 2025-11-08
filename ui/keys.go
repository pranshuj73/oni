package ui

import (
	"github.com/charmbracelet/bubbles/key"
)

// UniversalKeys defines keybindings available in all views
type UniversalKeys struct {
	Help key.Binding
	Quit key.Binding
}

// DefaultUniversalKeys returns the default universal keybindings
func DefaultUniversalKeys() UniversalKeys {
	return UniversalKeys{
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// UniversalKeyMap implements help.KeyMap for universal keys
type UniversalKeyMap struct {
	UniversalKeys
}

func (k UniversalKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k UniversalKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Help, k.Quit},
	}
}

// ExtendedKeyMap wraps a view-specific keymap with universal keys
type ExtendedKeyMap struct {
	Universal UniversalKeys
	ViewKeys  []key.Binding // View-specific keys for short help
	ViewFull  [][]key.Binding // View-specific keys for full help
}

func (k ExtendedKeyMap) ShortHelp() []key.Binding {
	// Append universal keys to view keys
	return append(k.ViewKeys, k.Universal.Help, k.Universal.Quit)
}

func (k ExtendedKeyMap) FullHelp() [][]key.Binding {
	// Append universal keys as the last column
	full := make([][]key.Binding, len(k.ViewFull))
	copy(full, k.ViewFull)
	full = append(full, []key.Binding{k.Universal.Help, k.Universal.Quit})
	return full
}

