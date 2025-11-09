package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pranshuj73/oni/config"
)

// ConfigEditorState represents the config editor state
type ConfigEditorState int

const (
	ConfigMenuSelection ConfigEditorState = iota
	ConfigTextEdit
	ConfigToggleEdit
	ConfigSelectEdit
	ConfigSaving
	ConfigSaved
)

// ConfigEditor represents the config editor model
type ConfigEditor struct {
	cfg                *config.Config
	styles             Styles
	state              ConfigEditorState
	cursor             int
	configItems        []ConfigItem
	err                error
	textInput          textinput.Model
	selectList         list.Model
	selectOptions      []string
	selectCursor       int
	help               help.Model
	universalKeys       UniversalKeys
	prevIncognitoState bool // Track previous incognito state to detect toggle off
}

// ConfigItem represents a configuration item
type ConfigItem struct {
	Name        string
	DisplayName string
	Value       interface{}
	Type        ConfigItemType
	Category    string
	Options     []string // For select type
}

// ConfigItemType represents the type of config item
type ConfigItemType int

const (
	ConfigTypeText ConfigItemType = iota
	ConfigTypeToggle
	ConfigTypeSelect
)

// NewConfigEditor creates a new config editor
func NewConfigEditor(cfg *config.Config) *ConfigEditor {
	items := []ConfigItem{
		{"player", "Player", cfg.Player.Player, ConfigTypeText, "Player", nil},
		{"player_arguments", "Player Arguments", cfg.Player.PlayerArguments, ConfigTypeText, "Player", nil},
		{"provider", "Provider", cfg.Provider.Provider, ConfigTypeSelect, "Provider", []string{"allanime", "aniwatch", "yugen", "hdrezka", "aniworld"}},
		{"quality", "Quality", cfg.Provider.Quality, ConfigTypeSelect, "Provider", []string{"1080", "720", "480", "360", "240", "best", "worst"}},
		{"sub_or_dub", "Sub or Dub", cfg.Playback.SubOrDub, ConfigTypeSelect, "Playback", []string{"sub", "dub"}},
		{"subs_language", "Subtitles Language", cfg.Playback.SubsLanguage, ConfigTypeText, "Playback", nil},
		{"persist_incognito_sessions", "Persist Incognito Sessions", cfg.Playback.PersistIncognitoSessions, ConfigTypeToggle, "Playback", nil},
		{"discord_presence", "Discord Presence", cfg.Discord.DiscordPresence, ConfigTypeToggle, "Discord", nil},
		{"show_adult_content", "Show Adult Content", cfg.Advanced.ShowAdultContent, ConfigTypeToggle, "Advanced", nil},
	}

	ti := textinput.New()
	ti.Placeholder = "Enter value..."
	ti.CharLimit = 0

	ce := &ConfigEditor{
		cfg:           cfg,
		styles:        DefaultStyles(),
		state:         ConfigMenuSelection,
		cursor:        0,
		configItems:   items,
		textInput:     ti,
		help:          help.New(),
		universalKeys: DefaultUniversalKeys(),
	}
	ce.help.ShowAll = false
	return ce
}

// Init initializes the config editor
func (m *ConfigEditor) Init() tea.Cmd {
	return nil
}

// ConfigSavedMsg is sent when config is saved
type ConfigSavedMsg struct {
	Err error
}

// saveConfig saves the configuration
func (m *ConfigEditor) saveConfig() tea.Msg {
	// Note: Incognito mode is now runtime-only (toggled with 'p' key)
	// We only handle persist_incognito_sessions setting here
	err := config.Save(m.cfg)
	return ConfigSavedMsg{Err: err}
}

