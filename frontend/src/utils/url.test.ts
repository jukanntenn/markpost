import { describe, it, expect, vi } from "vitest";
import { buildPostUrl, buildFullPostUrl } from "./url";

describe("buildPostUrl", () => {
  it("returns /{qid} for a non-empty qid", () => {
    expect(buildPostUrl("p-abc123")).toBe("/p-abc123");
  });

  it("returns / for an empty qid", () => {
    expect(buildPostUrl("")).toBe("/");
  });
});

describe("buildFullPostUrl", () => {
  it("returns full URL with origin", () => {
    vi.stubGlobal("location", { origin: "https://example.com" });
    expect(buildFullPostUrl("p-abc123")).toBe(
      "https://example.com/p-abc123",
    );
    vi.unstubAllGlobals();
  });

  it('returns "" when window is undefined', () => {
    const originalWindow = globalThis.window;
    Object.defineProperty(globalThis, "window", {
      value: undefined,
      writable: true,
      configurable: true,
    });
    expect(buildFullPostUrl("p-abc123")).toBe("");
    globalThis.window = originalWindow;
  });
});
