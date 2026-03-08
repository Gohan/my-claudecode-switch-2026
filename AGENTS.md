# AGENTS.md

Project-specific guide for AI agents working in this codebase.

## Project Overview

**claude-switch** is a terminal UI (TUI) application for managing and switching Claude Code settings profiles. It allows users to save, preview, apply, and delete different configuration profiles for Claude Code's `settings.json`.

## Essential Commands

### Setup
```bash
mise install          # Install Go via mise
```

### Development
```bash
go run .              # Run the application directly
go build -o dist/claude-switch.exe .   # Build Windows executable
go build -o dist/claude-switch .       # Build Unix executable
```

### Cross-platform Build (requires bash)
```bash
./build.sh windows amd64    # Build for Windows x64
./build.sh linux arm64      # Build for Linux ARM64
./build.sh darwin arm64     # Build for macOS ARM
./build.sh all              # Build for all platforms
```

### Dependencies
```bash
go mod tidy           # Clean up dependencies
go mod download       # Download dependencies
```

## Project Structure

```
claude-switch/
├── main.go                 # Entry point, initializes Bubble Tea program
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
├── mise.toml               # Tool version management (go = "latest")
├── build.sh                # Cross-platform build script
├── internal/
│   ├── tui/
│   │   ├── model.go        # TUI state machine and view rendering
│   │   └── styles.go       # Lipgloss style definitions
│   └── profile/
│       └── profile.go      # Profile CRUD, settings.json management
```

## Architecture

### TUI Framework
- Uses [Bubble Tea](https://github.com/charmbracelet/bubbletea) (Elm-style architecture)
- [Lipgloss](https://github.com/charmbracelet/lipgloss) for styling
- [Bubbles](https://github.com/charmbracelet/bubbles) for text input component

### State Machine (model.go)
The TUI has 5 view states:
- `viewList` - Main profile list
- `viewPreview` - Diff preview before applying
- `viewSave` - Save current settings as new profile
- `viewConfirmApply` - Confirmation dialog for apply
- `viewConfirmDelete` - Confirmation dialog for delete

### Profile Storage
- **Current settings**: `~/.claude/settings.json`
- **Saved profiles**: `~/.claude-switch/profiles/<name>.json`

## Code Patterns

### Error Handling
- Functions return errors, callers display via `m.message` field
- Non-critical errors (like missing profiles dir) are silently handled

### Settings Format
Settings are stored as `map[string]interface{}` to handle nested JSON:
```json
{
  "model": "claude-sonnet-4-20250514",
  "env": {
    "ANTHROPIC_BASE_URL": "https://api.anthropic.com"
  }
}
```

### Sensitive Data
The `MaskSensitive()` function in `profile.go` masks values for keys containing: `token`, `key`, `secret`, `password`, `credential`

## Key Bindings (TUI)

| Key | Action |
|-----|--------|
| `j`/`down` | Move cursor down |
| `k`/`up` | Move cursor up |
| `enter` | Apply selected profile |
| `p` | Preview profile diff |
| `s` | Save current settings |
| `d` | Delete selected profile |
| `q` | Quit |
| `ctrl+c` | Force quit |

## Notes

- Go version: 1.25.0 (specified in go.mod)
- No test files present in this project
- Build script uses `CGO_ENABLED=0` for static binaries
- Uses alternate screen buffer (`tea.WithAltScreen()`) for clean TUI experience