// Update handles messages
func (m *ConfigEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle universal keys
		switch {
		case key.Matches(msg, m.universalKeys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		case key.Matches(msg, m.universalKeys.Quit):
			if m.state == ConfigMenuSelection {
				return m, func() tea.Msg { return BackMsg{} }
			}
		}

		switch m.state {
		case ConfigMenuSelection:
			switch msg.String() {
			case "esc", "q", "backspace":
				return m, func() tea.Msg { return BackMsg{} }

			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}

			case "down", "j":
				if m.cursor < len(m.configItems)-1 {
					m.cursor++
				}

			case "enter":
				item := m.configItems[m.cursor]
				switch item.Type {
				case ConfigTypeText:
					m.textInput.SetValue(fmt.Sprintf("%v", item.Value))
					m.textInput.Focus()
					m.state = ConfigTextEdit
					return m, textinput.Blink

				case ConfigTypeToggle:
					// Toggle boolean value
					currentValue := item.Value.(bool)
					m.applyConfigChange(item.Name, !currentValue)
					m.configItems[m.cursor].Value = !currentValue

				case ConfigTypeSelect:
					// Show select list
					m.selectOptions = item.Options
					m.selectCursor = 0
					// Find current value index
					for i, opt := range item.Options {
						if opt == fmt.Sprintf("%v", item.Value) {
							m.selectCursor = i
							break
						}
					}
					m.state = ConfigSelectEdit
					m.buildSelectList()
				}

			case "s":
				m.state = ConfigSaving
				return m, m.saveConfig
			}

		case ConfigTextEdit:
			switch msg.String() {
			case "esc", "q", "backspace":
				m.state = ConfigMenuSelection
				m.textInput.Blur()

			case "enter":
				item := &m.configItems[m.cursor]
				value := m.textInput.Value()
				m.applyConfigChange(item.Name, value)
				item.Value = value
				m.state = ConfigMenuSelection
				m.textInput.Blur()
			}

			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd

		case ConfigSelectEdit:
			// Don't match any of the keys below if we're actively filtering.
			if m.selectList.FilterState() == list.Filtering {
				var cmd tea.Cmd
				m.selectList, cmd = m.selectList.Update(msg)
				// Update cursor based on selected item
				if selectedItem := m.selectList.SelectedItem(); selectedItem != nil {
					item := selectedItem.(selectItem)
					for i, opt := range m.selectOptions {
						if opt == item.title {
							m.selectCursor = i
							break
						}
					}
				}
				return m, cmd
			}
			
			switch msg.String() {
			case "esc", "q", "backspace":
				m.state = ConfigMenuSelection
				return m, nil

			case "up", "k":
				if m.selectCursor > 0 {
					m.selectCursor--
					m.selectList.Select(m.selectCursor)
				}
				return m, nil

			case "down", "j":
				if m.selectCursor < len(m.selectOptions)-1 {
					m.selectCursor++
					m.selectList.Select(m.selectCursor)
				}
				return m, nil

			case "enter":
				item := &m.configItems[m.cursor]
				selectedValue := m.selectOptions[m.selectCursor]
				m.applyConfigChange(item.Name, selectedValue)
				item.Value = selectedValue
				m.state = ConfigMenuSelection
				return m, nil
			}
			
			// Delegate to list component for other keys (like filtering)
			var cmd tea.Cmd
			m.selectList, cmd = m.selectList.Update(msg)
			// Sync cursor position
			if selectedItem := m.selectList.SelectedItem(); selectedItem != nil {
				item := selectedItem.(selectItem)
				for i, opt := range m.selectOptions {
					if opt == item.title {
						m.selectCursor = i
						break
					}
				}
			}
			return m, cmd

		case ConfigSaved:
			switch msg.String() {
			case "enter", "esc":
				return m, func() tea.Msg { return BackMsg{} }
			}
		}

	case tea.WindowSizeMsg:
		m.help.Width = msg.Width
		if m.state == ConfigSelectEdit {
			m.buildSelectList()
		}

	case ConfigSavedMsg:
		m.state = ConfigSaved
		m.err = msg.Err
	}

	return m, nil
}

