# Changelog

All notable changes to this custom AgentAPI fork will be documented in this file.

## [v1.0.1-ccs] - 2025-08-05

### Fixed
- **Permission Mode Detection**: Fixed "Bypassing Permissions" detection
  - Now correctly detects when mode appears at the end of status line
  - Increased search range from 5 to 10 lines for better reliability
  - Uses `strings.Contains` instead of regex for more robust matching

### Changed
- Improved `ExtractPermissionMode` function to handle different line formats
- Status line format support: `? for shortcuts                    Bypassing Permissions`

## [v1.0.0-ccs] - 2025-08-05

### Added
- **Permission Mode API Support**: New field `permissionMode` in API responses
  - Available in both `/status` and `/messages` endpoints
  - Real-time extraction from terminal screen
  - Supports detection of:
    - "Bypassing Permissions"
    - "plan mode on"
    - "auto-accept edits on"
    - "Normal mode"
- **New function**: `ExtractPermissionMode` in `lib/msgfmt/permission_mode.go`
- **Build script**: `build-release.sh` for easy cross-platform builds

### Changed
- **Message Box Parser**: Modified `removeMessageBox` to preserve status lines
  - Only removes the 3-line message box (line above >, line with >, line below >)
  - Keeps all content below the message box, including permission mode status
- **API Models**: Updated to include `permissionMode` field
  - `StatusResponse` struct enhanced
  - `MessagesResponse` struct enhanced

### Removed
- **Web UI**: Completely removed chat interface
  - Deleted `chat/` directory
  - Removed embedded static files
  - Simplified `FileServerWithIndexFallback` to return API-only message
  - Removed unnecessary dependencies (afero, etc.)
- **Binary size reduced**: From ~20-30MB to ~12MB

### Technical Details
- Based on original AgentAPI commit: [latest]
- Go version: 1.20+
- Supported platforms:
  - macOS Intel (darwin-amd64)
  - macOS Apple Silicon (darwin-arm64)
  - Linux (linux-amd64)
  - Windows (windows-amd64)

### Why This Fork?
This fork was created specifically for Claude Code Studio to enable:
1. Real-time permission mode tracking in VS Code
2. Better integration with Claude's terminal interface
3. Smaller, focused binary without unnecessary UI components
4. Enhanced parsing that preserves important status information

---

CCS = Claude Code Studio