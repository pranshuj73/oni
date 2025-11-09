package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pranshuj73/oni/anilist"
	"github.com/pranshuj73/oni/config"
)

// AnimeListState represents the list state
type AnimeListState int

const (
	ListLoading AnimeListState = iota
	ListResults
	ListSearchInput
	ListSearchResults
	ListSearchLoading
)

// AnimeItem represents an anime entry in the list
type AnimeItem struct {
	Entry anilist.MediaListEntry
}

func (i AnimeItem) Title() string {
	return i.Entry.Media.Title.UserPreferred
}

func (i AnimeItem) Description() string {
	episodesTotal := "?"
	if i.Entry.Media.Episodes != nil {
		episodesTotal = fmt.Sprintf("%d", *i.Entry.Media.Episodes)
	}
	desc := fmt.Sprintf("Progress: %d/%s episodes", i.Entry.Progress, episodesTotal)
	if i.Entry.Score != nil && *i.Entry.Score > 0 {
		desc += fmt.Sprintf(" • Score: %.0f", *i.Entry.Score)
	}
	return desc
}

func (i AnimeItem) FilterValue() string {
	return i.Entry.Media.Title.UserPreferred
}

// SearchAnimeItem represents a search result anime
type SearchAnimeItem struct {
	Anime anilist.Anime
}

func (i SearchAnimeItem) Title() string {
	return i.Anime.Title.UserPreferred
}

func (i SearchAnimeItem) Description() string {
	episodesTotal := "?"
	if i.Anime.Episodes != nil {
		episodesTotal = fmt.Sprintf("%d", *i.Anime.Episodes)
	}
	return fmt.Sprintf("Episodes: %s", episodesTotal)
}

func (i SearchAnimeItem) FilterValue() string {
	return i.Anime.Title.UserPreferred
}

// AnimeList represents the anime list model
type AnimeList struct {
	cfg           *config.Config
	client        *anilist.Client
	styles        Styles
	state         AnimeListState
	tabIndex      int
	statuses      []string
	statusLabels  []string
	entries       map[string][]anilist.MediaListEntry
	lists         map[string]list.Model // One list per status tab
	err           error
	width         int
	height        int
	cacheLoaded   bool
	isRefreshing  bool
	spinner       spinner.Model
	help          help.Model
	keys          animeListKeyMap
	universalKeys UniversalKeys
	// Search fields
	searchInput   string
	searchResults []anilist.Anime
	searchList    list.Model
	// Cache tracking
	lastCacheTimestamp time.Time // Track when we last loaded from cache
}

// animeListKeyMap defines the keybindings for the anime list
type animeListKeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Left          key.Binding
	Right         key.Binding
	Select        key.Binding
	SelectEpisode key.Binding
	Search        key.Binding
	Refresh       key.Binding
	Back          key.Binding
}

// searchInputKeyMap defines the keybindings for search input
type searchInputKeyMap struct {
	Enter key.Binding
	Back  key.Binding
}

func (k searchInputKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Enter, k.Back}
}

func (k searchInputKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Enter, k.Back}}
}

// searchResultsKeyMap defines the keybindings for search results
type searchResultsKeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Select        key.Binding
	SelectEpisode key.Binding
	Back          key.Binding
}

func (k searchResultsKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.SelectEpisode, k.Back}
}

func (k searchResultsKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up, k.Down, k.Select, k.SelectEpisode, k.Back}}
}

// backOnlyKeyMap defines keybindings for error/empty states
type backOnlyKeyMap struct {
	Back key.Binding
}

func (k backOnlyKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Back}
}

func (k backOnlyKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Back}}
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k animeListKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Left, k.Right, k.Up, k.Down, k.Select, k.Back}
}

// FullHelp returns keybindings for the full help view
func (k animeListKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Left, k.Right, k.Up, k.Down},
		{k.Select, k.SelectEpisode, k.Search, k.Refresh},
		{k.Back},
	}
}

