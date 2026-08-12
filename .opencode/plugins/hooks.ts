/**
 * OpenCode plugin: PostToolUse formatter.
 *
 * Delegates to prek (the fmt group, single source of truth) — no formatter
 * logic lives here, so it can never drift from prek/CI. prek discovers the
 * workspace root from cwd and routes the file to the right project formatter.
 *
 * Plugin SDK: packages/plugin/src/index.ts (tool.execute.after)
 * Write/edit tool args: { filePath: string }
 *
 * The Stop-hook lint gate cannot be replicated: OpenCode event hooks are
 * fire-and-forget with no channel to feed a block decision back into the
 * session. Lint enforcement therefore falls back to prek's pre-commit gate
 * and CI.
 */
export const HooksPlugin = async ({ $ }) => {
  return {
    "tool.execute.after": async (input, output) => {
      if (input.tool !== "write" && input.tool !== "edit") return;

      const filePath = output.args?.filePath ?? output.args?.file_path;
      if (!filePath) return;

      // Best-effort: format via prek, never block the session.
      await $`prek run --group fmt --files ${filePath}`.quiet().nothrow();
    },
  };
};
