// Package version holds the build-time version of the workspace-manager binary.
//
// The single exported variable Version is meant to be overridden at release
// build time via the linker flag:
//
//	-ldflags "-X github.com/cy83rt00n/workspace-manager/internal/version.Version=<version>"
//
// Built without this flag it defaults to "dev".
package version

// Version is the build-time version string. It is overridden at release time
// through -ldflags "-X .../internal/version.Version=..."; otherwise it stays
// "dev".
var Version = "dev"
