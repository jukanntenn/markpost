import { describe, it, expect } from "vitest";
import fs from "node:fs";
import path from "node:path";
import { ApiErrorCodes } from "./error-codes";

const BACKEND_ERRORS_PATH = path.resolve(
  __dirname,
  "../../../../backend/internal/service/errors.go",
);

function extractErrorCodes(source: string): Set<string> {
  const matches = source.matchAll(/ErrCode\s*=\s*"([^"]+)"/g);
  return new Set([...matches].map((m) => m[1]));
}

describe("ApiErrorCodes sync with backend", () => {
  const source = fs.readFileSync(BACKEND_ERRORS_PATH, "utf-8");
  const backendCodes = extractErrorCodes(source);
  const frontendCodes = Object.values(ApiErrorCodes);

  it("has no frontend codes missing from the backend", () => {
    const missing = frontendCodes.filter((c) => !backendCodes.has(c));
    expect(missing, `Frontend codes not in backend: ${missing.join(", ")}`).toEqual([]);
  });

  it("has no backend codes missing from the frontend", () => {
    const missing = Array.from(backendCodes).filter((c) => !frontendCodes.includes(c));
    expect(missing, `Backend codes not in frontend: ${missing.join(", ")}`).toEqual([]);
  });
});