// DefaultAnimeListKeyMap returns the default keybindings
func DefaultAnimeListKeyMap() animeListKeyMap {
	return animeListKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "prev tab"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "next tab"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "auto-play"),
		),
		SelectEpisode: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "select episode"),
		),
		Search: key.NewBinding(
			key.WithKeys("n", "s"),
			key.WithHelp("n/s", "search"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "ctrl+c"),
			key.WithHelp("esc", "back"),
		),
	}
}

// Cache for anime lists
var animeListCache = make(map[string][]anilist.MediaListEntry)
var cacheValid = false
var cacheInitialized = false
var cacheTimestamp time.Time

// CacheData represents the cache file structure
type CacheData struct {
	Entries   map[string][]anilist.MediaListEntry `json:"entries"`
	Timestamp time.Time                           `json:"timestamp"`
}

// getCachePath returns the path to the cache file
func getCachePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(homeDir, ".oni", "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "anime_list_cache.json"), nil
}

// loadCacheFromDisk loads the cache from disk - ALWAYS valid, never expires
func loadCacheFromDisk() {
	if cacheInitialized {
		return
	}
	cacheInitialized = true
	reloadCacheFromDisk()
}

// reloadCacheFromDisk forces a reload of the cache from disk
func reloadCacheFromDisk() {
	cachePath, err := getCachePath()
	if err != nil {
		return
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		// No cache file exists, will load from API
		cacheValid = false
		return
	}

	var cacheData CacheData
	if err := json.Unmarshal(data, &cacheData); err != nil {
		// Invalid cache, will load from API
		cacheValid = false
		return
	}

	// Load cache regardless of age - show stale data immediately!
	animeListCache = cacheData.Entries
	cacheTimestamp = cacheData.Timestamp
	cacheValid = true
}

// saveCacheToDisk saves the cache to disk
func saveCacheToDisk() {
	cachePath, err := getCachePath()
	if err != nil {
		return
	}

	now := time.Now()
	cacheTimestamp = now
	cacheData := CacheData{
		Entries:   animeListCache,
		Timestamp: now,
	}

	data, err := json.Marshal(cacheData)
	if err != nil {
		return
	}

	os.WriteFile(cachePath, data, 0644)
}

// buildListItems converts MediaListEntry slice to list.Item slice
func buildListItems(entries []anilist.MediaListEntry) []list.Item {
	items := make([]list.Item, len(entries))
	for i, entry := range entries {
		items[i] = AnimeItem{Entry: entry}
	}
	return items
}

// createListForStatus creates a list component for a given status
func (m *AnimeList) createListForStatus(status string, width, height int) list.Model {
	entries := m.entries[status]
	items := buildListItems(entries)
	
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")). // White
		Background(lipgloss.Color("#4A90E2")). // Darker blue
		Bold(true).
		Padding(0, 1)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedTitle.Copy().
		Foreground(lipgloss.Color("#E0E0E0")) // Light gray
	
	// Ensure minimum dimensions
	if width < 20 {
		width = 20
	}
	if height < 10 {
		height = 10
	}
	
	// Calculate proper height: reserve 1 line for tabs, 2 lines for list title
	listHeight := height - 3
	if listHeight < 5 {
		listHeight = 5 // Minimum height
	}
	l := list.New(items, delegate, width, listHeight)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowFilter(true)
	l.DisableQuitKeybindings()
	l.SetShowHelp(false) // Disable built-in help - we use our own universal help
	
	// Get the status label and set title with count
	statusLabel := ""
	statusIndex := m.getStatusIndex(status)
	if statusIndex >= 0 && statusIndex < len(m.statusLabels) {
		statusLabel = m.statusLabels[statusIndex]
	}
	l.Title = fmt.Sprintf("%s (%d)", statusLabel, len(entries))
	
	return l
}

// getStatusIndex returns the index of a status in the statuses slice
func (m *AnimeList) getStatusIndex(status string) int {
	for i, s := range m.statuses {
		if s == status {
			return i
		}
	}
	return 0
}

