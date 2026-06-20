// Package buildinfo exposes linker-injected build metadata for the mailgraph binary.
package buildinfo

// Version is the application version, set at link time via -ldflags.
var Version = "dev"

// BuildDate is the UTC build timestamp, set at link time via -ldflags.
var BuildDate = "unknown"

// GitCommit is the short git revision, set at link time via -ldflags.
var GitCommit = "unknown"