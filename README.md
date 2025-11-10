# ONI - Anime Streaming Client

A modern rewrite of [jerry](https://github.com/justchokingaround/jerry) in Go with a beautiful TUI powered by Bubble Tea.

## Features

- 🎨 **Beautiful Terminal UI** - Interactive menus powered by Bubble Tea and Lipgloss
- 📑 **Tab-Based Interface** - Navigate between anime categories with arrow keys
- 🚀 **Instant Loading** - Cached lists load instantly on subsequent visits
- 📜 **Smart Scrolling** - Clean 5-item view with automatic scroll indicators
- 🔄 **Async Refresh** - Background updates without blocking UI
- 📺 **Multiple Providers** - Support for allanime, aniwatch, yugen, hdrezka, and aniworld
- 🔄 **AniList Integration** - Sync your watch progress, scores, and status
- 🎮 **Discord Presence** - Show what you're watching on Discord
- 🎬 **Multiple Players** - Support for mpv, vlc, and iina
- 📝 **Watch History** - Resume from where you left off
- ⚙️ **Easy Configuration** - INI-based config at `~/.oni/config.ini`

## Installation

### Prerequisites

- Go 1.21 or higher
- A video player (mpv, vlc, or iina)
- AniList account (optional, for progress tracking)

### Build from Source

```bash
git clone https://github.com/pranshuj73/oni
cd oni
go build -o oni
sudo mv oni /usr/local/bin/
```

## Usage

### Interactive Mode

Simply run oni to start the interactive TUI:

```bash
oni
```

### Command Line Options

```bash
oni [options] [query]

Options:
  -c             Continue watching from list
  -e             Edit configuration
  -d             Enable Discord presence
  -h             Show help
  -q <quality>   Video quality (e.g., 1080, 720)
  -v             Show version
  -w <provider>  Provider (allanime, aniwatch, yugen, hdrezka, aniworld)
  --sub-or-dub   Audio type (sub, dub)
```

### Examples

```bash
# Start interactive menu
oni

# Continue watching from your list
oni -c

# Search and watch anime in 720p
oni -q 720 one piece

# Use a specific provider
oni -w aniwatch demon slayer

# Enable Discord presence
oni -d naruto
```

## Configuration

Configuration is stored at `~/.oni/config.ini`. You can edit it directly or use the built-in editor:

```bash
oni -e
```

### Default Configuration

```ini
[player]
player = mpv
player_arguments = 

[provider]
provider = allanime
download_dir = 
quality = 1080

[anilist]
no_anilist = false
score_on_completion = false

[ui]
use_external_menu = false
image_preview = false
json_output = false

[playback]
sub_or_dub = sub
subs_language = english

[discord]
discord_presence = false

[advanced]
show_adult_content = false
```

## AniList Setup

1. Run oni for the first time
2. You'll be prompted to visit: https://anilist.co/api/v2/oauth/authorize?client_id=9857&response_type=token
3. Copy the access token and paste it into the terminal
4. Your token will be saved at `~/.local/share/jerry/anilist_token.txt`

## Project Structure

```
oni/
├── main.go                  # Entry point and app orchestration
├── config/                  # Configuration management
│   ├── config.go           # INI file handling
│   └── types.go            # Config structures
├── anilist/                # AniList API integration
│   ├── client.go           # GraphQL client
│   ├── auth.go             # Token management
│   ├── queries.go          # GraphQL queries
│   └── types.go            # Response types
├── providers/              # Anime providers
│   ├── provider.go         # Provider interface
│   ├── allanime.go         # AllAnime scraper
│   ├── aniwatch.go         # Aniwatch scraper
│   ├── yugen.go            # Yugen scraper
│   ├── hdrezka.go          # HDRezka scraper
│   └── aniworld.go         # Aniworld scraper
├── player/                 # Video player integration
│   ├── player.go           # Player interface
│   ├── mpv.go              # MPV implementation
│   ├── vlc.go              # VLC implementation
│   └── history.go          # Watch history tracking
├── discord/                # Discord Rich Presence
│   └── presence.go         # Presence management
└── ui/                     # Bubble Tea UI components
    ├── main_menu.go        # Main menu
    ├── anime_search.go     # Search interface
    ├── anime_list.go       # List viewer
    ├── episode_select.go   # Episode selector
    ├── update_progress.go  # Progress updater
    ├── config_editor.go    # Config editor
    └── styles.go           # UI styling
```

## Providers

### AllAnime (Default)
- Fast and reliable
- Good quality streams
- Extensive library

### Aniwatch
- High-quality streams
- Multiple subtitle options
- Good for popular anime

### Yugen
- Alternative source
- Decent quality
- Good uptime

### HDRezka
- Russian-focused provider
- Multiple quality options
- Full decryption support implemented

### Aniworld
- German provider
- Good selection
- M3U8 streams

## Data Storage

- **Config**: `~/.oni/config.ini`
- **AniList Token**: `~/.local/share/jerry/anilist_token.txt`
- **User ID**: `~/.local/share/jerry/anilist_user_id.txt`
- **Watch History**: `~/.local/share/jerry/jerry_history.txt`

## Keyboard Navigation

### Main Menu
- `↑/↓` or `j/k` - Navigate
- `Enter` - Select
- `q` - Quit

### Anime List (Tab-Based)
- `←/→` or `h/l` - Switch between tabs (categories)
- `↑/↓` or `j/k` - Navigate within list (auto-scrolls)
- `Enter` - Select anime
- `r` - Manually refresh list
- `Esc` - Return to main menu

**Performance Note:** Lists show max 5 items with scroll indicators. First load caches results - subsequent visits are instant!

### Search/List
- `↑/↓` or `j/k` - Navigate
- `Enter` - Select
- `Backspace` - Go back
- `Esc` - Return to main menu

### Config Editor
- `↑/↓` or `j/k` - Navigate
- `Enter` - Edit value
- `s` - Save configuration
- `Esc` - Return to main menu

## Pending Tasks

- [ ] **Download Anime Menu** - Create a download menu that allows searching for anime to download
- [ ] **Download Functionality** - Implement download functionality for single episodes and episode ranges

## Limitations
- Image preview in TUI is not yet implemented
- Resume from history shows a simple list (could be improved with better UI)

## Credits

- Original [jerry](https://github.com/justchokingaround/jerry) by justchokingaround
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) for styling
- [rich-go](https://github.com/hugolgst/rich-go) for Discord integration

## License

This project maintains compatibility with the original jerry license.

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## Disclaimer

This tool is for educational purposes only. Please support official anime streaming services.