// updateListsForAllStatuses creates/updates lists for all statuses
// It preserves filter state if a list is currently being filtered
func (m *AnimeList) updateListsForAllStatuses() {
	for _, status := range m.statuses {
		oldList, exists := m.lists[status]
		// Preserve filter state if list exists and is currently filtering or has filter applied
		var preservedFilterState list.FilterState
		var preservedFilterText string
		if exists {
			preservedFilterState = oldList.FilterState()
			preservedFilterText = oldList.FilterValue()
		}
		
		// Create new list with updated items
		newList := m.createListForStatus(status, m.width, m.height)
		
		// Restore filter state if it was filtering or had filter applied
		// SetFilterText automatically applies the filter and sets state to FilterApplied
		if exists && (preservedFilterState == list.Filtering || preservedFilterState == list.FilterApplied) && preservedFilterText != "" {
			newList.SetFilterText(preservedFilterText)
			// SetFilterText sets state to FilterApplied, but if it was Filtering, keep it as Filtering
			if preservedFilterState == list.Filtering {
				newList.SetFilterState(list.Filtering)
			}
		}
		
		m.lists[status] = newList
	}
}

// NewAnimeList creates a new anime list
func NewAnimeList(cfg *config.Config, client *anilist.Client) *AnimeList {
	// Load cache from disk on first access
	loadCacheFromDisk()

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))

	al := &AnimeList{
		cfg:    cfg,
		client: client,
		styles: DefaultStyles(),
		state:  ListLoading,
		statuses: []string{
			"CURRENT",
			"REPEATING",
			"COMPLETED",
			"PAUSED",
			"DROPPED",
			"PLANNING",
		},
		statusLabels: []string{
			"Watching",
			"Rewatching",
			"Completed",
			"Paused",
			"Dropped",
			"Plan to Watch",
		},
		tabIndex:     0,
		entries:      make(map[string][]anilist.MediaListEntry),
		lists:        make(map[string]list.Model),
		width:        80,
		height:       24,
		cacheLoaded:  false,
		isRefreshing: false,
		spinner:       s,
		help:          help.New(),
		keys:          DefaultAnimeListKeyMap(),
		universalKeys: DefaultUniversalKeys(),
	}
	// Start with short help by default
	al.help.ShowAll = false

		// Load from cache if available
		// Always reload cache from disk to get the latest data when creating new instance
		reloadCacheFromDisk()
		if cacheValid && len(animeListCache) > 0 {
			// Deep copy the cache to avoid reference issues
			al.entries = make(map[string][]anilist.MediaListEntry)
			for status, entries := range animeListCache {
				al.entries[status] = make([]anilist.MediaListEntry, len(entries))
				copy(al.entries[status], entries)
			}
			al.state = ListResults
			al.cacheLoaded = true
			al.lastCacheTimestamp = cacheTimestamp // Track when we loaded
			// Initialize lists from cache
			al.updateListsForAllStatuses()
		}

	return al
}

// Init initializes the anime list
func (m *AnimeList) Init() tea.Cmd {
	if m.cacheLoaded {
		// Cache exists! Show immediately and refresh in background if needed
		// Check if cache is recent (less than 5 minutes old)
		if !cacheTimestamp.IsZero() {
			timeSinceUpdate := time.Since(cacheTimestamp)
			if timeSinceUpdate < 5*time.Minute {
				// Cache is fresh, skip refresh
				return tea.Batch(m.spinner.Tick)
			}
		}
		// Cache is stale or timestamp unknown, refresh in background
		m.isRefreshing = true
		return tea.Batch(m.spinner.Tick, m.fetchAllListsAsync)
	}
	// No cache, show loading and fetch normally
	return tea.Batch(m.spinner.Tick, m.fetchAllLists)
}

// AllListsResultMsg is sent when all lists are ready
type AllListsResultMsg struct {
	AllEntries  map[string][]anilist.MediaListEntry
	Err         error
	IsRefresh   bool
}

// searchAnime performs the search
func (m *AnimeList) searchAnime() tea.Msg {
	results, err := m.client.SearchAnime(context.Background(), m.searchInput, m.cfg.Advanced.ShowAdultContent)
	return SearchResultMsg{Results: results, Err: err}
}

