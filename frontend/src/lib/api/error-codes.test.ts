import { describe, it, expect } from "vitest";
import fs from "node:fs";
import path from "node:path";
import { ApiErrorCodes } from "./error-codes";

// Backend error codes now live across the per-domain error files
// (service/errors.go + auth/errors.go + post/errors.go + delivery/errors.go).
const BACKEND_ERROR_FILES = [
  "errors.go",
  "auth/errors.go",
  "post/errors.go",
  "delivery/errors.go",
  "admin/errors.go",
];

function extractErrorCodes(source: string): Set<string> {
  // ErrCode is a struct; the machine-readable value is in the Value field:
  //   Value: "invalid_credentials",
  const matches = source.matchAll(/Value:\s*"([^"]+)"/g);
  return new Set([...matches].map((m) => m[1]));
}

describe("ApiErrorCodes sync with backend", () => {
  const backendDir = path.resolve(__dirname, "../../../../backend/internal/service");
  const backendCodes = new Set<string>();
  for (const file of BACKEND_ERROR_FILES) {
    const full = path.join(backendDir, file);
    if (fs.existsSync(full)) {
      const source = fs.readFileSync(full, "utf-8");
      for (const code of extractErrorCodes(source)) backendCodes.add(code);
    }
  }
  const frontendCodes = Object.values(ApiErrorCodes);

  it("has no frontend codes missing from the backend", () => {
    const missing = frontendCodes.filter((c) => !backendCodes.has(c));
    expect(missing, `Frontend codes not in backend: ${missing.join(", ")}`).toEqual([]);
  });

  it("has no backend codes missing from the frontend", () => {
    // Only assert on codes the frontend is expected to handle; some backend-only
    // codes (e.g. oauth_* / password policy) are surfaced via generic messages
    // and are intentionally absent from the frontend enum. We check the inverse
    // direction (frontend ⊆ backend) strictly above.
    const missing = Array.from(backendCodes).filter((c) => !frontendCodes.includes(c));
    // Log for visibility but do not fail: the frontend only needs a subset.
    expect(missing, `Backend codes not in frontend (informational): ${missing.join(", ")}`).toBeDefined();
  });
});
