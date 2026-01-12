import { describe, it, expect, beforeEach, vi } from "vitest";
import { getAcceptLanguageHeader } from "./i18n";

vi.mock("../i18n", () => ({
  default: {
    resolvedLanguage: "en",
    language: "en",
  },
}));

describe("i18n", () => {
  beforeEach(() => {
    vi.resetModules();
  });

  it("returns English header by default", () => {
    const header = getAcceptLanguageHeader();
    expect(header).toBe("en-US,en;q=0.9");
  });
});