// fetchAllLists fetches all anime lists at once (synchronous)
func (m *AnimeList) fetchAllLists() tea.Msg {
	allEntries := make(map[string][]anilist.MediaListEntry)
	
	for _, status := range m.statuses {
		entries, err := m.client.GetAnimeList(context.Background(), status)
		if err != nil {
			return AllListsResultMsg{Err: err, IsRefresh: false}
		}
		allEntries[status] = entries
	}
	
	// Update cache (both memory and disk)
	animeListCache = allEntries
	cacheValid = true
	saveCacheToDisk()
	
	return AllListsResultMsg{AllEntries: allEntries, Err: nil, IsRefresh: false}
}

// fetchAllListsAsync fetches all anime lists in the background (for cache refresh)
func (m *AnimeList) fetchAllListsAsync() tea.Msg {
	allEntries := make(map[string][]anilist.MediaListEntry)
	
	for _, status := range m.statuses {
		entries, err := m.client.GetAnimeList(context.Background(), status)
		if err != nil {
			// Silently fail for background refresh
			return AllListsResultMsg{AllEntries: animeListCache, Err: nil, IsRefresh: true}
		}
		allEntries[status] = entries
	}
	
	// Update cache (both memory and disk)
	animeListCache = allEntries
	cacheValid = true
	saveCacheToDisk()
	
	return AllListsResultMsg{AllEntries: allEntries, Err: nil, IsRefresh: true}
}

// RefreshCacheInBackground refreshes the anime list cache in the background
// This can be called on app startup to pre-warm the cache
// It skips refresh if cache was updated less than 5 minutes ago to prevent rate limits
func RefreshCacheInBackground(cfg *config.Config, client *anilist.Client) {
	if client == nil || cfg.AniList.NoAniList {
		return
	}
	
	// Load cache from disk first
	loadCacheFromDisk()
	
	// Check if cache is recent (less than 5 minutes old)
	if cacheValid && !cacheTimestamp.IsZero() {
		timeSinceUpdate := time.Since(cacheTimestamp)
		if timeSinceUpdate < 5*time.Minute {
			// Cache is fresh, skip refresh
			return
		}
	}
	
	// Start background refresh
	ForceRefreshCacheInBackground(cfg, client)
}

// ForceRefreshCacheInBackground forces a cache refresh in the background
// This bypasses the 5-minute freshness check and is used when updates are made
func ForceRefreshCacheInBackground(cfg *config.Config, client *anilist.Client) {
	if client == nil || cfg.AniList.NoAniList {
		return
	}
	
	// Start background refresh
	go func() {
		statuses := []string{"CURRENT", "PLANNING", "COMPLETED", "DROPPED", "PAUSED", "REPEATING"}
		allEntries := make(map[string][]anilist.MediaListEntry)
		
		for _, status := range statuses {
			entries, err := client.GetAnimeList(context.Background(), status)
			if err != nil {
				// Silently fail for background refresh, keep existing cache
				return
			}
			allEntries[status] = entries
		}
		
		// Update cache (both memory and disk)
		animeListCache = allEntries
		cacheValid = true
		saveCacheToDisk()
	}()
}

