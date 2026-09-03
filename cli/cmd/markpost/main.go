// Command markpost is the markpost CLI: a standalone client for publishing
// and managing markdown posts on a markpost server, designed to be driven by
// both humans and AI agents.
package main

import (
	"os"

	"markpost/cli/internal/cmd"
	"markpost/cli/internal/iostreams"
)

// version is overridden at build time via
// -ldflags "-X main.version=$(git describe --tags --always)".
var version = "dev"

func main() {
	os.Exit(cmd.Main(version, os.Args, iostreams.System(), ""))
}
