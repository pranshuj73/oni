# ONI Development Roadmap & TODO

**Last Updated:** 2026-02-02
**Version:** 0.1.4 → 0.2.0+

This document tracks all planned improvements, bug fixes, refactoring tasks, and new features for the ONI project.

---

## Table of Contents

1. [Critical Bug Fixes](#critical-bug-fixes)
2. [High Priority Tasks](#high-priority-tasks)
3. [Refactoring Tasks](#refactoring-tasks)
4. [Stub Feature Completion](#stub-feature-completion)
5. [Security Improvements](#security-improvements)
6. [Performance Optimizations](#performance-optimizations)
7. [New Features & Enhancements](#new-features--enhancements)
8. [Documentation](#documentation)
9. [Long-term Vision](#long-term-vision)

---

## Critical Bug Fixes

### 🔴 P0: Production Blockers

- [x] **~~Fix VLC Player Implementation~~** - REMOVED (Low priority, MPV is sufficient)
- [x] **~~Fix IINA Player Implementation~~** - REMOVED (Low priority, MPV is sufficient)

- [ ] **Fix History File Format Bug** (`player/history.go:111-126`, `player/history.go:283`)
  - [ ] Migrate from tab-separated format to JSON
  - [ ] Create migration script for existing history files
  - [ ] Add version field to history format for future migrations
  - [ ] Handle anime titles with special characters (tabs, newlines)
  - [ ] Fix format mismatch between `SaveHistoryEntryWithIncognito` and `DeleteHistoryEntry`
  - [ ] Add atomic file write operations
  - [ ] Add file corruption recovery
  - **Impact:** Data corruption when titles contain tabs, history deletion breaks format
  - **Files:** `player/history.go`

- [ ] **Add HTTP Client Timeouts** (All HTTP clients)
  - [ ] Add 60-second timeout to `anilist/client.go:29`
  - [ ] Add 60-second timeout to `providers/allanime.go:30`
  - [ ] Add 60-second timeout to `providers/aniwatch.go`
  - [ ] Add 60-second timeout to `providers/yugen.go`
  - [ ] Add 60-second timeout to `providers/hdrezka.go`
  - [ ] Add 60-second timeout to `providers/aniworld.go`
  - [ ] Configure connection pooling (MaxIdleConns, IdleConnTimeout)
  - [ ] Add context cancellation support
  - **Impact:** Requests can hang indefinitely, application becomes unresponsive
  - **Files:** `anilist/client.go`, `providers/*.go`

---

## High Priority Tasks

### 🟡 P1: Important Improvements

- [ ] **Replace String Manipulation with JSON Parsing** (`providers/allanime.go:229-232`)
  - [ ] Define proper Go structs for AllAnime API response
  - [ ] Replace string replacement with `json.Unmarshal()`
  - [ ] Add error handling for malformed JSON
  - [ ] Add validation of parsed data
  - **Impact:** Fragile parsing breaks on API changes
  - **Files:** `providers/allanime.go`

- [ ] **Add Input Validation**
  - [ ] Validate configuration values at load time
    - [ ] Validate `player` (mpv, vlc, iina)
    - [ ] Validate `provider` (allanime, aniwatch, yugen, hdrezka, aniworld)
    - [ ] Validate `quality` (1080, 720, 480, 360)
    - [ ] Validate `sub_or_dub` (sub, dub)
  - [ ] Sanitize anime titles before URL use
    - [ ] Use `url.QueryEscape()` in all providers
    - [ ] Replace in `providers/allanime.go:53`
    - [ ] Replace in `providers/aniwatch.go`
    - [ ] Replace in `providers/yugen.go`
    - [ ] Replace in `providers/hdrezka.go`
    - [ ] Replace in `providers/aniworld.go`
  - [ ] Add bounds checking for episode numbers
  - [ ] Validate quality values before provider API calls
  - **Impact:** Invalid config causes runtime errors, URL injection risk
  - **Files:** `config/config.go`, `providers/*.go`

- [ ] **Fix HDRezka Regex Parsing** (`providers/hdrezka.go:55`)
  - [ ] Replace regex with proper JSON parsing
  - [ ] Handle escaped quotes in titles
  - [ ] Handle newlines and special characters
  - [ ] Add unit tests for edge cases
  - **Impact:** Crashes on titles with special characters
  - **Files:** `providers/hdrezka.go`

- [ ] **Fix Race Condition in Incognito Mode** (`main.go:453`)
  - [ ] Capture incognito state at playback initialization
  - [ ] Lock incognito state for duration of playback
  - [ ] Add mutex for incognito mode access
  - [ ] Add unit tests for concurrent access
  - **Impact:** History might save when user expects incognito
  - **Files:** `main.go`

- [ ] **Implement Retry Logic for Provider Calls**
  - [ ] Create `providers/retry.go` with retry helper
  - [ ] Implement exponential backoff (2^n seconds)
  - [ ] Add configurable max retries (default: 3)
  - [ ] Add circuit breaker for repeated failures
  - [ ] Add retry metrics/logging
  - [ ] Apply to all provider `GetEpisodeInfo` calls
  - [ ] Apply to all provider `GetVideoLink` calls
  - **Impact:** Transient network failures cause immediate failure
  - **Files:** `providers/retry.go`, `providers/*.go`

---

## Refactoring Tasks

### Code Quality Improvements

- [ ] **Deduplicate Episode Completion Logic**
  - [ ] Extract to `utils/episode.go`
  - [ ] Create `IsEpisodeComplete(percentageProgress float64) bool`
  - [ ] Create `GetNextEpisode(current, total, percentage) int`
  - [ ] Create constant `CompletionThreshold = 95.0`
  - [ ] Replace logic in `main.go:512-535`
  - [ ] Replace logic in `ui/main_menu.go`
  - [ ] Add unit tests
  - **Files:** `main.go`, `ui/main_menu.go`, new `utils/episode.go`

- [ ] **Extract Common Provider Utilities**
  - [ ] Create `providers/utils.go`
  - [ ] Extract URL building helper
  - [ ] Extract quality parsing helper
  - [ ] Extract sub/dub selection helper
  - [ ] Extract error wrapping patterns
  - [ ] Refactor all providers to use utilities
  - **Files:** `providers/*.go`, new `providers/utils.go`

- [ ] **Centralize Constants**
  - [ ] Create `constants/constants.go`
  - [ ] Move Discord app ID
  - [ ] Move API URLs
  - [ ] Move default quality values
  - [ ] Move completion threshold
  - [ ] Move timeout values
  - [ ] Update all references
  - **Files:** New `constants/constants.go`, all files

- [ ] **Improve Error Handling Consistency**
  - [ ] Create `errors/errors.go` for custom error types
  - [ ] Define `ErrProviderNotFound`
  - [ ] Define `ErrPlayerNotFound`
  - [ ] Define `ErrInvalidConfig`
  - [ ] Define `ErrNetworkTimeout`
  - [ ] Define `ErrAnimeNotFound`
  - [ ] Replace string error messages with typed errors
  - [ ] Add error wrapping helpers
  - **Files:** New `errors/errors.go`, all files

- [ ] **Refactor State Machine in main.go**
  - [ ] Extract state machine to `app/state.go`
  - [ ] Create state transition diagram documentation
  - [ ] Add state validation
  - [ ] Add state transition logging
  - [ ] Add unit tests for state transitions
  - **Files:** `main.go`, new `app/state.go`

- [ ] **Improve Configuration Management**
  - [ ] Add `Config.Validate() error` method
  - [ ] Add `Config.SetDefaults()` method
  - [ ] Add configuration versioning for future migrations
  - [ ] Add configuration change detection
  - [ ] Add hot reload support (optional)
  - **Files:** `config/config.go`

---

## Stub Feature Completion

### Complete or Remove Stub Features

- [x] **~~VLC Player Full Implementation~~** - REMOVED (Low priority, MPV is sufficient)
- [x] **~~IINA Player Full Implementation~~** - REMOVED (Low priority, MPV is sufficient)

- [ ] **Image Preview Feature**
  - [ ] Design image preview UI
  - [ ] Implement terminal image rendering (using kitty/iTerm2 protocols)
  - [ ] Add image fetching from AniList
  - [ ] Add caching for images
  - [ ] Integrate with anime list UI
  - [ ] Integrate with search UI
  - [ ] Add fallback for terminals without image support
  - **Alternative:** Remove `image_preview` config flag
  - **Files:** `ui/anime_list.go`, `ui/anime_search.go`, `config/config.go`

- [ ] **JSON Output Feature**
  - [ ] Design JSON output schema
  - [ ] Implement `--json` flag
  - [ ] Output anime search results as JSON
  - [ ] Output episode list as JSON
  - [ ] Output watch history as JSON
  - [ ] Output configuration as JSON
  - [ ] Add documentation for JSON output
  - **Alternative:** Remove `json_output` config flag
  - **Files:** `main.go`, `config/config.go`

- [ ] **External Menu Feature**
  - [ ] Design external menu integration
  - [ ] Support `fzf` for anime selection
  - [ ] Support `rofi` for anime selection
  - [ ] Support `dmenu` for anime selection
  - [ ] Add external menu configuration options
  - [ ] Add documentation for external menus
  - **Alternative:** Remove `use_external_menu` config flag
  - **Files:** `main.go`, `ui/`, `config/config.go`

- [ ] **Download Feature**
  - [ ] Implement download to `download_dir`
  - [ ] Add progress bar for downloads
  - [ ] Support resume on failed downloads
  - [ ] Add download queue
  - [ ] Add download history
  - [ ] Support batch downloads
  - **Alternative:** Remove `download_dir` config option
  - **Files:** New `download/` package, `config/config.go`

- [ ] **Persist Incognito Sessions**
  - [ ] Implement session persistence flag
  - [ ] Add incognito session storage
  - [ ] Add session cleanup logic
  - [ ] Add configuration option
  - **Alternative:** Keep runtime-only, remove config flag
  - **Files:** `config/config.go`, `player/history.go`

---

## Security Improvements

### Security Hardening

- [ ] **Make Discord App ID Configurable**
  - [ ] Support `ONI_DISCORD_APP_ID` environment variable
  - [ ] Add `discord_app_id` to config file
  - [ ] Update documentation
  - [ ] Default to current app ID if not set
  - **Files:** `discord/presence.go`, `config/config.go`

- [ ] **Implement Token Storage Security**
  - [ ] Research OS keychain integration (keyring library)
  - [ ] Implement keychain storage for Linux (Secret Service)
  - [ ] Implement keychain storage for macOS (Keychain)
  - [ ] Implement keychain storage for Windows (Credential Manager)
  - [ ] Add fallback to file-based storage
  - [ ] Add migration from file-based to keychain
  - **Files:** `anilist/auth.go`

- [ ] **Add Request Rate Limiting**
  - [ ] Implement rate limiter for AniList API
  - [ ] Implement rate limiter for provider APIs
  - [ ] Add configurable rate limits
  - [ ] Add backoff on rate limit errors
  - **Files:** `anilist/client.go`, `providers/*.go`

- [ ] **Implement Request Signing (Optional)**
  - [ ] Research provider API requirements
  - [ ] Add request signing for supported providers
  - [ ] Add HMAC signature generation
  - **Files:** `providers/*.go`

- [ ] **Add Input Sanitization**
  - [ ] Sanitize all user inputs before URL use
  - [ ] Use `url.QueryEscape()` consistently
  - [ ] Validate file paths
  - [ ] Sanitize player arguments
  - **Files:** All files handling user input

- [ ] **Add Security Audit Logging**
  - [ ] Log authentication attempts
  - [ ] Log configuration changes
  - [ ] Log external command executions
  - [ ] Add security event log rotation
  - **Files:** `logger/logger.go`, all relevant packages

---

## Performance Optimizations

### Improve Application Performance

- [ ] **Implement Connection Pooling**
  - [ ] Configure `http.Transport` with connection limits
  - [ ] Set `MaxIdleConns` to 100
  - [ ] Set `MaxIdleConnsPerHost` to 10
  - [ ] Set `IdleConnTimeout` to 90 seconds
  - [ ] Set `TLSHandshakeTimeout` to 10 seconds
  - [ ] Apply to all HTTP clients
  - **Files:** `anilist/client.go`, `providers/*.go`

- [ ] **Add Provider Response Caching**
  - [ ] Create `providers/response_cache.go`
  - [ ] Implement in-memory cache with TTL (5-10 minutes)
  - [ ] Cache episode info responses
  - [ ] Cache video link responses
  - [ ] Add cache invalidation
  - [ ] Add cache statistics
  - **Files:** New `providers/response_cache.go`, `providers/*.go`

- [ ] **Implement Parallel Provider Queries**
  - [ ] Create `GetEpisodeWithFallback()` function
  - [ ] Query multiple providers in parallel
  - [ ] Use first successful response
  - [ ] Add timeout for parallel queries
  - [ ] Add provider priority configuration
  - **Files:** `providers/provider.go`

- [ ] **Add Lazy Loading for Anime Lists**
  - [ ] Implement pagination for large lists
  - [ ] Load lists in chunks of 50-100 items
  - [ ] Add infinite scroll in UI
  - [ ] Add loading indicators
  - **Files:** `ui/anime_list.go`, `anilist/client.go`

- [ ] **Optimize History File Loading**
  - [ ] Use streaming JSON parser
  - [ ] Load history entries lazily
  - [ ] Index history by media ID
  - [ ] Add LRU cache for recent entries
  - **Files:** `player/history.go`

- [ ] **Add Database Option for Large Datasets**
  - [ ] Research SQLite integration
  - [ ] Implement optional SQLite backend
  - [ ] Migrate history to SQLite
  - [ ] Migrate cache to SQLite
  - [ ] Add configuration option for storage backend
  - **Files:** New `storage/` package

---

## New Features & Enhancements

### User-Requested Features

- [ ] **Add Batch Download Support**
  - [ ] Design batch download UI
  - [ ] Implement episode range selection
  - [ ] Add download queue management
  - [ ] Add parallel downloads (configurable limit)
  - [ ] Add download progress tracking
  - [ ] Add pause/resume for downloads
  - **Files:** New `download/` package, `ui/download.go`

- [ ] **Add Subtitle Customization**
  - [ ] Add subtitle font configuration
  - [ ] Add subtitle size configuration
  - [ ] Add subtitle color configuration
  - [ ] Pass custom subtitle options to MPV
  - [ ] Add subtitle preview in config editor
  - **Files:** `config/config.go`, `player/mpv.go`

- [ ] **Add Watchlist Management**
  - [ ] Create local watchlist (separate from AniList)
  - [ ] Add "Add to Watchlist" option
  - [ ] Add "Remove from Watchlist" option
  - [ ] Display watchlist in main menu
  - [ ] Sync with AniList "Planning" list
  - **Files:** New `watchlist/` package, `ui/main_menu.go`

- [ ] **Add Episode Notifications**
  - [ ] Detect new episodes for watching anime
  - [ ] Send desktop notifications (using beeep library)
  - [ ] Add configurable notification frequency
  - [ ] Add "New Episodes" section in main menu
  - **Files:** New `notifications/` package, `ui/main_menu.go`

- [ ] **Add Multi-Language Support (i18n)**
  - [ ] Setup i18n infrastructure (using go-i18n)
  - [ ] Extract all UI strings
  - [ ] Add English translation (default)
  - [ ] Add Japanese translation
  - [ ] Add Spanish translation
  - [ ] Add configuration for language selection
  - **Files:** New `i18n/` package, all UI files

- [ ] **Add Anime Recommendations**
  - [ ] Fetch recommendations from AniList
  - [ ] Display recommendations in main menu
  - [ ] Add "Recommended for You" section
  - [ ] Add recommendation filters (genre, year, rating)
  - **Files:** `anilist/client.go`, `ui/main_menu.go`

- [ ] **Add Watch Statistics**
  - [ ] Track total watch time
  - [ ] Track episodes watched
  - [ ] Track favorite genres
  - [ ] Display statistics in UI
  - [ ] Export statistics as JSON
  - **Files:** New `stats/` package, `ui/stats.go`

- [ ] **Add Keyboard Shortcut Customization**
  - [ ] Make keybindings configurable
  - [ ] Add keybindings configuration section
  - [ ] Support vim-style keybindings
  - [ ] Support emacs-style keybindings
  - [ ] Add keybindings editor in TUI
  - **Files:** `config/config.go`, `ui/keys.go`

- [ ] **Add Theme Customization**
  - [ ] Make colors configurable
  - [ ] Add preset themes (dark, light, catppuccin, dracula)
  - [ ] Support custom color schemes
  - [ ] Add theme preview in config editor
  - **Files:** `config/config.go`, `ui/styles.go`

- [ ] **Add MAL (MyAnimeList) Support**
  - [ ] Implement MAL API client
  - [ ] Support MAL authentication
  - [ ] Sync watch progress with MAL
  - [ ] Add configuration option to choose AniList or MAL
  - [ ] Support both AniList and MAL simultaneously
  - **Files:** New `mal/` package, `config/config.go`

---

## Documentation

### Improve Documentation

- [ ] **API Documentation**
  - [ ] Add GoDoc comments to all exported functions
  - [ ] Add GoDoc comments to all exported types
  - [ ] Add package-level documentation
  - [ ] Generate API documentation website
  - [ ] Add code examples in GoDoc

- [ ] **Architecture Documentation**
  - [ ] Create `docs/ARCHITECTURE.md`
  - [ ] Document state machine diagram
  - [ ] Document data flow diagrams
  - [ ] Document provider integration flow
  - [ ] Document player integration flow
  - [ ] Document AniList integration flow

- [ ] **Contributing Guide**
  - [ ] Create `CONTRIBUTING.md`
  - [ ] Document development setup
  - [ ] Document testing guidelines
  - [ ] Document PR process
  - [ ] Document code style guide
  - [ ] Add contributor covenant code of conduct

- [ ] **User Guide**
  - [ ] Create `docs/USER_GUIDE.md`
  - [ ] Document all features
  - [ ] Add screenshots/GIFs
  - [ ] Document keyboard shortcuts
  - [ ] Add troubleshooting section
  - [ ] Add FAQ section

- [ ] **Provider Documentation**
  - [ ] Create `docs/PROVIDERS.md`
  - [ ] Document each provider's API
  - [ ] Document provider limitations
  - [ ] Document how to add new providers
  - [ ] Add provider comparison table

- [ ] **Configuration Reference**
  - [ ] Create `docs/CONFIGURATION.md`
  - [ ] Document all configuration options
  - [ ] Add configuration examples
  - [ ] Document advanced configurations
  - [ ] Add environment variable reference

---

## Long-term Vision

### Future Enhancements (v0.3.0+)

- [ ] **Plugin System**
  - [ ] Design plugin architecture
  - [ ] Support Go plugins (plugin package)
  - [ ] Support custom providers via plugins
  - [ ] Support custom players via plugins
  - [ ] Create plugin registry
  - [ ] Add plugin marketplace

- [ ] **Web Interface**
  - [ ] Create web-based UI (using Go templates or React)
  - [ ] Support remote control of TUI
  - [ ] Add web-based configuration
  - [ ] Add mobile-responsive design
  - [ ] Support multiple concurrent users

- [ ] **Cloud Sync**
  - [ ] Sync watch history across devices
  - [ ] Sync configuration across devices
  - [ ] Support self-hosted sync server
  - [ ] Support encryption for cloud data

- [ ] **Collaborative Watching**
  - [ ] Implement watch party feature
  - [ ] Sync playback across multiple users
  - [ ] Add chat functionality
  - [ ] Support voice chat integration

- [ ] **Advanced Filtering**
  - [ ] Add genre filters
  - [ ] Add year filters
  - [ ] Add rating filters
  - [ ] Add status filters (airing, completed)
  - [ ] Add format filters (TV, Movie, OVA)
  - [ ] Add sort options (popularity, rating, trending)

- [ ] **AI-Powered Features**
  - [ ] AI-based anime recommendations
  - [ ] Automatic genre detection
  - [ ] Similar anime suggestions
  - [ ] Mood-based recommendations

- [ ] **Torrent Support**
  - [ ] Integrate torrent client (aria2 or transmission)
  - [ ] Search torrents from nyaa.si
  - [ ] Download high-quality releases
  - [ ] Auto-seed after watching

---

## Metrics & Observability

### Add Monitoring and Metrics

- [ ] **Application Metrics**
  - [ ] Create `metrics/metrics.go`
  - [ ] Track episodes watched
  - [ ] Track providers queried
  - [ ] Track AniList updates
  - [ ] Track playback errors
  - [ ] Track average watch time
  - [ ] Export metrics as Prometheus format (optional)

- [ ] **Performance Monitoring**
  - [ ] Add request duration tracking
  - [ ] Add API response time tracking
  - [ ] Add memory usage tracking
  - [ ] Add goroutine count tracking
  - [ ] Add profiling support (pprof)

- [ ] **Error Tracking**
  - [ ] Integrate error tracking (Sentry, optional)
  - [ ] Track error rates by type
  - [ ] Track provider failure rates
  - [ ] Add error dashboard

---

## CI/CD & DevOps

### Improve Development Workflow

- [ ] **GitHub Actions**
  - [ ] Add CI workflow for tests
  - [ ] Add CI workflow for linting (golangci-lint)
  - [ ] Add CI workflow for builds
  - [ ] Add CI workflow for releases
  - [ ] Add code coverage reporting (codecov)
  - [ ] Add dependency updates (dependabot)

- [ ] **Release Automation**
  - [ ] Improve GoReleaser configuration
  - [ ] Add changelog generation
  - [ ] Add version bump automation
  - [ ] Add release notes template
  - [ ] Add binary signing (GPG)

- [ ] **Development Tools**
  - [ ] Add Makefile for common tasks
  - [ ] Add pre-commit hooks
  - [ ] Add commit message linting
  - [ ] Add version management (git-chglog)

---

## Priority Matrix

### Critical Path (0.2.0)

**Must Fix Before Release:**
1. VLC/IINA player implementations
2. History file format bug
3. HTTP client timeouts
4. AllAnime JSON parsing
5. Input validation

**Should Have:**
1. Unit test infrastructure
2. Retry logic
3. Code deduplication
4. Configuration validation

### Next Release (0.3.0)

**Focus Areas:**
1. Testing (60%+ coverage)
2. Performance optimizations
3. Security improvements
4. Stub feature completion

**Nice to Have:**
1. New features (watchlist, notifications)
2. Documentation improvements
3. Theme customization

### Future Releases (0.4.0+)

**Exploration:**
1. Plugin system
2. Web interface
3. Cloud sync
4. Advanced filtering
5. AI-powered features

---

## Task Tracking

### How to Use This TODO

1. **Pick a task** from the appropriate priority level
2. **Create a feature branch**: `git checkout -b feature/task-name`
3. **Implement the task** with tests
4. **Update this TODO**: Mark task as complete with `[x]`
5. **Submit a PR** referencing this TODO item
6. **Get code review** and merge

### Task Status Legend

- [ ] Not started
- [x] Completed
- [~] In progress
- [!] Blocked

### Estimated Effort

- 🟢 Small (< 1 day)
- 🟡 Medium (1-3 days)
- 🔴 Large (> 3 days)

---

## Contributing

Want to contribute? Pick any task from this TODO and open a PR! Please reference the TODO item in your PR description.

For questions about any task, open a GitHub issue with the label `question`.

---

**Last Updated:** 2026-02-02
**Next Review:** Weekly
