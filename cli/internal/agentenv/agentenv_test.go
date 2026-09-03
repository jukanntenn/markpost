package agentenv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"no agent", nil, ""},
		{"generic AI_AGENT", map[string]string{"AI_AGENT": "my-agent"}, "my-agent"},
		{"AI_AGENT invalid name falls through", map[string]string{"AI_AGENT": "not a name!", "CLAUDECODE": "1"}, "claude-code"},
		{"amp via AGENT", map[string]string{"AGENT": "amp"}, "amp"},
		{"codex", map[string]string{"CODEX_SANDBOX": "seatbelt"}, "codex"},
		{"gemini", map[string]string{"GEMINI_CLI": "1"}, "gemini-cli"},
		{"copilot", map[string]string{"COPILOT_CLI": "1"}, "copilot-cli"},
		{"opencode", map[string]string{"OPENCODE": "1"}, "opencode"},
		{"cowork beats claude-code", map[string]string{"CLAUDECODE": "1", "CLAUDE_CODE_IS_COWORK": "1"}, "cowork"},
		{"claude code", map[string]string{"CLAUDECODE": "1"}, "claude-code"},
		{"cursor", map[string]string{"CURSOR_TRACE_ID": "x"}, "cursor"},
		{"cursor cli", map[string]string{"CURSOR_AGENT": "1"}, "cursor-cli"},
		{"empty value does not count", map[string]string{"CLAUDECODE": ""}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectWith(func(key string) (string, bool) { return tt.env[key], true })
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUserAgentSuffix(t *testing.T) {
	assert.Equal(t, "", UserAgentSuffix(""))
	assert.Equal(t, " Agent/claude-code", UserAgentSuffix("claude-code"))
}