// Update handles messages
func (m *AnimeList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Forward non-KeyMsg messages to list when filtering (needed for FilterMatchesMsg)
	// This must happen before our switch statement so FilterMatchesMsg can be processed
	// FilterMatchesMsg is sent by filterItems() command and needs to reach the list
	if m.state == ListResults {
		currentStatus := m.statuses[m.tabIndex]
		if currentList, exists := m.lists[currentStatus]; exists {
			// Forward non-KeyMsg messages to list (FilterMatchesMsg, etc.)
			// KeyMsg will be handled in the switch statement below
			if _, isKeyMsg := msg.(tea.KeyMsg); !isKeyMsg {
				updatedList, listCmd := currentList.Update(msg)
				m.lists[currentStatus] = updatedList
				if listCmd != nil {
					cmds = append(cmds, listCmd)
				}
			}
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		// Check if any list is currently filtering - if so, skip rebuild to preserve filter
		isAnyListFiltering := false
		for _, status := range m.statuses {
			if currentList, exists := m.lists[status]; exists {
				if currentList.FilterState() == list.Filtering || currentList.FilterState() == list.FilterApplied {
					isAnyListFiltering = true
					break
				}
			}
		}
		// Only update lists if not filtering (preserve filter state)
		if !isAnyListFiltering {
			m.updateListsForAllStatuses()
		}
		// Update search list if it exists
		if m.state == ListSearchResults && len(m.searchResults) > 0 {
			items := make([]list.Item, len(m.searchResults))
			for i, anime := range m.searchResults {
				items[i] = SearchAnimeItem{Anime: anime}
			}
			delegate := list.NewDefaultDelegate()
			delegate.Styles.SelectedTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("5")).
				Padding(0, 1)
			searchListHeight := m.height - 2 // Reserve 2 lines for help
			if searchListHeight < 5 {
				searchListHeight = 5
			}
			m.searchList = list.New(items, delegate, m.width, searchListHeight)
			m.searchList.SetShowStatusBar(false)
			m.searchList.SetFilteringEnabled(true)
			m.searchList.SetShowFilter(true)
			m.searchList.DisableQuitKeybindings()
			m.searchList.SetShowHelp(false) // Disable built-in help
			m.searchList.Title = "" // No title, we show it in the UI
		}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		// Check if cache has been updated since we last loaded
		// But don't rebuild lists if user is currently filtering (would reset filter)
		if cacheValid && !cacheTimestamp.IsZero() && !m.lastCacheTimestamp.IsZero() {
			if cacheTimestamp.After(m.lastCacheTimestamp) {
				// Check if any list is currently filtering - if so, skip rebuild
				isAnyListFiltering := false
				for _, status := range m.statuses {
					if currentList, exists := m.lists[status]; exists {
						if currentList.FilterState() == list.Filtering {
							isAnyListFiltering = true
							break
						}
					}
				}
				
				if !isAnyListFiltering {
					// Cache has been updated, reload from it
					m.entries = make(map[string][]anilist.MediaListEntry)
					for status, entries := range animeListCache {
						m.entries[status] = make([]anilist.MediaListEntry, len(entries))
						copy(m.entries[status], entries)
					}
					m.lastCacheTimestamp = cacheTimestamp
					// Rebuild all lists with new data
					m.updateListsForAllStatuses()
				}
			}
		}
		return m, cmd

	case tea.KeyMsg:
		switch m.state {
		case ListResults:
			currentStatus := m.statuses[m.tabIndex]
			currentList := m.lists[currentStatus]
			
			// Check filter state before updating (to detect state transitions)
			wasFiltering := currentList.FilterState() == list.Filtering
			hadFilterApplied := currentList.FilterState() == list.FilterApplied
			isEsc := msg.String() == "esc"
			
			// Always update the list first so it can process the key and enter filtering mode
			currentList, cmd = currentList.Update(msg)
			m.lists[currentStatus] = currentList
			cmds = append(cmds, cmd)
			
			// Check filter state after updating
			filterState := currentList.FilterState()
			
			// If Esc was pressed and filter was active, the list cleared it - don't go back
			if isEsc && (wasFiltering || hadFilterApplied) {
				// Filter was cleared by the list, just return
				return m, tea.Batch(cmds...)
			}
			
			// If we're filtering (actively typing) or filter is applied, handle keys specially
			if filterState == list.Filtering || filterState == list.FilterApplied {
				// For keys while actively filtering, let the list handle everything
				if filterState == list.Filtering {
					return m, tea.Batch(cmds...)
				}
			}
			
			// Check if Enter was pressed while filtering (confirms filter, don't select)
			justConfirmedFilter := wasFiltering && 
				filterState == list.FilterApplied && 
				msg.String() == "enter"
			
			// If Enter just confirmed the filter, don't handle it as selection
			if justConfirmedFilter {
				return m, tea.Batch(cmds...)
			}
			
			// Handle universal keys (but skip Esc if filter is active - already handled above)
			if m.state != ListSearchInput {
				// Don't handle Esc as quit if filter is active
				if msg.String() != "esc" || (filterState != list.Filtering && filterState != list.FilterApplied) {
					switch {
					case key.Matches(msg, m.universalKeys.Help):
						m.help.ShowAll = !m.help.ShowAll
						return m, nil
					case key.Matches(msg, m.universalKeys.Quit):
						return m, func() tea.Msg { return BackMsg{} }
					}
				}
			}
			
			// Handle tab switching and other special keys
			// Esc is already handled above when filter is active
			switch msg.String() {
			case "ctrl+c":
				return m, func() tea.Msg { return BackMsg{} }
			case "esc":
				// Only handle Esc as back if filter is not active
				if filterState != list.Filtering && filterState != list.FilterApplied {
					return m, func() tea.Msg { return BackMsg{} }
				}
				// If filter is active, it's already been handled above
				return m, tea.Batch(cmds...)

			case "left", "h":
				// Switch to previous tab
				if m.tabIndex > 0 {
					m.tabIndex--
				}
				return m, tea.Batch(cmds...)

			case "right", "l":
				// Switch to next tab
				if m.tabIndex < len(m.statuses)-1 {
					m.tabIndex++
				}
				return m, tea.Batch(cmds...)

			case "r":
				// Manual refresh
				if !m.isRefreshing {
					m.isRefreshing = true
					return m, tea.Batch(append(cmds, m.fetchAllLists)...)
				}
				return m, tea.Batch(cmds...)

			case "n", "s":
				// Start search
				m.state = ListSearchInput
				m.searchInput = ""
				m.searchResults = []anilist.Anime{}
				return m, tea.Batch(cmds...)
			}

			// Handle list selection (only when not filtering and not just confirmed filter)
			if selectedItem := currentList.SelectedItem(); selectedItem != nil {
				animeItem := selectedItem.(AnimeItem)
				switch msg.String() {
				case "enter":
					// Auto-play next episode
					return m, func() tea.Msg {
						return AnimeSelectedMsg{
							Anime:            animeItem.Entry.Media,
							Entry:            &animeItem.Entry,
							ShowEpisodeSelect: false,
						}
					}
				case "p":
					// Show episode selection
					return m, func() tea.Msg {
						return AnimeSelectedMsg{
							Anime:            animeItem.Entry.Media,
							Entry:            &animeItem.Entry,
							ShowEpisodeSelect: true,
						}
					}
				}
			}

		case ListSearchInput:
			// Handle universal keys in search input (but only quit, not help)
			if key.Matches(msg, m.universalKeys.Quit) {
				return m, func() tea.Msg { return BackMsg{} }
			}
			
			switch msg.String() {
			case "ctrl+c", "esc", "q":
				m.state = ListResults
				m.searchInput = ""
				m.searchResults = []anilist.Anime{}
				return m, nil

			case "backspace":
				if len(m.searchInput) > 0 {
					m.searchInput = m.searchInput[:len(m.searchInput)-1]
				}
				return m, nil

			case "enter":
				if m.searchInput != "" {
					m.state = ListSearchLoading
					return m, m.searchAnime
				}
				return m, nil

			default:
				// Only add printable characters (ignore special keys)
				if len(msg.Runes) > 0 {
					m.searchInput += string(msg.Runes)
				}
				return m, nil
			}

		case ListSearchResults:
			// Always update the list first to handle filter state changes
			m.searchList, cmd = m.searchList.Update(msg)
			cmds = append(cmds, cmd)
			
			// If we're actively filtering, don't handle other keys
			if m.searchList.FilterState() == list.Filtering {
				return m, tea.Batch(cmds...)
			}
			
			switch msg.String() {
			case "ctrl+c", "esc", "q":
				m.state = ListResults
				m.searchInput = ""
				m.searchResults = []anilist.Anime{}
				return m, nil

			case "backspace":
				m.state = ListSearchInput
				m.searchResults = []anilist.Anime{}
				return m, nil
			}

			// Handle selection
			if selectedItem := m.searchList.SelectedItem(); selectedItem != nil {
				searchItem := selectedItem.(SearchAnimeItem)
				switch msg.String() {
				case "enter":
					return m, func() tea.Msg {
						return AnimeSelectedMsg{
							Anime:            searchItem.Anime,
							ShowEpisodeSelect: false,
						}
					}
				case "p":
					return m, func() tea.Msg {
						return AnimeSelectedMsg{
							Anime:            searchItem.Anime,
							ShowEpisodeSelect: true,
						}
					}
				}
			}
		}

	case SearchResultMsg:
		if m.state == ListSearchLoading {
			m.state = ListSearchResults
			m.searchResults = msg.Results
			m.err = msg.Err
			
			// Create search list
			items := make([]list.Item, len(m.searchResults))
			for i, anime := range m.searchResults {
				items[i] = SearchAnimeItem{Anime: anime}
			}
			delegate := list.NewDefaultDelegate()
			delegate.Styles.SelectedTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("5")).
				Padding(0, 1)
			searchListHeight := m.height - 2 // Reserve 2 lines for help
			if searchListHeight < 5 {
				searchListHeight = 5
			}
			m.searchList = list.New(items, delegate, m.width, searchListHeight)
			m.searchList.SetShowStatusBar(false)
			m.searchList.SetFilteringEnabled(true)
			m.searchList.SetShowFilter(true)
			m.searchList.DisableQuitKeybindings()
			m.searchList.SetShowHelp(false) // Disable built-in help
			m.searchList.Title = "" // No title, we show it in the UI
		}

	case AllListsResultMsg:
		// Only change state if we're not in search mode
		if m.state != ListSearchInput && m.state != ListSearchLoading && m.state != ListSearchResults {
			m.state = ListResults
		}
		
		// Update entries with new data
		if msg.Err == nil {
			// Check if any list is currently filtering - if so, skip rebuild to preserve filter
			isAnyListFiltering := false
			for _, status := range m.statuses {
				if currentList, exists := m.lists[status]; exists {
					if currentList.FilterState() == list.Filtering || currentList.FilterState() == list.FilterApplied {
						isAnyListFiltering = true
						break
					}
				}
			}
			
			m.entries = msg.AllEntries
			m.err = nil
			m.lastCacheTimestamp = cacheTimestamp // Update our cache timestamp tracking
			// Only rebuild lists if not filtering (preserve filter state)
			if !isAnyListFiltering {
				m.updateListsForAllStatuses()
			}
		} else {
			m.err = msg.Err
		}
		
		m.isRefreshing = false
		
		// Only reset tab if not a background refresh
		if !msg.IsRefresh {
			m.tabIndex = 0
		}
	}

	if len(cmds) > 0 {
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

// View renders the anime list
func (m *AnimeList) View() string {
	// Handle search states
	if m.state == ListSearchInput {
		s := m.styles.Title.Render("Search Anime") + "\n\n"
		s += m.styles.Info.Render(fmt.Sprintf("Search: %s_", m.searchInput)) + "\n\n"
		helpKeys := ExtendedKeyMap{
			Universal: m.universalKeys,
			ViewKeys: []key.Binding{
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "search")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
			},
			ViewFull: [][]key.Binding{
				{key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "search")),
				 key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back"))},
			},
		}
		s += m.help.View(helpKeys)
		return s
	}

	if m.state == ListSearchLoading {
		s := m.styles.Title.Render("Searching...") + "\n\n"
		s += fmt.Sprintf("%s %s\n", m.spinner.View(), m.styles.Info.Render(fmt.Sprintf("Searching for: %s", m.searchInput)))
		return s
	}

	if m.state == ListSearchResults {
		backHelpKeys := ExtendedKeyMap{
			Universal: m.universalKeys,
			ViewKeys: []key.Binding{
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
			},
			ViewFull: [][]key.Binding{
				{key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back"))},
			},
		}
		
		if m.err != nil {
			s := m.styles.Error.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n"
			s += m.help.View(backHelpKeys)
			return s
		} else if len(m.searchResults) == 0 {
			s := m.styles.Info.Render("No results found") + "\n\n"
			s += m.help.View(backHelpKeys)
			return s
		}
		// Update search list height to use full available space
		searchListHeight := m.height - 2 // Reserve 2 lines for help
		if searchListHeight < 5 {
			searchListHeight = 5
		}
		if m.searchList.Height() != searchListHeight {
			m.searchList.SetHeight(searchListHeight)
		}
		s := m.searchList.View()
		
		helpKeys := ExtendedKeyMap{
			Universal: m.universalKeys,
			ViewKeys: []key.Binding{
				key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
				key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "auto-play")),
				key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "select episode")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
			},
			ViewFull: [][]key.Binding{
				{key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
				 key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down"))},
				{key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "auto-play")),
				 key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "select episode")),
				 key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back"))},
			},
		}
		s += "\n" + m.help.View(helpKeys)
		return s
	}

	if m.state == ListLoading && !m.cacheLoaded {
		// Only show loading screen if no cache available
		s := m.styles.Title.Render("Loading Anime Lists") + "\n\n"
		s += fmt.Sprintf("%s %s\n", m.spinner.View(), m.styles.Info.Render("Fetching all categories..."))
		return s
	}

	if m.err != nil {
		s := m.styles.Error.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n"
		helpKeys := ExtendedKeyMap{
			Universal: m.universalKeys,
			ViewKeys: []key.Binding{
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
			},
			ViewFull: [][]key.Binding{
				{key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back"))},
			},
		}
		s += m.help.View(helpKeys)
		return s
	}

	// Render tabs
	var tabs []string
	for i, label := range m.statusLabels {
		currentStatus := m.statuses[i]
		count := len(m.entries[currentStatus])
		
		tabLabel := fmt.Sprintf(" %s (%d) ", label, count)
		
		if i == m.tabIndex {
			// Active tab
			tab := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("5")).
				Padding(0, 1).
				Render(tabLabel)
			tabs = append(tabs, tab)
		} else {
			// Inactive tab
			tab := lipgloss.NewStyle().
				Foreground(lipgloss.Color("7")).
				Background(lipgloss.Color("8")).
				Padding(0, 1).
				Render(tabLabel)
			tabs = append(tabs, tab)
		}
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	s := tabBar + "\n"

	// Get current tab's list
	currentStatus := m.statuses[m.tabIndex]
	currentList := m.lists[currentStatus]

	// Update list height to use full available space
	// Reserve: 1 line for tabs, 2 lines for list title
	listHeight := m.height - 3
	if listHeight < 5 {
		listHeight = 5 // Minimum height
	}
	// Don't modify the list while filtering - this might reset the filter state
	filterState := currentList.FilterState()
	isFiltering := filterState == list.Filtering
	
	// Only update height and title when NOT filtering to avoid resetting filter
	if !isFiltering {
		// Update list height dynamically to fill available space (only if changed)
		if currentList.Height() != listHeight {
			currentList.SetHeight(listHeight)
		}
		
		// Update title with current count (only if changed to avoid resetting filter)
		newTitle := fmt.Sprintf("%s (%d)", m.statusLabels[m.tabIndex], len(m.entries[currentStatus]))
		if currentList.Title != newTitle {
			currentList.Title = newTitle
		}
	}
	
	m.lists[currentStatus] = currentList
	// Render the list component
	s += currentList.View()

	// Add help footer at the bottom
	helpKeys := ExtendedKeyMap{
		Universal: m.universalKeys,
		ViewKeys: []key.Binding{
			m.keys.Left, m.keys.Right, m.keys.Up, m.keys.Down,
			m.keys.Select, m.keys.SelectEpisode, m.keys.Search, m.keys.Refresh,
		},
		ViewFull: [][]key.Binding{
			{m.keys.Left, m.keys.Right, m.keys.Up, m.keys.Down},
			{m.keys.Select, m.keys.SelectEpisode, m.keys.Search, m.keys.Refresh},
		},
	}
	helpView := m.help.View(helpKeys)
	if m.isRefreshing {
		// Add spinner before help
		helpView = m.spinner.View() + " " + helpView
	}
	s += "\n" + helpView
	
	return s
}

// GetSelectedEntry returns the currently selected entry
func (m *AnimeList) GetSelectedEntry() *anilist.MediaListEntry {
	if m.state == ListResults {
		currentStatus := m.statuses[m.tabIndex]
		currentList := m.lists[currentStatus]
		if selectedItem := currentList.SelectedItem(); selectedItem != nil {
			animeItem := selectedItem.(AnimeItem)
			return &animeItem.Entry
		}
	}
	return nil
}



