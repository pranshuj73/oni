# ONI Codebase Review

**Project:** ONI - Terminal-based Anime Streaming Client
**Version:** 0.1.4
**Language:** Go 1.23.0
**License:** GNU GPL v3
**Review Date:** 2026-02-02
**Lines of Code:** ~8,864

---

## Executive Summary

ONI is a well-structured TUI application for anime streaming with AniList integration. The codebase demonstrates good separation of concerns, clean interface design, and comprehensive logging. However, it suffers from **lack of testing**, **incomplete player implementations**, **fragile string parsing**, and **data persistence bugs** that need immediate attention before production deployment.

**Overall Assessment:** 7/10

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Critical Issues](#critical-issues)
3. [Code Quality Analysis](#code-quality-analysis)
4. [Security Concerns](#security-concerns)
5. [Performance Considerations](#performance-considerations)
6. [Recommendations](#recommendations)

---

## Architecture Overview

### Project Structure

```
oni/
├── main.go (1,061 lines)          # Application orchestration & state machine
├── anilist/                        # AniList API integration
│   ├── client.go                   # GraphQL client
│   ├── types.go                    # Data models
│   ├── auth.go                     # OAuth authentication
│   └── queries.go                  # GraphQL queries
├── providers/                      # Streaming providers (5 implementations)
│   ├── allanime.go (17KB)         # Default provider
│   ├── aniwatch.go (6KB)
│   ├── yugen.go (4KB)
│   ├── hdrezka.go (11KB)          # Russian provider with decryption
│   ├── aniworld.go (6KB)          # German provider
│   ├── cache.go                    # Provider ID caching
│   └── provider.go                 # Interface definition
├── player/                         # Video player management
│   ├── player.go                   # Interface & factory
│   ├── mpv.go                      # Full implementation ✅
│   ├── vlc.go                      # Stub implementation ❌
│   ├── iina.go                     # Stub implementation ❌
│   └── history.go                  # Watch history tracking
├── ui/                             # Bubble Tea UI components (11 files)
│   ├── main_menu.go
│   ├── anime_list.go
│   ├── anime_search.go
│   ├── episode_select.go
│   ├── config_editor.go
│   ├── update_progress.go
│   ├── anilist_auth.go
│   ├── autoplay_prompt.go
│   ├── banner.go
│   ├── styles.go
│   └── keys.go
├── config/                         # Configuration management
│   └── config.go                   # INI-based config
├── logger/                         # Structured logging with rotation
│   └── logger.go
└── discord/                        # Discord Rich Presence
    └── presence.go
```

### Design Patterns

- **Factory Pattern**: `GetProvider()`, `GetPlayer()` - Extensible provider/player selection
- **Singleton Pattern**: Global logger with `sync.Once` initialization
- **Model-View Pattern**: Bubble Tea MVC architecture
- **Command Pattern**: Message-based state transitions
- **Interface-Based Design**: `Provider` and `Player` interfaces for extensibility

### State Machine

The application uses a 6-state state machine in `main.go`:

1. **MainMenu** - Primary navigation
2. **UpdateProgress** - AniList progress/score management
3. **EditConfig** - Built-in configuration editor
4. **AnimeList** - Tab-based anime list browsing
5. **EpisodeSelect** - Episode selection
6. **AniListAuth** - OAuth token authentication

---

## Critical Issues

### 1. VLC & IINA Player Implementations are Broken 🔴

**Severity:** CRITICAL
**Files:** `player/vlc.go:30-50`, `player/iina.go` (similar)

**Problem:**
The VLC and IINA player implementations always return hardcoded values indicating successful completion:

```go
// player/vlc.go:30-50
func (v *VLCPlayer) Play(ctx context.Context, episode *EpisodeInfo,
    videoData *VideoData, incognitoMode bool) (*PlaybackInfo, error) {
    // ... setup code ...

    return &PlaybackInfo{
        StoppedAt:           "00:00:00",      // Always 00:00:00!
        PercentageProgress:  100,              // Always 100%!
        CompletedSuccessful: true,             // Always true!
    }, nil
}
```

**Impact:**
- Resume functionality completely broken for VLC/IINA users
- Watch history tracking incorrect
- AniList progress updates wrong
- Users cannot resume from where they stopped
- Episode completion detection fails

**Root Cause:**
No mechanism implemented to extract playback position from VLC or IINA processes.

**Recommendation:**
Either properly implement position tracking or remove these player options until implemented.

---

### 2. History File Format Corruption Bug 🔴

**Severity:** CRITICAL
**Files:** `player/history.go:111-126`, `player/history.go:283`

**Problem 1: Tab-separated titles break parsing**

```go
// player/history.go:111-126
if len(parts) >= 6 {
    // New format with Duration
    duration = parts[3]
    lastWatched = parts[4]
    title = strings.Join(parts[5:], "\t")  // ❌ Tab-separated!
}
```

Anime titles containing tab characters will corrupt the history file format because tabs are used as field delimiters.

**Problem 2: Format mismatch in DeleteHistoryEntry**

```go
// player/history.go:283
line := fmt.Sprintf("%d\t%d/%d\t%s\t%s\n",  // Only 4 fields!
    e.MediaID, e.Progress, e.EpisodesTotal, e.Timestamp, e.Title)
```

But `SaveHistoryEntryWithIncognito` uses 6 fields:
```go
// player/history.go:184-186
fmt.Fprintf(file, "%d\t%d/%d\t%s\t%s\t%s\t%s\n",  // 6 fields!
    entry.MediaID, entry.Progress, entry.EpisodesTotal,
    entry.Position, entry.Duration, entry.LastWatched, entry.Title)
```

**Impact:**
- Data corruption when titles contain special characters
- History entries deleted with old format lose new fields
- Backward compatibility broken

**Recommendation:**
Switch to JSON or protobuf for history serialization. Current format is too fragile.

---

### 3. Fragile String Parsing in AllAnime Provider 🔴

**Severity:** HIGH
**File:** `providers/allanime.go:229-232`

**Problem:**

```go
// providers/allanime.go:229-232
jsonStr = strings.ReplaceAll(jsonStr, "{", "\n")
jsonStr = strings.ReplaceAll(jsonStr, "}", "\n")
jsonStr = strings.ReplaceAll(jsonStr, "\\u002F", "/")
jsonStr = strings.ReplaceAll(jsonStr, "\\", "")
```

This replaces ALL braces and backslashes in what should be JSON, corrupting the data structure.

**Impact:**
- Breaks if API response contains nested objects
- Corrupts valid JSON data
- Fails unpredictably when API format changes
- No error handling for malformed responses

**Recommendation:**
Use proper JSON unmarshaling with defined structs instead of string manipulation.

---

### 4. Missing HTTP Client Timeouts 🟡

**Severity:** HIGH
**Files:** `anilist/client.go:29`, `providers/allanime.go:30`, and all provider files

**Problem:**

```go
// anilist/client.go:29
client: &http.Client{},  // No timeout!
```

**Impact:**
- Requests can hang indefinitely
- No way to recover from network issues
- Application becomes unresponsive
- Resource leaks on stuck connections

**Recommendation:**
Add reasonable timeouts (30-60s) to all HTTP clients:

```go
client: &http.Client{
    Timeout: 60 * time.Second,
},
```

---

### 5. No Unit Tests 🟡

**Severity:** HIGH
**Coverage:** 0%

**Problem:**
Zero test files found in the entire codebase. No unit tests, integration tests, or end-to-end tests.

**Impact:**
- No regression protection
- Refactoring is risky
- Bug fixes might introduce new bugs
- No documentation of expected behavior
- Difficult to onboard new contributors

**Critical Test Gaps:**
1. Provider parsing logic (AllAnime, HDRezka regex parsing)
2. History file serialization/deserialization
3. Episode calculation logic
4. AniList API client error handling
5. Player factory logic
6. Configuration validation

**Recommendation:**
Add unit tests starting with:
- `player/history_test.go` - History serialization
- `providers/allanime_test.go` - JSON parsing
- `config/config_test.go` - Config validation

---

### 6. Hardcoded Discord Application ID 🟡

**Severity:** MEDIUM (Security)
**File:** `discord/presence.go:39`

**Problem:**

```go
const appID = "1436820992306450532"  // Hardcoded!
```

**Impact:**
- Public Discord app ID in source code
- Could be abused by other applications
- No way to use custom Discord app
- Best practice: use environment variables

**Recommendation:**
Make configurable via environment variable or config file:

```go
appID := os.Getenv("DISCORD_APP_ID")
if appID == "" {
    appID = "1436820992306450532"  // Default fallback
}
```

---

### 7. Race Condition in Incognito Mode 🟡

**Severity:** MEDIUM
**File:** `main.go:453`

**Problem:**

```go
// main.go:453
a.incognitoMode = a.mainMenu.GetIncognitoMode()
```

The incognito mode is fetched from the main menu at the time of continue watching selection. If the user toggles incognito mode between selection and playback, the state becomes inconsistent.

**Impact:**
- History might be saved when user expects incognito
- AniList might update when user expects incognito
- Race condition between UI state and playback state

**Recommendation:**
Capture incognito state at playback initialization and lock it for the duration.

---

### 8. Missing Input Validation 🟡

**Severity:** MEDIUM (Security)
**Files:** Multiple provider files

**Problem:**

```go
// providers/allanime.go:53
queryTitle := strings.ReplaceAll(title, " ", "+")
```

Anime titles are used directly in URLs without proper validation or encoding. Titles with special characters (`, `, `&`, `?`, etc.) can break URL parsing or cause injection.

**Example Attack Vector:**
```
Title: "Anime?id=malicious&inject=true"
URL: https://api.example.com/search?title=Anime?id=malicious&inject=true
```

**Recommendation:**
Use `url.QueryEscape()` for all user-controlled input in URLs:

```go
queryTitle := url.QueryEscape(title)
```

---

## Code Quality Analysis

### Strengths ✅

1. **Clean Architecture**
   - Clear separation of concerns
   - Well-organized package structure
   - Single responsibility principle followed
   - Minimal coupling between modules

2. **Comprehensive Logging**
   - Structured logging throughout
   - Log rotation (10MB, 5 backups, 30-day retention)
   - Caller info tracking
   - Field-based context logging
   - Location: `~/.oni/logs/oni.log`

3. **Good Error Handling Patterns**
   - Consistent error wrapping with context
   - Informative error messages
   - Error propagation through call stack
   - Example: `fmt.Errorf("failed to fetch episode: %w", err)`

4. **Interface-Based Design**
   - `Provider` interface enables provider extensibility
   - `Player` interface supports multiple video players
   - Easy to add new implementations

5. **Configuration Management**
   - INI-based config with sensible defaults
   - Built-in configuration editor in TUI
   - Clear config sections (player, provider, anilist, etc.)
   - Config location: `~/.oni/config.ini`

6. **Context Usage**
   - Proper use of `context.Context` for cancellation
   - HTTP requests support timeout/cancellation
   - Player processes can be interrupted

### Weaknesses ❌

1. **No Unit Tests (0% coverage)**
   - No test files found
   - No test infrastructure
   - No CI/CD validation
   - Risky for refactoring

2. **Fragile String Parsing**
   - Regex-based parsing instead of proper parsers
   - String replacement instead of JSON unmarshaling
   - No validation of parsed data
   - Examples: `providers/allanime.go:229`, `providers/hdrezka.go:55`

3. **Incomplete Implementations**
   - VLC player stub (always returns 100% complete)
   - IINA player stub (always returns 100% complete)
   - Image preview feature (config flag exists, no implementation)
   - JSON output feature (config flag exists, no implementation)
   - External menu feature (config flag exists, no implementation)

4. **Missing Input Validation**
   - No validation of quality values
   - No validation of provider names
   - No validation of player names
   - Anime titles not sanitized before URL use

5. **HTTP Client Issues**
   - No timeouts configured
   - No retry logic
   - No circuit breaker pattern
   - No rate limiting

6. **Code Duplication**
   - Episode 95% completion logic duplicated (main.go, main_menu.go)
   - Similar provider parsing code across providers
   - Error handling patterns could be centralized

7. **Hardcoded Values**
   - Discord application ID hardcoded
   - API URLs hardcoded in providers
   - Magic numbers (95% completion threshold)
   - No constants file

8. **Configuration Validation**
   - No validation at config load time
   - Invalid values silently accepted
   - Could cause runtime errors later

---

## Security Concerns

### 1. Token Storage

**File:** `anilist/auth.go:77`

**Current Implementation:**
```go
err = os.WriteFile(tokenPath, []byte(token), 0600)  // ✅ Restrictive permissions
```

**Assessment:** ✅ ACCEPTABLE
- Token file has `0600` permissions (user read-write only)
- Stored in plaintext (acceptable for OAuth tokens)
- Location: `~/.local/share/jerry/anilist_token.txt`

**Recommendation:** Consider using OS keychain for token storage in future.

---

### 2. Hardcoded Discord Application ID

**File:** `discord/presence.go:39`

**Problem:**
```go
const appID = "1436820992306450532"
```

**Risk:** Public Discord app ID could be abused by malicious applications.

**Recommendation:** Move to environment variable or config file.

---

### 3. URL Injection Risk

**Files:** Multiple provider files

**Problem:**
User-controlled anime titles inserted into URLs without proper escaping.

**Example:**
```go
// providers/allanime.go:53
queryTitle := strings.ReplaceAll(title, " ", "+")  // ❌ Insufficient
```

**Recommendation:** Use `url.QueryEscape()` for all user input in URLs.

---

### 4. Regex Injection

**File:** `providers/hdrezka.go:55`

**Problem:**
```go
re := regexp.MustCompile(`"title":\s*"([^"]*)"`)
```

Regex doesn't handle escaped quotes or newlines in titles, which could cause denial of service or incorrect parsing.

**Recommendation:** Use proper JSON parsing instead of regex on JSON data.

---

### 5. Command Injection (Mitigated)

**Files:** `player/mpv.go`, `player/vlc.go`

**Assessment:** ✅ SAFE
- Uses `exec.CommandContext()` with separate arguments (not shell)
- No shell expansion possible
- Arguments properly quoted

**Example:**
```go
cmd := exec.CommandContext(ctx, m.playerPath, args...)  // ✅ Safe
```

---

## Performance Considerations

### 1. Provider Caching ✅

**File:** `providers/cache.go`

**Implementation:**
- INI-based caching of provider IDs
- Reduces repeated API calls
- Caches anime ID to provider ID mapping
- Location: `~/.oni/providers_cache.ini`

**Assessment:** Good approach, significantly improves performance on repeated searches.

---

### 2. HTTP Request Performance

**Issues:**
- No connection pooling configuration
- No request timeout (requests can hang)
- No retry logic for transient failures
- No parallel provider queries

**Recommendations:**
1. Configure HTTP client with connection pooling
2. Add request timeouts (30-60s)
3. Implement retry with exponential backoff
4. Consider parallel provider queries with first-to-respond wins

---

### 3. Memory Usage

**Observations:**
- Anime lists loaded entirely into memory
- No pagination for large lists
- History file loaded entirely into memory

**Potential Issues:**
- Large anime lists (1000+ entries) could consume significant memory
- No streaming JSON parsing
- No lazy loading

**Recommendations:**
- Implement pagination for anime lists
- Use streaming JSON parser for large responses
- Lazy load history entries

---

### 4. Logging Performance

**File:** `logger/logger.go`

**Assessment:** ✅ GOOD
- Log rotation configured (10MB max, 5 backups)
- Prevents unbounded disk usage
- 30-day retention policy
- Async writes (via lumberjack)

---

## Feature Completeness

### Fully Implemented Features ✅

1. **TUI Interface**
   - Tab-based navigation
   - Search functionality
   - Episode selection
   - Configuration editor
   - Progress management

2. **AniList Integration**
   - OAuth authentication
   - Search anime
   - Fetch user lists
   - Update progress/score/status
   - User ID caching

3. **Multiple Providers**
   - AllAnime (default, fully functional)
   - AniWatch (fully functional)
   - Yugen (fully functional)
   - HDRezka (Russian, with decryption)
   - AniWorld (German)

4. **MPV Player Integration**
   - Full playback support
   - Resume functionality
   - Position tracking
   - Subtitle handling
   - Custom arguments support

5. **Watch History**
   - Position tracking
   - Resume from saved position
   - Last watched tracking
   - Duration tracking

6. **Discord Rich Presence**
   - Shows currently watching anime
   - Episode number display
   - Cover art integration
   - Graceful degradation if Discord not running

7. **Incognito Mode**
   - Runtime toggle
   - Prevents AniList updates
   - Prevents history saving

### Incomplete Features ❌

1. **VLC Player** (STUB)
   - Always returns 100% completion
   - No position tracking
   - Resume doesn't work

2. **IINA Player** (STUB)
   - Always returns 100% completion
   - No position tracking
   - Resume doesn't work

3. **Image Preview** (UNPARSED CONFIG)
   - Config flag: `image_preview`
   - No implementation found

4. **JSON Output** (UNPARSED CONFIG)
   - Config flag: `json_output`
   - No implementation found

5. **External Menu** (UNPARSED CONFIG)
   - Config flag: `use_external_menu`
   - No implementation found

6. **Download Feature** (PARTIAL)
   - Config: `download_dir`
   - Not used in current implementation
   - Likely planned for future

7. **Persist Incognito Sessions** (INCOMPLETE)
   - Config: `persist_incognito_sessions`
   - Note in code: "kept for compatibility but incognito mode is now runtime-only"

---

## Specific Code References for Improvement

### High Priority Fixes

#### 1. Fix VLC Player Implementation
**File:** `player/vlc.go:30-50`

**Current Code:**
```go
return &PlaybackInfo{
    StoppedAt:           "00:00:00",
    PercentageProgress:  100,
    CompletedSuccessful: true,
}, nil
```

**Needed:** Implement actual position tracking via VLC's RC interface or HTTP interface.

---

#### 2. Fix History File Serialization
**File:** `player/history.go:184-186`, `player/history.go:283`

**Problem:** Format mismatch between save and delete operations.

**Solution:** Migrate to JSON format:
```go
type HistoryEntry struct {
    MediaID        int       `json:"media_id"`
    Progress       int       `json:"progress"`
    EpisodesTotal  int       `json:"episodes_total"`
    Position       string    `json:"position"`
    Duration       string    `json:"duration"`
    LastWatched    time.Time `json:"last_watched"`
    Title          string    `json:"title"`
}
```

---

#### 3. Replace String Manipulation with JSON Parsing
**File:** `providers/allanime.go:229-232`

**Current Code:**
```go
jsonStr = strings.ReplaceAll(jsonStr, "{", "\n")
jsonStr = strings.ReplaceAll(jsonStr, "}", "\n")
```

**Solution:** Define proper structs and use `json.Unmarshal()`:
```go
type AllAnimeResponse struct {
    Data struct {
        Shows struct {
            Edges []struct {
                ID    string `json:"_id"`
                Title string `json:"name"`
            } `json:"edges"`
        } `json:"shows"`
    } `json:"data"`
}
```

---

#### 4. Add HTTP Client Timeouts
**File:** `anilist/client.go:29`

**Current Code:**
```go
client: &http.Client{},
```

**Solution:**
```go
client: &http.Client{
    Timeout: 60 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
},
```

---

#### 5. Validate Configuration
**File:** `config/config.go:100-120`

**Add Validation Method:**
```go
func (c *Config) Validate() error {
    validPlayers := []string{"mpv", "vlc", "iina"}
    if !contains(validPlayers, c.PlayerConfig.Player) {
        return fmt.Errorf("invalid player: %s", c.PlayerConfig.Player)
    }

    validProviders := []string{"allanime", "aniwatch", "yugen", "hdrezka", "aniworld"}
    if !contains(validProviders, c.ProviderConfig.Provider) {
        return fmt.Errorf("invalid provider: %s", c.ProviderConfig.Provider)
    }

    validQualities := []string{"1080", "720", "480", "360"}
    if !contains(validQualities, c.ProviderConfig.Quality) {
        return fmt.Errorf("invalid quality: %s", c.ProviderConfig.Quality)
    }

    return nil
}
```

---

#### 6. Fix URL Encoding
**File:** `providers/allanime.go:53`

**Current Code:**
```go
queryTitle := strings.ReplaceAll(title, " ", "+")
```

**Solution:**
```go
import "net/url"

queryTitle := url.QueryEscape(title)
```

Apply to all provider files: `aniwatch.go`, `yugen.go`, `hdrezka.go`, `aniworld.go`.

---

### Medium Priority Improvements

#### 7. Deduplicate Episode Completion Logic
**Files:** `main.go:512-535`, `ui/main_menu.go` (similar logic)

**Current:** 95% completion threshold duplicated.

**Solution:** Create shared function:
```go
// utils/episode.go
const CompletionThreshold = 95.0

func IsEpisodeComplete(percentageProgress float64) bool {
    return percentageProgress >= CompletionThreshold
}

func GetNextEpisode(currentEpisode, totalEpisodes int, percentageProgress float64) int {
    if IsEpisodeComplete(percentageProgress) && currentEpisode < totalEpisodes {
        return currentEpisode + 1
    }
    return currentEpisode
}
```

---

#### 8. Add Retry Logic to Providers
**File:** Create `providers/retry.go`

**Implementation:**
```go
func WithRetry(ctx context.Context, maxRetries int, fn func() error) error {
    var err error
    for i := 0; i < maxRetries; i++ {
        err = fn()
        if err == nil {
            return nil
        }

        // Exponential backoff
        backoff := time.Duration(math.Pow(2, float64(i))) * time.Second
        select {
        case <-time.After(backoff):
            continue
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    return fmt.Errorf("max retries exceeded: %w", err)
}
```

---

#### 9. Remove Unused Config Flags
**File:** `config/config.go`

**Options:**
1. Implement the features (`image_preview`, `json_output`, `use_external_menu`)
2. Remove the flags entirely
3. Add deprecation warnings

**Recommendation:** Remove flags unless features will be implemented soon.

---

#### 10. Make Discord App ID Configurable
**File:** `discord/presence.go:39`

**Current:**
```go
const appID = "1436820992306450532"
```

**Solution:**
```go
func getDiscordAppID() string {
    if id := os.Getenv("ONI_DISCORD_APP_ID"); id != "" {
        return id
    }
    return "1436820992306450532" // Default
}
```

---

### Low Priority Enhancements

#### 11. Add Metrics/Observability
Create `metrics/metrics.go`:
```go
type Metrics struct {
    EpisodesWatched    int64
    ProvidersQueried   map[string]int64
    AniListUpdates     int64
    PlaybackErrors     int64
    AverageWatchTime   time.Duration
}
```

---

#### 12. Implement Connection Pooling
**File:** `anilist/client.go`, all provider files

**Add:**
```go
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,
    TLSHandshakeTimeout: 10 * time.Second,
}

client: &http.Client{
    Timeout:   60 * time.Second,
    Transport: transport,
},
```

---

#### 13. Add Provider Response Caching
Cache provider responses for 5-10 minutes to reduce API calls:
```go
type CachedResponse struct {
    Data      interface{}
    ExpiresAt time.Time
}

var responseCache = sync.Map{}
```

---

#### 14. Implement Parallel Provider Queries
Query multiple providers in parallel and use first successful response:
```go
func GetEpisodeWithFallback(ctx context.Context, providers []Provider,
    mediaID, episode string) (*EpisodeInfo, error) {

    results := make(chan *EpisodeInfo, len(providers))
    errors := make(chan error, len(providers))

    for _, p := range providers {
        go func(provider Provider) {
            info, err := provider.GetEpisodeInfo(ctx, mediaID, episode, "")
            if err != nil {
                errors <- err
                return
            }
            results <- info
        }(p)
    }

    // Return first successful result
    select {
    case result := <-results:
        return result, nil
    case <-time.After(30 * time.Second):
        return nil, fmt.Errorf("timeout waiting for providers")
    }
}
```

---

## Recommendations

### Immediate Actions (Before v0.2.0)

1. **Fix VLC/IINA player implementations** or remove them
   - Either implement proper position tracking
   - Or remove from supported players list
   - Priority: CRITICAL

2. **Migrate history to JSON format**
   - Replace tab-separated format with JSON
   - Add migration code for existing history
   - Priority: CRITICAL

3. **Add HTTP client timeouts**
   - Set 60-second timeout on all HTTP clients
   - Add context cancellation support
   - Priority: HIGH

4. **Fix AllAnime JSON parsing**
   - Replace string manipulation with proper JSON unmarshaling
   - Add error handling for malformed responses
   - Priority: HIGH

5. **Add input validation**
   - Validate configuration values
   - Sanitize anime titles for URL use
   - Add bounds checking
   - Priority: HIGH

### Short-term Improvements (v0.2.x)

1. **Add unit tests**
   - Start with critical paths (history, providers)
   - Aim for 60%+ coverage
   - Priority: HIGH

2. **Implement retry logic**
   - Add exponential backoff for provider calls
   - Implement circuit breaker for repeated failures
   - Priority: MEDIUM

3. **Deduplicate code**
   - Extract episode completion logic to shared util
   - Create common provider parsing utilities
   - Priority: MEDIUM

4. **Remove or implement stub features**
   - Image preview, JSON output, external menu
   - Either implement or remove config flags
   - Priority: MEDIUM

5. **Make Discord app ID configurable**
   - Move to environment variable
   - Allow user to specify custom app ID
   - Priority: LOW

### Long-term Enhancements (v0.3.0+)

1. **Add integration tests**
   - Test against real provider APIs (with mocking)
   - End-to-end TUI tests
   - Priority: MEDIUM

2. **Implement metrics/observability**
   - Track usage statistics
   - Monitor error rates
   - Measure performance
   - Priority: LOW

3. **Add performance optimizations**
   - Parallel provider queries
   - Response caching
   - Lazy loading for large lists
   - Priority: LOW

4. **Improve security**
   - Use OS keychain for token storage
   - Add rate limiting
   - Implement request signing
   - Priority: LOW

5. **Documentation**
   - API documentation (GoDoc)
   - Architecture documentation
   - Contributing guide
   - Priority: LOW

---

## Testing Strategy

### Unit Tests (Priority: HIGH)

**Coverage Target:** 60%+

**Critical Test Files:**

1. `player/history_test.go`
   - Test serialization/deserialization
   - Test format migration
   - Test concurrent access

2. `providers/allanime_test.go`
   - Test JSON parsing
   - Test error handling
   - Mock HTTP responses

3. `providers/hdrezka_test.go`
   - Test regex parsing
   - Test decryption logic

4. `config/config_test.go`
   - Test validation
   - Test default values
   - Test INI parsing

5. `anilist/client_test.go`
   - Test GraphQL query building
   - Test authentication flow
   - Mock API responses

### Integration Tests (Priority: MEDIUM)

1. **Provider Integration Tests**
   - Test against real APIs (with rate limiting)
   - Use recorded responses for CI
   - Test fallback logic

2. **Player Integration Tests**
   - Test MPV integration
   - Mock player processes
   - Test position tracking

### End-to-End Tests (Priority: LOW)

1. **TUI Tests**
   - Test navigation flows
   - Test state transitions
   - Use Bubble Tea testing utilities

---

## Conclusion

ONI is a well-architected application with clean separation of concerns and good interface design. The codebase demonstrates solid engineering practices in logging, error handling, and configuration management. However, it suffers from critical bugs in player implementations and history management that must be addressed before production use.

The lack of unit tests is the most significant quality issue. Combined with fragile string parsing and missing input validation, this creates a high-risk environment for refactoring and feature additions.

**Recommended Action Plan:**
1. Fix critical bugs (VLC/IINA, history format)
2. Add HTTP timeouts
3. Establish testing infrastructure
4. Validate all inputs
5. Then proceed with feature additions

With these improvements, ONI will be a robust, production-ready application ready for wider adoption.

---

**Reviewer Notes:**
- Codebase analyzed: ~8,864 lines of Go
- Time spent: Comprehensive static analysis
- Files reviewed: All 40+ source files
- Testing: Manual code review, no runtime analysis
