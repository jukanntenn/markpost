//go:build e2e

// End-to-end tests: a real markpost backend (built from ../backend against a
// postgres testcontainer) driven through the real markpost-mcp binary over
// stdio — the same shape as the golden reference's docker-image e2e, with the
// "live API" being a locally started backend instead of api.github.com.
//
// Run: go test --tags e2e ./e2e
// Skip when docker is unavailable: TESTCONTAINERS_SKIP=1 go test --tags e2e ./e2e

package e2e

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	adminUser     = "e2e_admin"
	adminPassword = "e2e-admin-pass-123"
	backendPort   = "37330"
)

func TestMarkpostMCPEndToEnd(t *testing.T) {
	ctx := context.Background()

	pg := startPostgres(t, ctx)
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	serverBin := buildBinary(t, "../../backend", "./cmd/server", "markpost-server")
	mcpBin := buildBinary(t, "..", "./cmd/markpost-mcp", "markpost-mcp")

	runBackend(t, serverBin, dsn, "migrate", "up")
	runBackend(t, serverBin, dsn, "seed-users", "--count", "1", "--prefix", adminUser, "--password", adminPassword, "--channels", "0")
	promoteToAdmin(t, ctx, pg)

	baseURL := startBackend(t, serverBin, dsn)
	session := startMCPStdio(t, mcpBin, baseURL)

	t.Run("create_get_list_delete_post", func(t *testing.T) {
		res := callTool(t, session, "create_post", map[string]any{
			"title": "E2E Hello",
			"body":  "# Heading\n\nSome **bold** markdown.",
		})
		require.False(t, res.IsError, text(t, res))

		var created struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		}
		require.NoError(t, json.Unmarshal([]byte(text(t, res)), &created))
		require.NotEmpty(t, created.ID)
		assert.Equal(t, baseURL+"/"+created.ID, created.URL)

		getRes := callTool(t, session, "get_post", map[string]any{"qid": created.ID})
		assert.False(t, getRes.IsError)
		assert.Equal(t, "# E2E Hello\n\n# Heading\n\nSome **bold** markdown.", text(t, getRes))

		listRes := callTool(t, session, "list_posts", map[string]any{})
		assert.False(t, listRes.IsError)
		assert.Contains(t, text(t, listRes), created.ID)

		delRes := callTool(t, session, "delete_post", map[string]any{"qid": created.ID})
		assert.False(t, delRes.IsError)
		assert.JSONEq(t, `{"deleted": true}`, text(t, delRes))

		goneRes := callTool(t, session, "get_post", map[string]any{"qid": created.ID})
		assert.True(t, goneRes.IsError, "deleted post must surface as a tool error")
		assert.Contains(t, text(t, goneRes), "404")
	})

	t.Run("account surface", func(t *testing.T) {
		res := callTool(t, session, "get_my_retention", map[string]any{})
		assert.False(t, res.IsError, text(t, res))
		assert.Contains(t, text(t, res), "posts_days")

		sessions := callTool(t, session, "list_my_sessions", map[string]any{})
		assert.False(t, sessions.IsError, text(t, sessions))
		assert.Contains(t, text(t, sessions), "sessions")
	})

	t.Run("delivery surface", func(t *testing.T) {
		for _, tool := range []string{"list_channels", "list_delivery_history", "list_latest_deliveries", "get_delivery_stats", "list_pending_deliveries"} {
			res := callTool(t, session, tool, map[string]any{})
			assert.False(t, res.IsError, "%s: %s", tool, text(t, res))
		}
	})

	t.Run("admin surface", func(t *testing.T) {
		stats := callTool(t, session, "admin_get_stats", map[string]any{})
		assert.False(t, stats.IsError, text(t, stats))
		assert.Contains(t, text(t, stats), "counts")

		users := callTool(t, session, "admin_list_users", map[string]any{"search": adminUser})
		assert.False(t, users.IsError, text(t, users))
		assert.Contains(t, text(t, users), adminUser+"_1")

		created := callTool(t, session, "admin_create_user", map[string]any{
			"username": "e2e_secondary", "password": "e2e-secondary-pass",
		})
		assert.False(t, created.IsError, text(t, created))

		var newUser struct {
			ID int `json:"id"`
		}
		require.NoError(t, json.Unmarshal([]byte(text(t, created)), &newUser))
		require.Positive(t, newUser.ID)

		impact := callTool(t, session, "admin_retention_impact", map[string]any{
			"user_ids": []int{newUser.ID}, "retention_days": 30,
		})
		assert.False(t, impact.IsError, text(t, impact))

		setRetention := callTool(t, session, "admin_set_user_retention", map[string]any{
			"user_id": newUser.ID, "retention_days": 30,
		})
		assert.False(t, setRetention.IsError, text(t, setRetention))

		audit := callTool(t, session, "admin_list_audit_logs", map[string]any{"action": "user.create"})
		assert.False(t, audit.IsError, text(t, audit))

		del := callTool(t, session, "admin_delete_user", map[string]any{"user_id": newUser.ID})
		assert.False(t, del.IsError, text(t, del))
	})
}

