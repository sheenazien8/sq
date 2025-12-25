package version

import "runtime/debug"

// Version is set at build time via -ldflags
var Version string

func init() {
	// If version was set via ldflags, use it
	if Version != "" {
		return
	}

	// Otherwise, try to get version from build info (for `go install` compatibility)
	info, ok := debug.ReadBuildInfo()
	if !ok {
		Version = "devel"
		return
	}
	Version = info.Main.Version
	if Version == "" || Version == "(devel)" {
		Version = "devel"
	}
}
