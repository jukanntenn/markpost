//go:build acceptance

// Package acceptance runs the compiled CLI binary against a real markpost
// server (gh's acceptance tier: build-tagged, env-configured, never run by
// plain `go test ./...`). Point it at a disposable instance — the tests
// create and delete posts, rotate the post key, and log the session out.
//
// Required environment:
//
//	MARKPOST_E2E_BASE_URL   e.g. http://localhost:2053
//	MARKPOST_E2E_USERNAME   an account on that server
//	MARKPOST_E2E_PASSWORD   its password
//
// For the dev stack: `python3 devops/dev.py start` then run with the
// credentials from devops/docker-compose.yml.
package acceptance

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var binPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "markpost-cli-acceptance-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "temp dir:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)
	binPath = filepath.Join(dir, "markpost")
	build := exec.Command("go", "build", "-o", binPath, "../cmd/markpost")
	build.Stdout, build.Stderr = os.Stdout, os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "building CLI:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

type harness struct {
	t      *testing.T
	base   string
	config string // isolated MARKPOST_CONFIG_DIR
}

func newHarness(t *testing.T) *harness {
	base := os.Getenv("MARKPOST_E2E_BASE_URL")
	username := os.Getenv("MARKPOST_E2E_USERNAME")
	password := os.Getenv("MARKPOST_E2E_PASSWORD")
	if base == "" || username == "" || password == "" {
		t.Skip("set MARKPOST_E2E_BASE_URL, MARKPOST_E2E_USERNAME, and MARKPOST_E2E_PASSWORD to run acceptance tests")
	}
	return &harness{t: t, base: strings.TrimSuffix(base, "/"), config: t.TempDir()}
}

// run executes the CLI binary with the isolated config dir and returns
// combined stdout plus the exit code.
func (h *harness) run(args ...string) (string, int) {
	h.t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Env = append(os.Environ(),
		"MARKPOST_CONFIG_DIR="+h.config,
		"MARKPOST_SERVER="+h.base,
	)
	out, err := cmd.Output()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		h.t.Fatalf("run %v: %v", args, err)
	}
	return string(out), code
}

func TestAcceptanceGoldenPath(t *testing.T) {
	h := newHarness(t)
	user := os.Getenv("MARKPOST_E2E_USERNAME")
	pass := os.Getenv("MARKPOST_E2E_PASSWORD")

	out, code := h.run("status")
	require.Equal(t, 0, code, "status output: %s", out)
	assert.Contains(t, out, "Health:  ok")
	assert.Contains(t, out, "Version:")

	out, code = h.run("status", "--json")
	require.Equal(t, 0, code, out)
	var status struct {
		Version string `json:"version"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &status))
	assert.NotEmpty(t, status.Version)

	out, code = h.run("auth", "login", "--username", user, "--password", pass)
	require.Equal(t, 0, code, out)
	assert.Contains(t, out, "Logged in as ")

	out, code = h.run("auth", "status")
	require.Equal(t, 0, code, out)
	assert.Contains(t, out, "Verification: ok")

	out, code = h.run("post-key", "show")
	require.Equal(t, 0, code, out)
	assert.True(t, strings.HasPrefix(strings.TrimSpace(out), "mpk-"), "post key: %s", out)

	title := fmt.Sprintf("acceptance-%d", os.Getpid())
	out, code = h.run("posts", "create", "--title", title, "e2e body")
	require.Equal(t, 0, code, out)
	url := strings.TrimSpace(out)
	qid := url[strings.LastIndex(url, "/")+1:]
	assert.NotEmpty(t, qid)
	assert.True(t, strings.HasPrefix(qid, "p-"), "qid: %s", qid)

	out, code = h.run("posts", "view", qid)
	require.Equal(t, 0, code, out)
	assert.Contains(t, out, title)

	out, code = h.run("posts", "list", "--json")
	require.Equal(t, 0, code, out)
	assert.Contains(t, out, qid)

	_, code = h.run("posts", "delete", "--yes", qid)
	require.Equal(t, 0, code)

	out, code = h.run("posts", "view", qid)
	assert.Equal(t, 1, code, "deleted post must 404")
	assert.Empty(t, out)

	out, code = h.run("api", "me/retention")
	require.Equal(t, 0, code, out)
	var retention struct {
		PostsDays int `json:"posts_days"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &retention))

	_, code = h.run("auth", "logout")
	require.Equal(t, 0, code)

	_, code = h.run("auth", "token")
	assert.Equal(t, 4, code, "after logout the CLI must exit 4 until re-login")
}

func TestAcceptanceEnvTokenSession(t *testing.T) {
	h := newHarness(t)
	user := os.Getenv("MARKPOST_E2E_USERNAME")
	pass := os.Getenv("MARKPOST_E2E_PASSWORD")

	// TestAcceptanceGoldenPath ends with a logout; two logins inside the same
	// second mint byte-identical JWTs, and the server blacklists by token
	// hash, so a same-second re-login inherits the revocation
	// (jukanntenn/markpost#84). Wait the collision window away until the
	// server mints unique tokens.
	time.Sleep(1200 * time.Millisecond)

	out, code := h.run("auth", "login", "--username", user, "--password", pass)
	require.Equal(t, 0, code, out)

	token, code := h.run("auth", "token")
	require.Equal(t, 0, code)

	// A fresh config dir with only MARKPOST_TOKEN must reach authed data.
	cmd := exec.Command(binPath, "posts", "list", "--json")
	cmd.Env = append(os.Environ(),
		"MARKPOST_CONFIG_DIR="+t.TempDir(),
		"MARKPOST_SERVER="+h.base,
		"MARKPOST_TOKEN="+strings.TrimSpace(token),
	)
	raw, err := cmd.Output()
	require.NoError(t, err, "env-token session output: %s", raw)
	assert.Contains(t, string(raw), `"items"`)
}
