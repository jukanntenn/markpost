package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"markpost/cli/internal/config"
	"markpost/cli/internal/iostreams"
	"markpost/cli/internal/testserver"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cliTest wires one fake backend, one temp config file, and a runner that
// invokes Main in-process with buffer streams — the urfave equivalent of
// gh's runCommand helper: real flag parsing, real command dispatch, real
// HTTP, but no subprocess.
type cliTest struct {
	t       *testing.T
	fake    *testserver.Server
	url     string
	cfgPath string
}

func newCLITest(t *testing.T) *cliTest {
	fake := testserver.New()
	url := fake.Start(t)
	return &cliTest{
		t:       t,
		fake:    fake,
		url:     url,
		cfgPath: filepath.Join(t.TempDir(), "config.toml"),
	}
}

type result struct {
	stdout string
	stderr string
	exit   int
}

// run seeds the config file (nil writes nothing), pipes stdin, scrubs then
// applies MARKPOST_* env vars, and executes args against Main.
func (ct *cliTest) run(cfg *config.Config, stdin string, env map[string]string, args ...string) result {
	ct.t.Helper()
	if cfg != nil {
		require.NoError(ct.t, config.Save(ct.cfgPath, cfg))
	}
	for _, key := range []string{"MARKPOST_SERVER", "MARKPOST_TOKEN", "MARKPOST_POST_KEY", "MARKPOST_CONFIG_DIR", "XDG_CONFIG_HOME", "AI_AGENT"} {
		ct.t.Setenv(key, "")
	}
	for key, value := range env {
		ct.t.Setenv(key, value)
	}
	io, stdinBuf, stdout, stderr := iostreams.Test()
	stdinBuf.WriteString(stdin)
	exit := Main("test-version", append([]string{"markpost"}, args...), io, ct.cfgPath)
	return result{stdout: stdout.String(), stderr: stderr.String(), exit: exit}
}

// serverOnlyCfg points at the fake backend with no session.
func (ct *cliTest) serverOnlyCfg() *config.Config {
	return &config.Config{Auth: config.Auth{Server: ct.url}}
}

// loggedInCfg seeds a valid session (token "at-1", refresh "rt-1").
func (ct *cliTest) loggedInCfg() *config.Config {
	return &config.Config{Auth: config.Auth{
		Server:       ct.url,
		UserID:       1,
		Username:     testserver.Username,
		Name:         "Alice",
		Role:         "user",
		Token:        "at-1",
		RefreshToken: "rt-1",
		ExpiresAt:    time.Now().Add(time.Hour).Unix(),
	}}
}

func (ct *cliTest) loadCfg() *config.Config {
	ct.t.Helper()
	cfg, err := config.Load(ct.cfgPath)
	require.NoError(ct.t, err)
	return cfg
}

// --- auth ---

func TestAuthLoginWithFlags(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", nil,
		"auth", "login", "--username", testserver.Username, "--password", testserver.Password)
	assert.Equal(t, 0, res.exit)
	assert.Equal(t, fmt.Sprintf("Logged in as %s to %s\n", testserver.Username, ct.url), res.stdout)
	assert.Empty(t, res.stderr)

	cfg := ct.loadCfg()
	assert.Equal(t, ct.url, cfg.Auth.Server)
	assert.Equal(t, "at-1", cfg.Auth.Token)
	assert.Equal(t, "rt-1", cfg.Auth.RefreshToken)
	assert.Equal(t, testserver.Username, cfg.Auth.Username)
	assert.Greater(t, cfg.Auth.ExpiresAt, time.Now().Unix())
}

func TestAuthLoginPasswordStdin(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), testserver.Password+"\n", nil,
		"auth", "login", "--username", testserver.Username, "--password-stdin")
	assert.Equal(t, 0, res.exit)
	assert.Equal(t, "at-1", ct.loadCfg().Auth.Token)
}

