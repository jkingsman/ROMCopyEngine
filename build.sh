#!/bin/bash

set -ex


TAG=""
if [ $# -eq 1 ]; then
    TAG="$1"
fi

# Create build directory
SCRIPT_DIR="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
BUILD_DIR="$SCRIPT_DIR/build"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

# Create a file for all checksums
CHECKSUM_FILE_NAME="SHA256SUMS.txt"
CHECKSUM_FILE_PATH="$BUILD_DIR/$CHECKSUM_FILE_NAME"
touch "$CHECKSUM_FILE_PATH"

# Project name
PROJECT="romcopyengine"

# Platforms to build for
PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "linux/amd64"
    "linux/arm64"
    "linux/386"
    "freebsd/amd64"
    "freebsd/386"
    "android/arm64"
    "windows/386"
    "windows/arm64"
    "windows/amd64"
)

# Create a file for all checksums
CHECKSUM_FILE="$BUILD_DIR/SHA256SUMS.txt"
touch "$CHECKSUM_FILE"

# Build for each platform
for PLATFORM in "${PLATFORMS[@]}"; do
    # Split platform into OS and ARCH
    OS="${PLATFORM%/*}"
    ARCH="${PLATFORM#*/}"

    echo "Building for $OS/$ARCH..."

    # Create temporary directory for this build
    TEMP_DIR=$(mktemp -d)

    # Set binary name based on OS
    if [ "$OS" = "windows" ]; then
        BINARY_NAME="$PROJECT.exe"
    else
        BINARY_NAME="$PROJECT"
    fi

    # Build the binary
    GOOS=$OS GOARCH=$ARCH go build -ldflags "-s -w" -o "$TEMP_DIR/$BINARY_NAME"

    # Copy README and LICENSE
    cp README.md LICENSE.md "$TEMP_DIR/"

    # Create compressed file
    COMPRESSED_NAME="${PROJECT}_${TAG:=dev}_${OS}_${ARCH}"
    if [ "$OS" = "windows" ]; then
        # Create zip for Windows
        (cd "$TEMP_DIR" && zip -r "$BUILD_DIR/${COMPRESSED_NAME}.zip" .)
        # Generate SHA256 checksum
        (cd "$BUILD_DIR" && sha256sum "${COMPRESSED_NAME}.zip" > "$CHECKSUM_FILE_NAME")
    else
        # Create tar.gz for other platforms
        tar -czf "$BUILD_DIR/${COMPRESSED_NAME}.tgz" -C "$TEMP_DIR" .
        # Generate SHA256 checksum
        (cd "$BUILD_DIR" && sha256sum "${COMPRESSED_NAME}.tgz" > "$CHECKSUM_FILE_NAME")
    fi

    # Clean up temporary directory
    rm -rf "$TEMP_DIR"
done

echo "Build complete! Artifacts are in the $BUILD_DIR directory"
echo "SHA256 checksums are available in $CHECKSUM_FILE"

# Only create and push tag if one was provided
if [ -n "$TAG" ]; then
    git tag -d "$TAG" || true
    git push --delete origin "$TAG" || true
    git tag "$TAG"
    git push origin "$TAG"
else
    echo "Warning: No tag was provided - git tag was not created or pushed"
fi
