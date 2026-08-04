/**
 * OpenCode plugin: PostToolUse formatter (mirrors .claude/hooks/format.py).
 *
 * OpenCode has no declarative hooks system; its Plugin SDK is the only way to
 * intercept tool execution. This plugin replicates the PostToolUse formatter
 * behavior by listening to `tool.execute.after` for write/edit calls and
 * delegating to the existing Python formatter script.
 *
 * The Stop-hook linter (lint_stop.py) cannot be replicated because OpenCode's
 * event hooks are fire-and-forget — there is no channel to feed a block
 * decision back into the session lifecycle. Lint enforcement falls back to
 * the prek pre-commit gate in that case.
 *
 * Plugin SDK: packages/plugin/src/index.ts (tool.execute.after)
 * Write/edit tool args: { filePath: string } (packages/opencode/src/tool/write.ts)
 */
const FORMAT_SCRIPT = ".claude/hooks/format.py";

export const HooksPlugin = async ({ $ }) => {
  return {
    "tool.execute.after": async (input, output) => {
      if (input.tool !== "write" && input.tool !== "edit") return;

      const filePath = output.args?.filePath ?? output.args?.file_path;
      if (!filePath) return;

      // format.py reads {"tool_input": {"file_path": "..."}} from stdin.
      const payload = JSON.stringify({
        tool_input: { file_path: filePath },
      });

      const proc = $`python3 ${FORMAT_SCRIPT}`.quiet().nothrow();
      const writer = proc.stdin.getWriter();
      await writer.write(payload);
      await writer.close();
      await proc;
    },
  };
};
