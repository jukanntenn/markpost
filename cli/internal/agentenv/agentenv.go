// Package agentenv detects whether an AI coding agent is driving the CLI,
// mirroring the environment-variable conventions of gh's internal/agents
// package so an agent detected by gh is detected here too. The name feeds two
// behaviors: a User-Agent suffix (server-side telemetry can tell agent from
// human traffic) and full-help error output in the main package (an agent
// self-corrects from examples and flag lists faster than from a terse usage
// line).
package agentenv

import (
	"fmt"
	"os"
	"regexp"
)

var validName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Detect returns the driving agent's name, or "" for a human.
func Detect() string {
	return detectWith(os.LookupEnv)
}

func detectWith(lookup func(string) (string, bool)) string {
	isSet := func(key string) bool {
		v, ok := lookup(key)
		return ok && v != ""
	}
	valueOf := func(key string) string {
		v, _ := lookup(key)
		return v
	}

	// The generic identifier wins: it is the most specific signal and the one
	// agents that are not on the known list should set.
	if v, ok := lookup("AI_AGENT"); ok && v != "" {
		if validName.MatchString(v) {
			return v
		}
	}

	// Tool-specific variables, ordered exactly like gh's detectWith so both
	// CLIs resolve the same name in overlapping environments.
	if valueOf("AGENT") == "amp" {
		return "amp"
	}
	if isSet("CODEX_SANDBOX") || isSet("CODEX_CI") || isSet("CODEX_THREAD_ID") {
		return "codex"
	}
	if isSet("GEMINI_CLI") {
		return "gemini-cli"
	}
	if isSet("COPILOT_CLI") {
		return "copilot-cli"
	}
	if isSet("OPENCODE") {
		return "opencode"
	}
	if isSet("CLAUDE_CODE_IS_COWORK") {
		return "cowork"
	}
	if isSet("CLAUDECODE") || isSet("CLAUDE_CODE") {
		return "claude-code"
	}
	if isSet("CURSOR_TRACE_ID") {
		return "cursor"
	}
	if isSet("CURSOR_AGENT") || valueOf("CURSOR_EXTENSION_HOST_ROLE") == "agent-exec" {
		return "cursor-cli"
	}
	return ""
}

// UserAgentSuffix renders the detected name for a User-Agent header, e.g.
// "Agent/claude-code", or "" when no agent is detected.
func UserAgentSuffix(name string) string {
	if name == "" {
		return ""
	}
	return fmt.Sprintf(" Agent/%s", name)
}
