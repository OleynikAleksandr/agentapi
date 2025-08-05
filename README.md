# AgentAPI - Custom Build for Claude Code Studio

This is a custom fork of [AgentAPI](https://github.com/coder/agentapi) modified specifically for [Claude Code Studio](https://github.com/oleksandroliinyk/claude-code-studio).

## What's Changed

This fork includes critical modifications to support Permission Mode tracking in Claude Code Studio:

### New Features
- **Permission Mode Extraction**: API now returns the current permission mode (e.g., "Bypassing Permissions", "plan mode on", "auto-accept edits on", "Normal mode") in both `/status` and `/messages` endpoints
- **Enhanced Message Box Parsing**: Modified `removeMessageBox` function to preserve status lines below the input box
- **Real-time Mode Tracking**: Permission mode is extracted from the terminal screen on every API call

### Technical Changes
- Added `ExtractPermissionMode` function in `lib/msgfmt/permission_mode.go`
- Modified `removeMessageBox` in `lib/msgfmt/message_box.go` to only remove the 3-line message box
- Updated API models to include `permissionMode` field in responses
- **UI Removed**: Chat interface removed for smaller binary size and API-only operation

### API Response Example
```json
{
  "body": {
    "messages": [...],
    "permissionMode": "Bypassing Permissions"
  }
}
```

## Download

Download pre-built binaries from the [latest release](https://github.com/OleynikAleksandr/agentapi/releases/latest):
- macOS Intel: `agentapi-darwin-amd64`
- macOS Apple Silicon: `agentapi-darwin-arm64`
- Linux: `agentapi-linux-amd64`
- Windows: `agentapi-windows-amd64.exe`

## Building from Source

1. Install Go 1.20 or later
2. Clone this repository
3. Run the build script:
   ```bash
   ./build-release.sh
   ```

This will create binaries in the `dist/` directory

## Usage

Same as original AgentAPI, but without the web UI:

```bash
# Start server
agentapi server --port 8080 -- claude

# Send message
curl -X POST localhost:8080/message \
  -H "Content-Type: application/json" \
  -d '{"content": "Hello!", "type": "user"}'

# Get messages with permission mode
curl localhost:8080/messages
```

## Integration with Claude Code Studio

This custom build is designed to work seamlessly with Claude Code Studio extension for VS Code, providing real-time permission mode tracking and enhanced Claude integration.

## Original Project

For the original AgentAPI with full UI support, visit: https://github.com/coder/agentapi