func TestAuthLoginWrongPassword(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", nil,
		"auth", "login", "--username", testserver.Username, "--password", "nope")
	assert.Equal(t, 1, res.exit)
	assert.Equal(t, "markpost: login failed: HTTP 401: invalid_credentials: invalid username or password\n", res.stderr)
}

func TestAuthLoginNonInteractiveRequiresUsername(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", nil, "auth", "login")
	assert.Equal(t, 1, res.exit)
	assert.Contains(t, res.stderr, "--username is required when not interactive")
}

func TestAuthLoginWithToken(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", nil,
		"auth", "login", "--token", ct.fake.ValidAccess())
	assert.Equal(t, 0, res.exit)
	assert.Equal(t, fmt.Sprintf("Logged in to %s with token\n", ct.url), res.stdout)
	cfg := ct.loadCfg()
	assert.Equal(t, "at-1", cfg.Auth.Token)
	assert.Empty(t, cfg.Auth.RefreshToken, "--token sessions carry no refresh token")
}

func TestAuthLoginWithRejectedToken(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", nil, "auth", "login", "--token", "garbage")
	assert.Equal(t, 1, res.exit)
	assert.Contains(t, res.stderr, "token rejected")
}

func TestAuthToken(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.loggedInCfg(), "", nil, "auth", "token")
	assert.Equal(t, 0, res.exit)
	assert.Equal(t, "at-1\n", res.stdout)
}

func TestAuthTokenEnvOverride(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", map[string]string{"MARKPOST_TOKEN": "at-1"}, "auth", "token")
	assert.Equal(t, 0, res.exit)
	assert.Equal(t, "at-1\n", res.stdout)
}

func TestAuthNotLoggedInExitsFour(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", nil, "auth", "token")
	assert.Equal(t, 4, res.exit)
	assert.Contains(t, res.stderr, "not logged in")
	assert.Contains(t, res.stderr, "markpost auth login")
}

func TestAuthStatusLoggedIn(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.loggedInCfg(), "", nil, "auth", "status")
	assert.Equal(t, 0, res.exit)
	assert.Contains(t, res.stdout, fmt.Sprintf("Server: %s", ct.url))
	assert.Contains(t, res.stdout, "Session: alice")
	assert.Contains(t, res.stdout, "Verification: ok")
}

func TestAuthStatusNotLoggedIn(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", nil, "auth", "status")
	assert.Equal(t, 4, res.exit)
	assert.Contains(t, res.stdout, "Session: none")
}

func TestAuthLogout(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.loggedInCfg(), "", nil, "auth", "logout")
	assert.Equal(t, 0, res.exit)
	assert.Equal(t, fmt.Sprintf("Logged out of %s\n", ct.url), res.stdout)

	cfg := ct.loadCfg()
	assert.Equal(t, ct.url, cfg.Auth.Server, "the server stays configured after logout")
	assert.Empty(t, cfg.Auth.Token)
	assert.Empty(t, cfg.Auth.RefreshToken)
	assert.Contains(t, ct.fake.Requests(), "POST /api/v1/auth/logout")
}

// --- posts ---

func TestPostsCreateFetchesPostKeyWhenLoggedIn(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.loggedInCfg(), "", nil,
		"posts", "create", "--title", "Hello", "World body")
	assert.Equal(t, 0, res.exit)
	assert.Equal(t, ct.url+"/p-2\n", res.stdout)
	requests := ct.fake.Requests()
	assert.Contains(t, requests, "GET /api/v1/post-key")
	assert.Contains(t, requests, "POST /"+testserver.PostKey)
}

func TestPostsCreateJSON(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.loggedInCfg(), "", nil,
		"posts", "create", "--title", "Hello", "--json", "World")
	assert.Equal(t, 0, res.exit)
	assert.JSONEq(t, fmt.Sprintf(`{"qid":"p-2","url":"%s/p-2"}`, ct.url), res.stdout)
}

