# ONI Codebase Analysis - Executive Summary

**Date:** 2026-02-02
**Version:** 0.1.4
**Analyst:** Automated Code Review System

---

## Quick Links

- [Full Code Review](./CODE_REVIEW.md) - Comprehensive technical analysis with specific code references
- [Development TODO](./TODO.md) - Complete task list organized by priority
- [Main README](../README.md) - User-facing documentation

---

## TL;DR

ONI is a **well-architected** TUI anime streaming application with clean separation of concerns and good interface design. However, it has **critical bugs** in player implementations and data persistence that must be fixed before production use. The codebase lacks unit tests (0% coverage), which is the most significant quality risk.

**Overall Assessment:** 7/10 - Good foundation, needs critical bug fixes and testing.

---

## Critical Issues Requiring Immediate Attention

### 🔴 **1. VLC/IINA Players Are Broken**
**Impact:** Resume functionality completely broken for non-MPV users

The VLC and IINA player implementations always return hardcoded values:
- Position: Always `00:00:00`
- Progress: Always `100%`
- Status: Always `CompletedSuccessful: true`

**Location:** `player/vlc.go:30-50`, `player/iina.go`

**Fix:** Either implement proper position tracking or remove these players from the supported list.

---

### 🔴 **2. History File Format Corruption**
**Impact:** Data loss when anime titles contain tabs; delete operation uses wrong format

The tab-separated history format breaks when titles contain tab characters. Additionally, there's a format mismatch between save (6 fields) and delete (4 fields) operations.

**Location:** `player/history.go:111-126`, `player/history.go:283`

**Fix:** Migrate to JSON format with proper serialization.

---

### 🔴 **3. Fragile JSON Parsing in AllAnime**
**Impact:** Breaks unpredictably on API changes; corrupts valid JSON

The AllAnime provider uses string replacement instead of proper JSON parsing:
```go
jsonStr = strings.ReplaceAll(jsonStr, "{", "\n")
jsonStr = strings.ReplaceAll(jsonStr, "}", "\n")
```

**Location:** `providers/allanime.go:229-232`

**Fix:** Define proper structs and use `json.Unmarshal()`.

---

### 🟡 **4. No HTTP Timeouts**
**Impact:** Application can hang indefinitely on network issues

All HTTP clients are created without timeouts:
```go
client: &http.Client{},  // No timeout!
```

**Location:** `anilist/client.go:29`, all `providers/*.go`

**Fix:** Add 60-second timeouts to all HTTP clients.

---

### 🟡 **5. Zero Unit Tests**
**Impact:** No regression protection, risky refactoring, difficult maintenance

**Coverage:** 0%

**Fix:** Establish testing infrastructure, prioritize tests for:
- History serialization (`player/history_test.go`)
- Provider parsing (`providers/*_test.go`)
- Configuration validation (`config/config_test.go`)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        main.go (1,061 lines)                 │
│                    Application State Machine                 │
│   States: MainMenu │ AnimeList │ EpisodeSelect │ Auth etc.  │
└────────────┬────────────────────────────────────────────────┘
             │
    ┌────────┼──────────┬──────────┬──────────┬───────────┐
    │        │          │          │          │           │
