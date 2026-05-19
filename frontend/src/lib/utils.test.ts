import { describe, expect, it } from "vitest";
import { truncate } from "./utils";

describe("truncate", () => {
  it("returns string unchanged when shorter than max", () => {
    expect(truncate("hi", 5)).toBe("hi");
  });

  it("returns string unchanged when equal to max", () => {
    expect(truncate("hello", 5)).toBe("hello");
  });

  it("truncates and appends ellipsis when longer", () => {
    expect(truncate("hello world", 5)).toBe("hello...");
  });

  it("handles empty string", () => {
    expect(truncate("", 3)).toBe("");
  });

  it("handles max of 0", () => {
    expect(truncate("abc", 0)).toBe("...");
  });
});