func TestPostsCreateWithPostKeyFlagWithoutLogin(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", nil,
		"posts", "create", "--title", "Hello", "--post-key", testserver.PostKey, "World")
	assert.Equal(t, 0, res.exit)
	assert.Equal(t, ct.url+"/p-2\n", res.stdout)
	for _, req := range ct.fake.Requests() {
		assert.NotContains(t, req, "/api/v1/post-key", "explicit post key must skip the JWT lookup")
	}
}

func TestPostsCreateFromStdin(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "# Piped\n\nbody", map[string]string{"MARKPOST_POST_KEY": testserver.PostKey},
		"posts", "create", "--title", "Hello")
	assert.Equal(t, 0, res.exit)
	assert.Equal(t, ct.url+"/p-2\n", res.stdout)
}

func TestPostsCreateFromFile(t *testing.T) {
	ct := newCLITest(t)
	file := filepath.Join(t.TempDir(), "post.md")
	require.NoError(t, os.WriteFile(file, []byte("from file"), 0o600))
	res := ct.run(ct.loggedInCfg(), "", nil,
		"posts", "create", "--title", "Hello", "--file", file)
	assert.Equal(t, 0, res.exit)
	assert.Equal(t, ct.url+"/p-2\n", res.stdout)
}

func TestPostsCreateEmptyBodyRejected(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.loggedInCfg(), "   ", nil, "posts", "create", "--title", "Hello")
	assert.Equal(t, 1, res.exit)
	assert.Contains(t, res.stderr, "body is empty")
}

func TestPostsCreateRequiresLoginWithoutPostKey(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", nil, "posts", "create", "--title", "Hello", "World")
	assert.Equal(t, 4, res.exit)
	assert.Contains(t, res.stderr, "not logged in")
}

func TestPostsList(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.loggedInCfg(), "", nil, "posts", "list")
	assert.Equal(t, 0, res.exit)
	assert.Contains(t, res.stdout, "QID")
	assert.Contains(t, res.stdout, "p-existing")
	assert.Contains(t, res.stdout, "Existing")
	assert.Contains(t, res.stderr, "Page 1 of 1 (1 posts)\n")
}

func TestPostsListJSON(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.loggedInCfg(), "", nil, "posts", "list", "--json")
	assert.Equal(t, 0, res.exit)
	var envelope struct {
		Items []struct {
			QID string `json:"qid"`
		} `json:"items"`
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal([]byte(res.stdout), &envelope))
	assert.Equal(t, 1, envelope.Total)
	assert.Equal(t, "p-existing", envelope.Items[0].QID)
}

func TestPostsListEmpty(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.loggedInCfg(), "", nil, "posts", "list", "--search", "nomatch")
	assert.Equal(t, 0, res.exit)
	assert.Empty(t, res.stdout)
	assert.Equal(t, "No posts found.\n", res.stderr)
}

func TestPostsViewRaw(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", nil, "posts", "view", "p-existing")
	assert.Equal(t, 0, res.exit)
	assert.Equal(t, "# Existing\n\nBody", res.stdout)
}

func TestPostsViewURL(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", nil, "posts", "view", "--format", "url", "p-existing")
	assert.Equal(t, 0, res.exit)
	assert.Equal(t, ct.url+"/p-existing\n", res.stdout)
}

func TestPostsViewHTML(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", nil, "posts", "view", "--format", "html", "p-existing")
	assert.Equal(t, 0, res.exit)
	assert.Contains(t, res.stdout, "<h1>Existing</h1>")
}

func TestPostsViewBadFormat(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", nil, "posts", "view", "--format", "xml", "p-existing")
	assert.Equal(t, 1, res.exit)
	assert.Contains(t, res.stderr, "invalid --format")
}

func TestPostsViewMissingArgument(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", nil, "posts", "view")
	assert.Equal(t, 1, res.exit)
	assert.Contains(t, res.stderr, "requires exactly one")
}