┌───▼───┐ ┌─▼──────┐ ┌─▼──────┐ ┌─▼──────┐ ┌─▼──────┐ ┌──▼─────┐
│AniList│ │Provider│ │ Player │ │  UI    │ │ Config │ │Discord │
│ (API) │ │ (5x)   │ │ (3x)   │ │ (11x)  │ │ (INI)  │ │  (RPC) │
└───────┘ └────────┘ └────────┘ └────────┘ └────────┘ └────────┘
```

### Package Breakdown

| Package | Files | Purpose | Status |
|---------|-------|---------|--------|
| `main` | 1 | Application orchestration, state machine | ✅ Good |
| `anilist/` | 4 | AniList API integration (GraphQL) | ✅ Good |
| `providers/` | 7 | 5 streaming providers + cache | ⚠️ Needs fixes |
| `player/` | 4 | MPV (✅), VLC (❌), IINA (❌) + history | ❌ Critical bugs |
| `ui/` | 11 | Bubble Tea components | ✅ Good |
| `config/` | 1 | INI-based configuration | ⚠️ Needs validation |
| `logger/` | 1 | Structured logging with rotation | ✅ Good |
| `discord/` | 1 | Rich Presence integration | ⚠️ Hardcoded ID |

---

## Code Quality Metrics

| Metric | Score | Target | Status |
|--------|-------|--------|--------|
| Architecture | 8/10 | - | ✅ Excellent |
| Code Organization | 8/10 | - | ✅ Excellent |
| Error Handling | 7/10 | - | ✅ Good |
| Logging | 9/10 | - | ✅ Excellent |
| Testing | 0/10 | 60% coverage | ❌ Critical |
| Documentation | 5/10 | - | ⚠️ Needs improvement |
| Security | 6/10 | - | ⚠️ Needs improvement |
| Performance | 7/10 | - | ✅ Good |

**Overall:** 7/10

---

## What Works Well ✅

1. **Clean Architecture**
   - Clear separation of concerns
   - Well-organized package structure
   - Interface-based design (Provider, Player)
   - Minimal coupling

2. **Comprehensive Logging**
   - Structured logging with caller info
   - Log rotation (10MB, 5 backups, 30-day retention)
   - Field-based context logging

3. **Good Error Handling**
   - Consistent error wrapping with context
   - Informative error messages
   - Proper error propagation

4. **Configuration Management**
   - INI-based config with sensible defaults
   - Built-in configuration editor
   - Multiple configuration sections

5. **Feature Completeness**
   - TUI with tab navigation
   - 5 streaming providers
   - AniList integration
   - MPV player support
   - Watch history with resume
   - Discord Rich Presence

---

## What Needs Work ❌

1. **No Unit Tests (0% coverage)**
   - Zero test files in entire codebase
   - No regression protection
   - Risky for refactoring

2. **Critical Bugs**
   - VLC/IINA players return fake data
   - History file format corruption
   - Fragile string parsing

3. **Missing Input Validation**
   - No config validation at load time
   - No URL encoding for user input
   - No bounds checking

4. **HTTP Client Issues**
   - No timeouts (can hang forever)
   - No retry logic
   - No circuit breaker

5. **Incomplete Features**
   - Image preview (config exists, no implementation)
   - JSON output (config exists, no implementation)
   - External menu (config exists, no implementation)

6. **Security Concerns**
   - Hardcoded Discord app ID
   - URL injection risk
   - Tokens stored in plaintext (acceptable for OAuth, but could use keychain)

---

## Recommended Action Plan

### Phase 1: Critical Bug Fixes (v0.1.5)

**Timeline:** 1-2 weeks

1. ✅ Fix VLC/IINA implementations OR remove them
2. ✅ Migrate history to JSON format
3. ✅ Add HTTP client timeouts (60s)
4. ✅ Fix AllAnime JSON parsing
5. ✅ Add input validation (config, URLs)

### Phase 2: Testing Infrastructure (v0.2.0)

**Timeline:** 2-3 weeks

1. Setup testing framework
2. Add unit tests for critical paths:
   - History serialization
   - Provider parsing
   - Config validation
3. Target: 60%+ code coverage
4. Setup CI/CD for tests

### Phase 3: Refactoring & Optimization (v0.2.x)

**Timeline:** 2-4 weeks

1. Deduplicate episode completion logic
2. Extract common provider utilities
3. Implement retry logic with exponential backoff
4. Add connection pooling
5. Implement response caching

### Phase 4: Feature Completion (v0.3.0)

**Timeline:** 4-6 weeks

1. Complete stub features (image preview, JSON output, external menu)
2. Add new features (watchlist, notifications, statistics)
3. Improve documentation
4. Add theme customization

---

## File-Specific Issues

### High Priority

| File | Issue | Severity | Line | Fix |
|------|-------|----------|------|-----|
| `player/vlc.go` | Always returns 100% complete | CRITICAL | 30-50 | Implement position tracking |
| `player/iina.go` | Always returns 100% complete | CRITICAL | Similar | Implement position tracking |
| `player/history.go` | Tab-separated format breaks | CRITICAL | 111-126 | Migrate to JSON |
| `player/history.go` | Format mismatch in delete | CRITICAL | 283 | Use consistent format |
| `providers/allanime.go` | String replacement on JSON | HIGH | 229-232 | Use json.Unmarshal |
| `anilist/client.go` | No HTTP timeout | HIGH | 29 | Add 60s timeout |
| `providers/*.go` | No HTTP timeout | HIGH | Various | Add 60s timeout |
| `config/config.go` | No validation | HIGH | - | Add Validate() method |

### Medium Priority

| File | Issue | Severity | Fix |
|------|-------|----------|-----|
| `discord/presence.go` | Hardcoded app ID | MEDIUM | Make configurable |
| `main.go` | Incognito race condition | MEDIUM | Lock state during playback |
| `providers/hdrezka.go` | Fragile regex parsing | MEDIUM | Use JSON parsing |
| `main.go` | Duplicate episode logic | MEDIUM | Extract to utils |

---

## Security Assessment

### ✅ Acceptable

- Token file permissions: `0600` (user-only)
- No shell command injection (uses `exec.CommandContext`)
- No SQL injection (no database)
- No unsafe code

### ⚠️ Needs Improvement

- Hardcoded Discord app ID (public)
- URL injection risk (titles not escaped)
- Regex injection in HDRezka provider
- Tokens stored in plaintext (consider keychain)

### 🔒 Recommendations

1. Make Discord app ID configurable
2. Use `url.QueryEscape()` for all user input
3. Replace regex with proper JSON parsing
4. Consider OS keychain for token storage (future)

---

## Performance Assessment

### ✅ Good

- Provider caching (INI-based)
- Log rotation configured
- Context-based cancellation
- Clean state management

### ⚠️ Could Improve

- No connection pooling
- No request timeouts
- No retry logic
- No parallel provider queries
- No response caching
- Entire lists loaded into memory

### 🚀 Optimization Opportunities

1. Add HTTP connection pooling
2. Implement retry with exponential backoff
3. Add response caching (5-10 min TTL)
4. Implement parallel provider queries
5. Add lazy loading for large lists

---

## Testing Strategy

### Phase 1: Unit Tests (Priority: HIGH)

**Target Coverage:** 60%+

**Critical Test Files:**
1. `player/history_test.go` - Serialization, concurrent access
2. `providers/allanime_test.go` - JSON parsing, error handling
3. `config/config_test.go` - Validation, defaults
4. `anilist/client_test.go` - GraphQL queries, auth

### Phase 2: Integration Tests

1. Provider integration with real APIs (rate-limited, recorded)
2. Player integration with test videos
3. End-to-end TUI flows

### Phase 3: CI/CD

1. GitHub Actions for tests
2. Code coverage reporting (codecov)
3. Linting (golangci-lint)
4. Automated releases (GoReleaser)

---

## Feature Roadmap

### v0.1.5 (Critical Fixes)
- Fix VLC/IINA or remove them
- Fix history format bug
- Add HTTP timeouts
- Fix JSON parsing
- Add input validation

### v0.2.0 (Testing & Refactoring)
- Unit test infrastructure (60%+ coverage)
- Retry logic
- Code deduplication
- Configuration validation
- Security improvements

### v0.3.0 (Feature Completion)
- Complete stub features (image preview, JSON output)
- Watchlist management
- Episode notifications
- Watch statistics
- Theme customization

### v0.4.0+ (Advanced Features)
- Multi-language support (i18n)
- MAL (MyAnimeList) integration
- Download feature
- Keyboard shortcut customization
- Plugin system

---

## Developer Onboarding

### Quick Start for Contributors

1. **Clone and build:**
   ```bash
   git clone https://github.com/pranshuj73/oni.git
   cd oni
   go build
   ```

2. **Pick a task from [TODO.md](./TODO.md)**

3. **Create a feature branch:**
   ```bash
   git checkout -b feature/task-name
   ```

4. **Make changes with tests:**
   - Write unit tests for new code
   - Ensure existing functionality works
   - Follow existing code style

5. **Submit PR:**
   - Reference TODO item
   - Include tests
   - Update documentation

### Code Style Guidelines

- Use `gofmt` for formatting
- Follow Go best practices
- Add GoDoc comments for exported functions
- Use meaningful variable names
- Keep functions small and focused
- Add error context with `fmt.Errorf("context: %w", err)`

---

## Documentation Inventory

### Existing Documentation

- ✅ `README.md` - User-facing documentation
- ✅ `docs/CODE_REVIEW.md` - Technical code review
- ✅ `docs/TODO.md` - Development task list
- ✅ `docs/SUMMARY.md` - This document

### Missing Documentation

- ❌ `CONTRIBUTING.md` - Contributing guidelines
- ❌ `docs/ARCHITECTURE.md` - Architecture diagrams
- ❌ `docs/USER_GUIDE.md` - Comprehensive user guide
- ❌ `docs/PROVIDERS.md` - Provider documentation
- ❌ `docs/CONFIGURATION.md` - Config reference
- ❌ GoDoc comments (incomplete)

---

## Metrics to Track

### Code Quality
- [ ] Code coverage: 0% → 60%+
- [ ] Test count: 0 → 100+
- [ ] Critical bugs: 5 → 0
- [ ] Security issues: 4 → 0

### Performance
- [ ] HTTP request timeout: ∞ → 60s
- [ ] Provider response time: Not measured
- [ ] Memory usage: Not measured
- [ ] Goroutine count: Not measured

### Features
- [ ] Working players: 1/3 → 3/3
- [ ] Implemented config flags: 60% → 100%
- [ ] Provider success rate: Not measured
- [ ] User satisfaction: Not measured

---

## Conclusion

ONI is a **promising project** with a solid architectural foundation and comprehensive feature set. The codebase demonstrates good engineering practices in logging, error handling, and separation of concerns.

However, **critical bugs** in player implementations and data persistence, combined with **zero test coverage**, make the current codebase **risky for production use**. The lack of input validation and HTTP timeouts adds additional risk.

**The recommended path forward:**

1. **Fix critical bugs immediately** (VLC/IINA, history format, HTTP timeouts)
2. **Establish testing infrastructure** to prevent regressions
3. **Add input validation** to improve security and robustness
4. **Then** proceed with feature additions and optimizations

With these improvements, ONI will be a **robust, production-ready** anime streaming client with a strong foundation for future growth.

---

**Assessment:** 7/10 - Good foundation, needs critical fixes and testing

**Recommendation:** Fix critical bugs before v0.2.0 release

**Timeline:** 1-2 weeks for critical fixes, 2-3 weeks for testing infrastructure

---

## Contact & Support

For questions about this analysis:
- Open a GitHub issue with label `code-review`
- Reference specific sections from CODE_REVIEW.md
- Refer to TODO.md for task details

Happy coding! 🎌
