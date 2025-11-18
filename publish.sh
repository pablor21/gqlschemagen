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

# Run plugin tests only if replace directive exists
if grep -q "^replace github.com/pablor21/gqlschemagen =>" plugin/go.mod; then
    print_info "Running tests in plugin module..."
    cd plugin
    go test ./... -v
    if [ $? -ne 0 ]; then
        print_error "Tests failed in plugin module"
        cd ..
        exit 1
    fi
    cd ..
else
    print_warning "Skipping plugin tests (replace directive not found, this is normal for published versions)"
fi

print_info "All tests passed ✓"

# Push current state to GitHub
print_info "Pushing to GitHub..."
git push origin $BRANCH
if [ $? -ne 0 ]; then
    print_error "Failed to push to GitHub"
    exit 1
fi

# Create backup and clean go.mod for release
print_info "Preparing clean release (removing replace directive)..."
if grep -q "^replace github.com/pablor21/gqlschemagen =>" plugin/go.mod; then
    # Save the original go.mod for restoration
    cp plugin/go.mod plugin/go.mod.bkp
    
    # Remove the replace directive and blank line after it
    sed -i.tmp '/^replace github.com\/pablor21\/gqlschemagen =>/d' plugin/go.mod
    sed -i.tmp '/^$/N;/^\n$/d' plugin/go.mod  # Remove extra blank lines
    rm -f plugin/go.mod.tmp
    
    # Update the version requirement to use the actual version (replace any existing version)
    sed -i.tmp "s|github.com/pablor21/gqlschemagen v[0-9]*\.[0-9]*\.[0-9]*|github.com/pablor21/gqlschemagen ${VERSION}|" plugin/go.mod
    rm -f plugin/go.mod.tmp
    
    print_info "✓ Backup saved to plugin/go.mod.bkp"
fi

# Commit the clean version
print_info "Committing clean release..."
git add plugin/go.mod
git commit -m "release: ${VERSION}"

# Tag main module
print_info "Tagging main module with $VERSION..."
git tag -a "$VERSION" -m "Release $VERSION"

# Tag plugin module (same version with /plugin suffix)
print_info "Tagging plugin module with plugin/$VERSION..."
git tag -a "plugin/$VERSION" -m "Release plugin $VERSION"

# Push tags and clean release commit
print_info "Pushing tags and release commit to GitHub..."
git push origin $BRANCH
git push origin "$VERSION"
git push origin "plugin/$VERSION"

# Verify tags were pushed correctly
print_info "Verifying tags on remote..."
MAIN_TAG_SHA=$(git ls-remote --tags origin | grep "refs/tags/$VERSION^{}" | awk '{print $1}')
PLUGIN_TAG_SHA=$(git ls-remote --tags origin | grep "refs/tags/plugin/$VERSION^{}" | awk '{print $1}')

if [ -z "$MAIN_TAG_SHA" ]; then
    print_error "Main tag $VERSION not found on remote!"
    exit 1
fi

if [ -z "$PLUGIN_TAG_SHA" ]; then
    print_error "Plugin tag plugin/$VERSION not found on remote!"
    exit 1
fi

EXPECTED_SHA=$(git rev-parse HEAD)
if [ "$MAIN_TAG_SHA" != "$EXPECTED_SHA" ] || [ "$PLUGIN_TAG_SHA" != "$EXPECTED_SHA" ]; then
    print_error "Tag SHA mismatch! Expected: $EXPECTED_SHA"
    print_error "Main tag: $MAIN_TAG_SHA"
    print_error "Plugin tag: $PLUGIN_TAG_SHA"
    exit 1
fi

print_info "✓ Tags verified on remote (SHA: ${EXPECTED_SHA:0:8})"

# Verify go.mod is clean at the tagged commit
print_info "Verifying plugin/go.mod is clean at tag..."
TAGGED_GOMOD=$(git show "plugin/$VERSION:plugin/go.mod")
if echo "$TAGGED_GOMOD" | grep -q "^replace github.com/pablor21/gqlschemagen =>"; then
    print_error "Replace directive found in tagged version!"
    print_error "The tag points to a commit with replace directive, which will break installation."
    exit 1
fi
print_info "✓ plugin/go.mod is clean at tag"

# Trigger Go proxy to fetch the module
print_info "Triggering Go proxy to fetch modules..."
GOPROXY=https://proxy.golang.org,direct go list -m github.com/pablor21/gqlschemagen@$VERSION > /dev/null 2>&1 || true
GOPROXY=https://proxy.golang.org,direct go list -m github.com/pablor21/gqlschemagen/plugin@$VERSION > /dev/null 2>&1 || true
print_info "✓ Proxy fetch triggered (indexing may take 10-30 minutes)"

# Restore replace directive from backup
if [ -f plugin/go.mod.bkp ]; then
    print_info "Restoring replace directive from backup..."
    mv plugin/go.mod.bkp plugin/go.mod
    git add plugin/go.mod
    git commit -m "chore: restore replace directive for development"
    print_info "✓ Replace directive restored and committed"
else
    print_warning "No backup found (plugin/go.mod.bkp), skipping restore"
fi

print_info "Successfully published:"
print_info "  - Main module: github.com/pablor21/gqlschemagen@$VERSION"
print_info "  - Plugin module: github.com/pablor21/gqlschemagen/plugin@$VERSION"
print_info ""
print_info "Installation:"
print_info "  Main:   go get github.com/pablor21/gqlschemagen@$VERSION"
print_info "  Plugin: go get github.com/pablor21/gqlschemagen/plugin@$VERSION"
print_info ""
print_info "Note: The Go checksum database may take 10-30 minutes to index."
print_info "      If installation fails with 'invalid version: unknown revision',"
print_info "      wait a few minutes and try again, or use:"
print_info "      GONOSUMDB=github.com/pablor21/gqlschemagen/plugin go get github.com/pablor21/gqlschemagen/plugin@$VERSION"
print_info ""
print_info "pkg.go.dev will index automatically (may take a few minutes):"
print_info "  - https://pkg.go.dev/github.com/pablor21/gqlschemagen@$VERSION"
print_info "  - https://pkg.go.dev/github.com/pablor21/gqlschemagen/plugin@$VERSION"
