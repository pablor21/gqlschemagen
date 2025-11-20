package version

import (
	"runtime/debug"
)

// Version will be set via ldflags during build: -ldflags "-X github.com/pablor21/gqlschemagen/version.Version=v1.2.3"
var Version = ""

// Get returns the version string, trying multiple sources in order:
// 1. Version set via ldflags (for binary builds)
// 2. Module version from build info (for library imports via go get/install)
// 3. VCS revision from build info (git commit)
// 4. Default fallback version
func Get() string {
	// If version was set via ldflags, use it (CLI binary)
	if Version != "" {
		return Version
	}

	// Try to get version from build info
	// This works when the module is imported as a dependency
	if info, ok := debug.ReadBuildInfo(); ok {
		// First, try to get the module version (works with go get/install)
		// This will be the version tag when imported as a dependency
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}

		// Check for vcs.revision (git commit) for local development
		var revision, modified string
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value
			case "vcs.modified":
				modified = setting.Value
			}
		}

		// If we have a revision, use it
		if revision != "" {
			if len(revision) > 7 {
				revision = revision[:7] // Short commit hash
			}
			if modified == "true" {
				return revision + "-dirty"
			}
			return revision
		}
	}

	// Default fallback
	return "v0.1.11"
}
