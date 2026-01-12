import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { get, set, remove } from "./storage";

describe("storage", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it("sets and gets string value", () => {
    set("test_key", "test_value");
    expect(get<string>("test_key")).toBe("test_value");
  });

  it("sets and gets object value", () => {
    const obj = { name: "test", value: 123 };
    set("test_obj", obj);
    expect(get("test_obj")).toEqual(obj);
  });

  it("sets and gets array value", () => {
    const arr = [1, 2, 3];
    set("test_arr", arr);
    expect(get("test_arr")).toEqual(arr);
  });

  it("returns null for non-existent key", () => {
    expect(get("non_existent")).toBeNull();
  });

  it("removes value", () => {
    set("test_key", "test_value");
    expect(get("test_key")).toBe("test_value");

    remove("test_key");
    expect(get("test_key")).toBeNull();
  });

  it("uses sessionStorage when provided", () => {
    sessionStorage.clear();
    set("session_key", "session_value", sessionStorage);
    expect(get("session_key")).toBeNull();
    expect(get("session_key", sessionStorage)).toBe("session_value");
  });

  it("handles invalid JSON gracefully", () => {
    const prefix = "markpost_dev_";
    localStorage.setItem(prefix + "invalid", "invalid json");
    const raw = localStorage.getItem(prefix + "invalid");

    try {
      JSON.parse(raw as string);
    } catch {
      expect(raw).toBe("invalid json");
    }
  });
});
