package post

import (
	"os"
	"testing"

	"markpost/internal/infra"
)

// The shared postgres testcontainer outlives individual tests, so the package
// has to tear it down itself. See infra.RunTestMain.
func TestMain(m *testing.M) {
	os.Exit(infra.RunTestMain(m))
}
