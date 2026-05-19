import { describe, expect, it } from "vitest";

import { buildUrl, paginationParams, ApiError } from "./base";

describe("buildUrl", () => {
  it("returns base + path when no params", () => {
    expect(buildUrl("", "/api/v1/posts")).toBe("/api/v1/posts");
  });

  it("returns base + path with non-empty base", () => {
    expect(buildUrl("https://api.example.com", "/api/v1/posts")).toBe(
      "https://api.example.com/api/v1/posts"
    );
  });

  it("appends params as query string", () => {
    expect(buildUrl("", "/api/v1/posts", { page: 1, limit: 20 })).toBe(
      "/api/v1/posts?page=1&limit=20"
    );
  });

  it("stringifies number param values", () => {
    const result = buildUrl("", "/api", { count: 42 });
    expect(result).toContain("count=42");
  });

  it("does not append ? when params is undefined", () => {
    expect(buildUrl("", "/api/v1/posts", undefined)).toBe("/api/v1/posts");
  });

  it("does not append ? when params is empty object", () => {
    expect(buildUrl("", "/api/v1/posts", {})).toBe("/api/v1/posts");
  });

  it("strips trailing slash from base", () => {
    expect(buildUrl("https://api.example.com/", "/api/v1/posts")).toBe(
      "https://api.example.com/api/v1/posts"
    );
  });

  it("strips trailing slash from base with params", () => {
    expect(buildUrl("https://api.example.com/", "/api", { page: 1 })).toBe(
      "https://api.example.com/api?page=1"
    );
  });

  it("URL-encodes special characters in values", () => {
    const result = buildUrl("", "/api", { q: "hello world" });
    expect(result).toContain("q=hello+world");
  });
});

describe("paginationParams", () => {
  it("returns empty object when no args", () => {
    expect(paginationParams()).toEqual({});
  });

  it("includes page when provided", () => {
    expect(paginationParams(1)).toEqual({ page: 1 });
  });

  it("includes limit when provided", () => {
    expect(paginationParams(undefined, 20)).toEqual({ limit: 20 });
  });

  it("includes both when provided", () => {
    expect(paginationParams(2, 10)).toEqual({ page: 2, limit: 10 });
  });

  it("includes zero values", () => {
    expect(paginationParams(0, 0)).toEqual({ page: 0, limit: 0 });
  });
});

describe("ApiError", () => {
  it("sets message from response", () => {
    const err = new ApiError({ message: "Unauthorized" });
    expect(err.message).toBe("Unauthorized");
  });

  it("falls back to 'Request failed' when message is missing", () => {
    const err = new ApiError({});
    expect(err.message).toBe("Request failed");
  });

  it("sets code from response", () => {
    const err = new ApiError({ code: "unauthorized", message: "nope" });
    expect(err.code).toBe("unauthorized");
  });

  it("sets fieldErrors from response", () => {
    const err = new ApiError({
      errors: [{ field: "title", code: "required", message: "missing" }],
    });
    expect(err.fieldErrors).toEqual([
      { field: "title", code: "required", message: "missing" },
    ]);
  });

  it("has name 'ApiError'", () => {
    const err = new ApiError({ message: "x" });
    expect(err.name).toBe("ApiError");
  });

  it("defaults code and fieldErrors to undefined", () => {
    const err = new ApiError({ message: "x" });
    expect(err.code).toBeUndefined();
    expect(err.fieldErrors).toBeUndefined();
  });
});
