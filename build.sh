#!/bin/bash
# Build script for gqlschemagen with automatic version injection
#
# This script builds the gqlschemagen binary with version information
# injected at build time using -ldflags.
#
# Usage:
#   ./build.sh           # Build with git-based version
#   ./build.sh v1.2.3    # Build with specific version

set -e

# Get version from argument or git
if [ -n "$1" ]; then
    VERSION="$1"
else
    # Try to get version from git tag
    if git describe --tags --exact-match 2>/dev/null; then
        VERSION=$(git describe --tags --exact-match)
    elif git rev-parse --short HEAD 2>/dev/null; then
        # Use commit hash if no tag
        COMMIT=$(git rev-parse --short HEAD)
        if git diff-index --quiet HEAD -- 2>/dev/null; then
            VERSION="$COMMIT"
        else
            VERSION="$COMMIT-dirty"
        fi
    else
        VERSION="dev"
    fi
fi

echo "Building gqlschemagen version: $VERSION"

# Build with version injected
go build -ldflags "-X github.com/pablor21/gqlschemagen/version.Version=$VERSION" -o gqlschemagen

echo "Build complete: ./gqlschemagen"
./gqlschemagen version