func TestPostsDelete(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.loggedInCfg(), "", nil, "posts", "delete", "--yes", "p-existing")
	assert.Equal(t, 0, res.exit)
	assert.Empty(t, res.stdout)
	assert.False(t, ct.fake.HasPost("p-existing"))
}

func TestPostsDeleteRequiresYesWhenNonInteractive(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.loggedInCfg(), "", nil, "posts", "delete", "p-existing")
	assert.Equal(t, 1, res.exit)
	assert.Contains(t, res.stderr, "--yes is required")
}

func TestPostsDeleteNotFound(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.loggedInCfg(), "", nil, "posts", "delete", "--yes", "p-missing")
	assert.Equal(t, 1, res.exit)
	assert.Contains(t, res.stderr, "HTTP 404")
}

// --- post-key ---

func TestPostKeyShow(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.loggedInCfg(), "", nil, "post-key", "show")
	assert.Equal(t, 0, res.exit)
	assert.Equal(t, testserver.PostKey+"\n", res.stdout)
}

func TestPostKeyShowJSON(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.loggedInCfg(), "", nil, "post-key", "show", "--json")
	assert.Equal(t, 0, res.exit)
	assert.JSONEq(t, `{"post_key":"mpk-test","created_at":"2026-09-03T00:00:00Z"}`, res.stdout)
}

func TestPostKeyRotate(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.loggedInCfg(), "", nil, "post-key", "rotate")
	assert.Equal(t, 0, res.exit)
	assert.Equal(t, "mpk-rotated\n", res.stdout)
}

// --- config ---

func TestConfigSetGet(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(nil, "", nil, "config", "set", "server", ct.url)
	assert.Equal(t, 0, res.exit)
	assert.Equal(t, "server = "+ct.url+"\n", res.stdout)

	res = ct.run(nil, "", nil, "config", "get", "server")
	assert.Equal(t, 0, res.exit)
	assert.Equal(t, ct.url+"\n", res.stdout)
}

func TestConfigSetInvalidURL(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(nil, "", nil, "config", "set", "server", "notaurl")
	assert.Equal(t, 1, res.exit)
	assert.Contains(t, res.stderr, "must start with http")
}

func TestConfigUnknownKey(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(nil, "", nil, "config", "set", "frob", "x")
	assert.Equal(t, 1, res.exit)
	assert.Contains(t, res.stderr, `unknown config key "frob"`)
}

// --- api passthrough ---

func TestAPIRelativePathIsAuthed(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.loggedInCfg(), "", nil, "api", "me/retention")
	assert.Equal(t, 0, res.exit)
	assert.JSONEq(t, `{"posts_days":7,"history_days":30}`, res.stdout)
	assert.Contains(t, ct.fake.Requests(), "GET /api/v1/me/retention")
}

func TestAPIAbsolutePublicPath(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", nil, "api", "/api/v1/health")
	assert.Equal(t, 0, res.exit)
	assert.JSONEq(t, `{"status":"ok"}`, res.stdout)
}

