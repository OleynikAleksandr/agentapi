#!/bin/bash

# Build script for custom AgentAPI (Claude Code Studio version)

echo "Building custom AgentAPI for Claude Code Studio..."
echo "This build includes PTY with 160 columns width by default"

VERSION="v1.1.0-ccs"
mkdir -p dist

# macOS Intel
echo "Building for macOS Intel (amd64)..."
GOOS=darwin GOARCH=amd64 go build -o dist/agentapi-darwin-amd64 main.go

# macOS Apple Silicon
echo "Building for macOS Apple Silicon (arm64)..."
GOOS=darwin GOARCH=arm64 go build -o dist/agentapi-darwin-arm64 main.go

# Linux AMD64 (optional, for completeness)
echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -o dist/agentapi-linux-amd64 main.go

# Windows AMD64 (optional, for completeness)
echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -o dist/agentapi-windows-amd64.exe main.go

echo ""
echo "Build complete! Version: $VERSION"
echo "Binaries are in the dist/ directory:"
ls -la dist/

echo ""
echo "Features in this custom build:"
echo "- PTY terminal width set to 160 columns by default"
echo "- Message box parsing preserves status lines"
echo "- UI removed for smaller binary size"
echo "- Permission Mode parser removed (handled by extension)"