# ONI Documentation

Welcome to the ONI documentation directory. This folder contains comprehensive analysis, code review, and development planning documents.

## 📚 Document Index

### [SUMMARY.md](./SUMMARY.md) - Start Here ⭐
**Executive summary of the codebase analysis**
- TL;DR of critical issues
- Quick metrics and assessment
- Recommended action plan
- Perfect for: Project overview, new contributors, stakeholders

### [CODE_REVIEW.md](./CODE_REVIEW.md)
**Comprehensive technical code review**
- Detailed architecture analysis
- Critical issues with specific code references
- Security and performance assessment
- Code quality analysis by module
- Specific file:line references for all issues
- Perfect for: Developers, code reviewers, bug fixing

### [TODO.md](./TODO.md)
**Complete development task list**
- Organized by priority (P0-P3)
- Categorized by type (bugs, refactoring, features, testing)
- Estimated effort levels
- Includes both critical fixes and long-term vision
- Perfect for: Project planning, picking tasks, tracking progress

## 🎯 Quick Navigation

### I want to...

- **Understand what's wrong with the code** → [SUMMARY.md](./SUMMARY.md) (Critical Issues section)
- **Fix a specific bug** → [CODE_REVIEW.md](./CODE_REVIEW.md) (Code Quality Analysis section)
- **Pick a task to work on** → [TODO.md](./TODO.md) (Start with Critical Bug Fixes)
- **Understand the architecture** → [CODE_REVIEW.md](./CODE_REVIEW.md) (Architecture Overview section)
- **See the roadmap** → [TODO.md](./TODO.md) (Long-term Vision section)
- **Get metrics/assessment** → [SUMMARY.md](./SUMMARY.md) (Code Quality Metrics section)

## 🔴 Critical Issues at a Glance

If you only have 5 minutes, read this:

1. **VLC/IINA players are broken** - Always return 100% complete, resume doesn't work
   - Files: `player/vlc.go:30-50`, `player/iina.go`

2. **History file format bug** - Corrupts data with special characters
   - Files: `player/history.go:111-126`, `player/history.go:283`

3. **Fragile JSON parsing** - String replacement instead of proper parsing
   - Files: `providers/allanime.go:229-232`

4. **No HTTP timeouts** - Application can hang indefinitely
   - Files: `anilist/client.go:29`, all `providers/*.go`

5. **Zero unit tests** - 0% code coverage
   - Impact: No regression protection

**Action:** See [TODO.md](./TODO.md) Critical Bug Fixes section

## 📊 Codebase Stats

- **Total Lines:** ~8,864 lines of Go
- **Packages:** 7 main packages
- **Files:** 40+ source files
- **Code Quality:** 7/10
- **Test Coverage:** 0%
- **Critical Bugs:** 5
- **Security Issues:** 4

## 🛠️ For Contributors

1. **Start with:** [SUMMARY.md](./SUMMARY.md) to understand the project state
2. **Pick a task:** [TODO.md](./TODO.md) has tasks organized by priority
3. **Understand the code:** [CODE_REVIEW.md](./CODE_REVIEW.md) has detailed analysis
4. **Make changes:** Reference specific line numbers from CODE_REVIEW.md
5. **Submit PR:** Reference TODO.md task in your PR description

## 📋 Document Status

| Document | Status | Last Updated |
|----------|--------|--------------|
| SUMMARY.md | ✅ Complete | 2026-02-02 |
| CODE_REVIEW.md | ✅ Complete | 2026-02-02 |
| TODO.md | ✅ Complete | 2026-02-02 |
| ARCHITECTURE.md | ❌ Planned | - |
| CONTRIBUTING.md | ❌ Planned | - |
| USER_GUIDE.md | ❌ Planned | - |

## 🎯 Recommended Reading Order

### For Project Managers / Stakeholders:
1. SUMMARY.md (Executive Summary, Recommended Action Plan)
2. TODO.md (Priority Matrix, Timeline)

### For Developers:
1. SUMMARY.md (Critical Issues, Architecture Overview)
2. CODE_REVIEW.md (Full analysis)
3. TODO.md (Pick a task)

### For New Contributors:
1. SUMMARY.md (Quick overview)
2. TODO.md (Find good first issues)
3. CODE_REVIEW.md (Deep dive as needed)

### For Code Reviewers:
1. CODE_REVIEW.md (Specific Code References section)
2. SUMMARY.md (Context and assessment)

## 🚀 Next Steps

Based on this analysis, the recommended next steps are:

1. **Immediate (Week 1-2):** Fix critical bugs
   - VLC/IINA implementations
   - History format migration
   - HTTP timeouts
   - JSON parsing fixes

2. **Short-term (Week 3-4):** Add testing
   - Setup test infrastructure
   - Unit tests for critical paths
   - Target 60%+ coverage

3. **Medium-term (Month 2):** Refactor & optimize
   - Code deduplication
   - Input validation
   - Retry logic
   - Security improvements

4. **Long-term (Month 3+):** New features
   - Complete stub features
   - Add enhancements
   - Improve documentation

## 📞 Questions?

If you have questions about this analysis:
- Open a GitHub issue with label `documentation`
- Reference specific sections or line numbers
- Suggest improvements to these docs

## 📝 License

These documentation files are part of the ONI project and follow the same license (GNU GPL v3).

---

**Generated:** 2026-02-02
**Analyzer:** Automated Code Review System
**Version:** 0.1.4