func TestAPIPostWithInput(t *testing.T) {
	ct := newCLITest(t)
	file := filepath.Join(t.TempDir(), "body.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"title":"API","body":"posted"}`), 0o600))
	res := ct.run(ct.serverOnlyCfg(), "", nil,
		"api", "-X", "POST", "--input", file, "/"+testserver.PostKey)
	assert.Equal(t, 0, res.exit)
	assert.Contains(t, res.stdout, `"id":"p-2"`)
}

func TestAPIHTTPErrorPrintsBodyAndFails(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", nil, "api", "/p-missing")
	assert.Equal(t, 1, res.exit)
	assert.Contains(t, res.stdout, "not_found")
	assert.Contains(t, res.stderr, "HTTP 404")
}

// --- status / version / root ---

func TestStatus(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", nil, "status")
	assert.Equal(t, 0, res.exit)
	expected := fmt.Sprintf("Server:  %s\nHealth:  ok\nReady:   ready\nVersion: v0.1.0\n", ct.url)
	assert.Equal(t, expected, res.stdout)
}

func TestStatusJSON(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", nil, "status", "--json")
	assert.Equal(t, 0, res.exit)
	assert.JSONEq(t, fmt.Sprintf(`{"server":"%s","health":"ok","ready":"ready","version":"v0.1.0"}`, ct.url), res.stdout)
}

func TestStatusUnreachable(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(&config.Config{Auth: config.Auth{Server: "http://127.0.0.1:1"}}, "", nil, "status")
	assert.Equal(t, 1, res.exit)
	assert.Contains(t, res.stderr, "health check failed")
}

func TestVersionCommand(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(nil, "", nil, "version")
	assert.Equal(t, 0, res.exit)
	assert.Equal(t, "markpost-cli version test-version\n", res.stdout)
}

func TestBareInvocationShowsHelpOnStdout(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(nil, "", nil)
	assert.Equal(t, 0, res.exit)
	assert.Contains(t, res.stdout, "USAGE:")
	assert.Empty(t, res.stderr)
}

func TestUnknownCommand(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(nil, "", nil, "frobnicate")
	assert.Equal(t, 1, res.exit)
	assert.Equal(t, "markpost: unknown command \"frobnicate\" for \"markpost\"\n", res.stderr)
}

func TestUnknownSubcommand(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(nil, "", nil, "posts", "frobnicate")
	assert.Equal(t, 1, res.exit)
	assert.Equal(t, "markpost: unknown command \"frobnicate\" for \"posts\"\n", res.stderr)
}

func TestBadFlag(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(nil, "", nil, "posts", "list", "--frob")
	assert.Equal(t, 1, res.exit)
	assert.Contains(t, res.stderr, "markpost: flag provided but not defined")
}

func TestAgentGetsFullHelpOnUsageError(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(nil, "", map[string]string{"AI_AGENT": "claude-code"}, "posts", "list", "--frob")
	assert.Equal(t, 1, res.exit)
	assert.Contains(t, res.stderr, "flag provided but not defined")
	assert.Contains(t, res.stderr, "COMMANDS:", "an agent gets the full command list, not a terse usage line")
}

// --- session behaviors ---

func TestRefreshPersistsNewTokens(t *testing.T) {
	ct := newCLITest(t)
	stale := ct.loggedInCfg()
	stale.Auth.Token = "at-stale" // refresh token rt-1 stays valid
	res := ct.run(stale, "", nil, "posts", "list")
	assert.Equal(t, 0, res.exit, "stderr: %s", res.stderr)

	cfg := ct.loadCfg()
	assert.Equal(t, "at-2", cfg.Auth.Token, "the refreshed pair must be persisted")
	assert.Equal(t, "rt-2", cfg.Auth.RefreshToken)
	assert.Equal(t, 1, ct.fake.RefreshCalls())
}

func TestServerMismatchDropsStoredToken(t *testing.T) {
	ct := newCLITest(t)
	cfg := ct.loggedInCfg()
	cfg.Auth.Server = "http://elsewhere:9"
	res := ct.run(cfg, "", map[string]string{"MARKPOST_SERVER": ct.url}, "posts", "list")
	assert.Equal(t, 4, res.exit)
	assert.Contains(t, res.stderr, "not logged in", "a session must not leak to another server")
}

func TestEnvTokenSessionWorks(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(ct.serverOnlyCfg(), "", map[string]string{"MARKPOST_TOKEN": "at-1"}, "posts", "list")
	assert.Equal(t, 0, res.exit)
	assert.Contains(t, res.stdout, "p-existing")
}

func TestNoServerExitCode(t *testing.T) {
	ct := newCLITest(t)
	res := ct.run(nil, "", nil, "status")
	assert.Equal(t, 4, res.exit)
	assert.Contains(t, res.stderr, "no server configured")
}
