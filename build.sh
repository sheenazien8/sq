#!/bin/bash

# Cross-platform build script for sq

# Get version from git tag or use default
VERSION=${VERSION:-$(git describe --tags --abbrev=0 2>/dev/null || echo "devel")}

# Get build info
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

platforms=("darwin/amd64" "darwin/arm64" "linux/amd64" "linux/arm64" "windows/amd64")

for platform in "${platforms[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"

    output_name="sq-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi

    echo "Building for $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags "-X github.com/sheenazien8/sq/internal/version.Version=${VERSION}" \
        -o "$output_name" .

    if [ $? -ne 0 ]; then
        echo "Failed to build for $GOOS/$GOARCH"
        exit 1
    fi

    echo "Built $output_name"
done

echo "All builds completed successfully."