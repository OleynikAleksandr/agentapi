#!/bin/bash

# Build script for AgentAPI v1.6.0
VERSION="v1.6.0"

echo "Building AgentAPI $VERSION for all platforms..."

# Create dist directory
mkdir -p dist

# Build for each platform
PLATFORMS=(
    "darwin arm64"
    "darwin amd64"
    "linux amd64"
    "linux arm64"
    "windows amd64"
)

for platform in "${PLATFORMS[@]}"; do
    IFS=' ' read -r -a parts <<< "$platform"
    GOOS="${parts[0]}"
    GOARCH="${parts[1]}"
    
    output_name="agentapi_${GOOS}_${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    echo "Building for $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build -o "dist/$output_name" .
    
    if [ $? -eq 0 ]; then
        echo "✓ Built $output_name"
    else
        echo "✗ Failed to build $output_name"
    fi
done

echo "Build complete! Binaries are in dist/ directory"
ls -la dist/