// startPostgres mirrors the backend's container conventions (see
// backend/internal/infra/testdb.go): postgres:17-alpine, reaper disabled,
// TESTCONTAINERS_SKIP for environments without docker.
func startPostgres(t *testing.T, ctx context.Context) *tcpostgres.PostgresContainer {
	t.Helper()
	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	pg, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("markpost_e2e"),
		tcpostgres.WithUsername("markpost"),
		tcpostgres.WithPassword("markpost"),
	)
	if err != nil {
		if os.Getenv("TESTCONTAINERS_SKIP") != "" {
			t.Skipf("testcontainers unavailable: %v", err)
		}
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = pg.Terminate(shutdownCtx)
	})
	return pg
}

// buildBinary compiles a binary from dir (module root, relative to mcp/e2e).
func buildBinary(t *testing.T, dir, pkg, name string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), name)
	cmd := exec.Command("go", "build", "-o", bin, pkg)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", name, err, out)
	}
	return bin
}

func backendEnv(dsn string) []string {
	return []string{
		"MARKPOST_DB__DSN=" + dsn,
		"MARKPOST_JWT__ACCESS_SIGNING_KEY=e2e-access-signing-key-0123456789abcdef",
		"MARKPOST_JWT__REFRESH_SIGNING_KEY=e2e-refresh-signing-key-0123456789abcdef",
		"MARKPOST_SERVER__HOST=127.0.0.1",
		"MARKPOST_SERVER__PORT=" + backendPort,
		"MARKPOST_SERVER__PUBLIC_URL=http://127.0.0.1:" + backendPort,
	}
}

// runBackend runs a one-shot backend CLI subcommand in a clean cwd (the
// worktree's backend/config.toml must not leak into the e2e server).
func runBackend(t *testing.T, bin, dsn string, args ...string) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), backendEnv(dsn)...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s: %s", args, out)
}

// promoteToAdmin flips the seeded user's role — the CLI seeds regular users,
// and the admin surface needs the flip (one SQL statement in the container).
func promoteToAdmin(t *testing.T, ctx context.Context, pg *tcpostgres.PostgresContainer) {
	t.Helper()
	exitCode, reader, err := pg.Exec(ctx, []string{
		"psql", "-U", "markpost", "-d", "markpost_e2e", "-c",
		"UPDATE users SET role = 'admin' WHERE username = '" + adminUser + "_1'",
	})
	require.NoError(t, err)
	body, _ := io.ReadAll(reader)
	require.Zero(t, exitCode, "psql promote: %s", body)
}

// backendRuntimeDir prepares a cwd the backend can serve from: templates/
// and locales/ are loaded from the working directory (see backend
// cmd/server/main.go LoadHTMLGlob / i18n RootPath), while the backend tree's
// own config.toml must NOT be picked up — so a clean dir plus symlinks.
func backendRuntimeDir(t *testing.T) string {
	t.Helper()
	backendDir, err := filepath.Abs(filepath.Join("..", "..", "backend"))
	require.NoError(t, err)
	dir := t.TempDir()
	for _, sub := range []string{"templates", "locales"} {
		require.NoError(t, os.Symlink(filepath.Join(backendDir, sub), filepath.Join(dir, sub)))
	}
	return dir
}

func startBackend(t *testing.T, bin, dsn string) string {
	t.Helper()
	cmd := exec.Command(bin, "serve")
	cmd.Dir = backendRuntimeDir(t)
	cmd.Env = append(os.Environ(), backendEnv(dsn)...)
	cmd.Stdout = io.Discard
	var stderr strings.Builder
	cmd.Stderr = &stderr
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	baseURL := "http://127.0.0.1:" + backendPort
	require.Eventually(t, func() bool {
		res, err := http.Get(baseURL + "/api/v1/health")
		if err != nil {
			return false
		}
		defer res.Body.Close()
		return res.StatusCode == http.StatusOK
	}, 30*time.Second, 250*time.Millisecond, "backend did not become healthy; server logs:\n%s", stderr.String())
	return baseURL
}

// startMCPStdio launches the markpost-mcp binary and connects an MCP client
// to its stdio transport. The binary logs to stderr only — stdout carries the
// protocol.
func startMCPStdio(t *testing.T, bin, baseURL string) *mcp.ClientSession {
	t.Helper()

	cmd := exec.Command(bin, "stdio", "--url", baseURL, "--toolsets", "all")
	cmd.Env = append(os.Environ(),
		"MARKPOST_MCP_USERNAME="+adminUser+"_1",
		"MARKPOST_MCP_PASSWORD="+adminPassword,
	)
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	c := mcp.NewClient(&mcp.Implementation{Name: "e2e-client", Version: "test"}, nil)
	session, err := c.Connect(context.Background(), &mcp.IOTransport{Reader: stdout, Writer: stdin}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err, "protocol failure calling %s", name)
	return res
}

// text extracts (and logs) a result's text content.
func text(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	out := strings.TrimSpace(sb.String())
	t.Logf("→ %s", out)
	return out
}
