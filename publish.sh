#!/bin/bash

set -e

# Required branch for publishing (default: master)
REQUIRED_BRANCH="${PUBLISH_BRANCH:-master}"

# Default version increment type
VERSION_INCREMENT="patch"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to get the latest version from git tags
get_latest_version() {
    local latest=$(git tag -l "v[0-9]*.[0-9]*.[0-9]*" | sort -V | tail -n 1)
    if [ -z "$latest" ]; then
        echo "v0.0.0"
    else
        echo "$latest"
    fi
}

# Function to increment version (patch)
increment_version() {
    local version=$1
    local increment_type=$2
    # Remove 'v' prefix
    local ver=${version#v}
    # Split version into parts
    local major=$(echo $ver | cut -d. -f1)
    local minor=$(echo $ver | cut -d. -f2)
    local patch=$(echo $ver | cut -d. -f3)
    
    case "$increment_type" in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch)
            patch=$((patch + 1))
            ;;
    esac
    
    echo "v${major}.${minor}.${patch}"
}

# Parse arguments
VERSION=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --major|--minor|--patch)
            if [ "$VERSION_INCREMENT" != "patch" ]; then
                print_error "Only one version increment flag (--major, --minor, or --patch) can be specified"
                exit 1
            fi
            VERSION_INCREMENT="${1#--}"
            shift
            ;;
        v[0-9]*)
            VERSION="$1"
            shift
            ;;
        *)
            print_error "Unknown argument: $1"
            echo "Usage: ./publish.sh [--major|--minor|--patch] [version]"
            echo "Examples:"
            echo "  ./publish.sh              # Auto-increment patch version"
            echo "  ./publish.sh --minor      # Auto-increment minor version"
            echo "  ./publish.sh --major      # Auto-increment major version"
            echo "  ./publish.sh v1.2.3       # Publish specific version"
            exit 1
            ;;
    esac
done

# Check if version argument is provided, otherwise auto-increment
if [ -z "$VERSION" ]; then
    print_info "No version provided, auto-incrementing ${VERSION_INCREMENT} version from latest tag..."
    LATEST_VERSION=$(get_latest_version)
    VERSION=$(increment_version "$LATEST_VERSION" "$VERSION_INCREMENT")
    print_info "Latest version: $LATEST_VERSION"
    print_info "New version: $VERSION (${VERSION_INCREMENT} increment)"
    echo
    read -p "Publish version $VERSION? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Aborted."
        exit 1
    fi
else
    VERSION=$1
    # Validate version format (must start with v)
    if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+.*$ ]]; then
        print_error "Invalid version format. Must be in format v0.1.0"
        exit 1
    fi
fi

print_info "Publishing version: $VERSION"

# Check if on required branch
BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$BRANCH" != "$REQUIRED_BRANCH" ]]; then
    print_error "Must be on $REQUIRED_BRANCH branch (current: $BRANCH)"
    echo "Switch to $REQUIRED_BRANCH or set PUBLISH_BRANCH environment variable"
    exit 1
fi

# Check for uncommitted changes
if [[ -n $(git status -s) ]]; then
    print_warning "You have uncommitted changes:"
    git status -s
    echo
    read -p "Continue anyway? These changes will NOT be included in the release. (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Aborted. Commit your changes and try again."
        exit 1
    fi
fi

# Run tests in main module (with replace directive intact)
print_info "Running tests in main module..."
go test ./... -v
if [ $? -ne 0 ]; then
    print_error "Tests failed in main module"
    exit 1
fi

print_info "All tests passed ✓"

# Push current state to GitHub
print_info "Pushing to GitHub..."
git push origin $BRANCH
if [ $? -ne 0 ]; then
    print_error "Failed to push to GitHub"
    exit 1
fi

# Commit the clean version
print_info "Committing release..."
git commit --allow-empty -m "release: ${VERSION}"

# Tag main module
print_info "Tagging main module with $VERSION..."
git tag -a "$VERSION" -m "Release $VERSION"

# Push tags and release commit
print_info "Pushing tags and release commit to GitHub..."
git push origin $BRANCH
git push origin "$VERSION"

# Verify tags were pushed correctly
print_info "Verifying tags on remote..."
MAIN_TAG_SHA=$(git ls-remote --tags origin | grep "refs/tags/$VERSION^{}" | awk '{print $1}')

if [ -z "$MAIN_TAG_SHA" ]; then
    print_error "Main tag $VERSION not found on remote!"
    exit 1
fi

EXPECTED_SHA=$(git rev-parse HEAD)
if [ "$MAIN_TAG_SHA" != "$EXPECTED_SHA" ]; then
    print_error "Tag SHA mismatch! Expected: $EXPECTED_SHA"
    print_error "Main tag: $MAIN_TAG_SHA"
    exit 1
fi

print_info "✓ Tags verified on remote (SHA: ${EXPECTED_SHA:0:8})"

# Trigger Go proxy to fetch the module
print_info "Triggering Go proxy to fetch module..."
GOPROXY=https://proxy.golang.org,direct go list -m github.com/pablor21/gqlschemagen@$VERSION > /dev/null 2>&1 || true
print_info "✓ Proxy fetch triggered (indexing may take 10-30 minutes)"

print_info "Successfully published:"
print_info "  - Main module: github.com/pablor21/gqlschemagen@$VERSION"
print_info ""
print_info "Installation:"
print_info "  go get github.com/pablor21/gqlschemagen@$VERSION"
print_info ""
print_info "Note: The Go checksum database may take 10-30 minutes to index."
print_info "      If installation fails with 'invalid version: unknown revision',"
print_info "      wait a few minutes and try again."
print_info ""
print_info "pkg.go.dev will index automatically (may take a few minutes):"
print_info "  - https://pkg.go.dev/github.com/pablor21/gqlschemagen@$VERSION"

