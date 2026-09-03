package tools

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/jukanntenn/markpost/mcp/internal/markpost"
)

// The snapshot locks the tool surface (names, descriptions, annotations,
// input schemas) so any change to it is a deliberate, reviewed diff — the
// same protection the golden reference ships as internal/toolsnaps.
//
// Regenerate after an intentional change: go test ./internal/tools -run TestToolSnapshot -update

var updateSnapshot = flag.Bool("update", false, "rewrite tool snapshot files")

// snapshotEntry is one tool's stable projection.
type snapshotEntry struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Annotations *mcp.ToolAnnotations `json:"annotations,omitempty"`
	InputSchema json.RawMessage      `json:"input_schema"`
}

func TestToolSnapshot(t *testing.T) {
	client := markpost.NewClient("http://snapshot.invalid", "snap", "snap")
	s := mcp.NewServer(&mcp.Implementation{Name: "markpost-mcp-snapshot", Version: "test"}, nil)
	require.NoError(t, RegisterEnabled(s, client, AllNames(), false))

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := s.Connect(context.Background(), serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	c := mcp.NewClient(&mcp.Implementation{Name: "snapshot-client"}, nil)
	session, err := c.Connect(context.Background(), clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	res, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)

	entries := make([]snapshotEntry, 0, len(res.Tools))
	for _, tool := range res.Tools {
		schema, err := json.Marshal(tool.InputSchema)
		require.NoError(t, err)
		entries = append(entries, snapshotEntry{
			Name:        tool.Name,
			Description: tool.Description,
			Annotations: tool.Annotations,
			InputSchema: schema,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })

	got, err := json.MarshalIndent(entries, "", "  ")
	require.NoError(t, err)

	path := filepath.Join("testdata", "tools.json")
	if *updateSnapshot {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, append(got, '\n'), 0o644))
		return
	}

	want, err := os.ReadFile(path)
	require.NoError(t, err, "snapshot missing; regenerate with: go test ./internal/tools -run TestToolSnapshot -update")
	require.JSONEq(t, string(want), string(got), "tool surface changed; if intentional, regenerate with -update")

	var names []string
	for _, e := range entries {
		names = append(names, e.Name)
	}
	require.Len(t, names, 47, "expected 47 tools: 4 posts + 9 delivery + 6 account + 28 admin")
}
