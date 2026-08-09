package infra

import (
	"os"
	"testing"
)

// The shared postgres testcontainer outlives individual tests, so the package
// has to tear it down itself. See RunTestMain.
func TestMain(m *testing.M) {
	os.Exit(RunTestMain(m))
}