// buildSelectList builds the select list for dropdown
func (m *ConfigEditor) buildSelectList() {
	items := make([]list.Item, len(m.selectOptions))
	for i, opt := range m.selectOptions {
		items[i] = selectItem{title: opt, selected: i == m.selectCursor}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#4A90E2")).
		Bold(true).
		Padding(0, 1)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedTitle.Copy().
		Foreground(lipgloss.Color("#E0E0E0"))

	// Use reasonable dimensions for the select list
	// Reserve space for title, info text, and help
	listWidth := 40
	listHeight := 10
	if len(m.selectOptions) < listHeight {
		listHeight = len(m.selectOptions)
	}
	if listHeight < 3 {
		listHeight = 3
	}
	
	l := list.New(items, delegate, listWidth, listHeight)
	l.Title = "Select Option"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowFilter(true)
	l.SetShowHelp(false)
	l.Select(m.selectCursor)
	m.selectList = l
}

// selectItem represents an item in the select list
type selectItem struct {
	title    string
	selected bool
}

func (i selectItem) Title() string       { return i.title }
func (i selectItem) Description() string { return "" }
func (i selectItem) FilterValue() string { return i.title }

// applyConfigChange applies a configuration change
func (m *ConfigEditor) applyConfigChange(name string, value interface{}) {
	switch name {
	case "player":
		m.cfg.Player.Player = fmt.Sprintf("%v", value)
	case "player_arguments":
		m.cfg.Player.PlayerArguments = fmt.Sprintf("%v", value)
	case "quality":
		m.cfg.Provider.Quality = fmt.Sprintf("%v", value)
	case "provider":
		m.cfg.Provider.Provider = fmt.Sprintf("%v", value)
	case "sub_or_dub":
		m.cfg.Playback.SubOrDub = fmt.Sprintf("%v", value)
	case "subs_language":
		m.cfg.Playback.SubsLanguage = fmt.Sprintf("%v", value)
	case "persist_incognito_sessions":
		if boolVal, ok := value.(bool); ok {
			m.cfg.Playback.PersistIncognitoSessions = boolVal
		} else if strVal, ok := value.(string); ok {
			m.cfg.Playback.PersistIncognitoSessions = (strVal == "true")
		}
	case "discord_presence":
		if boolVal, ok := value.(bool); ok {
			m.cfg.Discord.DiscordPresence = boolVal
		} else if strVal, ok := value.(string); ok {
			m.cfg.Discord.DiscordPresence = (strVal == "true")
		}
	case "show_adult_content":
		if boolVal, ok := value.(bool); ok {
			m.cfg.Advanced.ShowAdultContent = boolVal
		} else if strVal, ok := value.(string); ok {
			m.cfg.Advanced.ShowAdultContent = (strVal == "true")
		}
	}
}

// View renders the config editor
func (m *ConfigEditor) View() string {
	switch m.state {
	case ConfigMenuSelection:
		s := m.styles.Title.Render("Settings") + "\n\n"

		// Display items in order (don't group by category to maintain order)
		for i, item := range m.configItems {
			cursor := " "
			var display string
			
			// Show category header if this is the first item of a category
			if i == 0 || m.configItems[i-1].Category != item.Category {
				s += m.styles.AnimeTitle.Render(item.Category) + "\n"
			}
			
			switch item.Type {
			case ConfigTypeText:
				display = fmt.Sprintf("%s: %v", item.DisplayName, item.Value)
			case ConfigTypeToggle:
				status := "OFF"
				if item.Value.(bool) {
					status = "ON"
				}
				display = fmt.Sprintf("%s: [%s]", item.DisplayName, status)
			case ConfigTypeSelect:
				display = fmt.Sprintf("%s: %v", item.DisplayName, item.Value)
			}

			if m.cursor == i {
				cursor = ">"
				s += m.styles.SelectedItem.Render(cursor + " " + display) + "\n"
			} else {
				s += m.styles.MenuItem.Render(cursor + " " + display) + "\n"
			}
		}
		s += "\n"

		// Help
		helpKeys := configMenuKeyMap{
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
				key.WithHelp("enter", "edit"),
			),
			Save: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "save"),
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

	case ConfigTextEdit:
		item := m.configItems[m.cursor]
		s := m.styles.Title.Render("Settings") + "\n\n"
		s += m.styles.Info.Render(fmt.Sprintf("Editing: %s", item.DisplayName)) + "\n\n"
		s += m.styles.Prompt.Render("Value:") + "\n"
		s += m.textInput.View() + "\n\n"
		
		helpKeys := configEditKeyMap{
			Enter: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "save"),
			),
			Back: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "cancel"),
			),
		}

		extendedKeys := ExtendedKeyMap{
			Universal: m.universalKeys,
			ViewKeys:  helpKeys.ShortHelp(),
			ViewFull:  helpKeys.FullHelp(),
		}

		s += m.help.View(extendedKeys)
		return s

	case ConfigSelectEdit:
		item := m.configItems[m.cursor]
		s := m.styles.Title.Render("Settings") + "\n\n"
		s += m.styles.Info.Render(fmt.Sprintf("Select: %s", item.DisplayName)) + "\n\n"
		
		// Use list component view for filtering support
		s += m.selectList.View()
		s += "\n"
		
		helpKeys := configSelectKeyMap{
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
			Back: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "cancel"),
			),
		}

		extendedKeys := ExtendedKeyMap{
			Universal: m.universalKeys,
			ViewKeys:  helpKeys.ShortHelp(),
			ViewFull:  helpKeys.FullHelp(),
		}

		s += m.help.View(extendedKeys)
		return s

	case ConfigSaving:
		return m.styles.Info.Render("Saving settings...") + "\n"

	case ConfigSaved:
		if m.err != nil {
			s := m.styles.Error.Render(fmt.Sprintf("Error saving settings: %v", m.err)) + "\n\n"
			s += m.styles.Help.Render("press any key to continue")
			return s
		}

		s := m.styles.Success.Render("Settings saved successfully!") + "\n\n"
		s += m.styles.Help.Render("press any key to continue")
		return s
	}

	return ""
}

// Key maps for help
type configMenuKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Save   key.Binding
	Back   key.Binding
}

func (k configMenuKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.Save, k.Back}
}

func (k configMenuKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select, k.Save},
		{k.Back},
	}
}

type configEditKeyMap struct {
	Enter key.Binding
	Back  key.Binding
}

func (k configEditKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Enter, k.Back}
}

func (k configEditKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Enter, k.Back},
	}
}

type configSelectKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Back   key.Binding
}

func (k configSelectKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.Back}
}

func (k configSelectKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select},
		{k.Back},
	}
}
