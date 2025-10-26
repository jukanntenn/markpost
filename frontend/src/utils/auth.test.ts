import { describe, it, expect } from "vitest";
import { checkLoginResponse } from "./auth";
import type { LoginResponse } from "../types/auth";

describe("checkLoginResponse", () => {
  it("should return true when all fields are valid", () => {
    const validResponse: LoginResponse = {
      access_token: "test_token",
      refresh_token: "test_refresh_token",
      user: {
        id: 1,
        username: "testuser",
      },
    };

    expect(checkLoginResponse(validResponse)).toBe(true);
  });

  it("should return false when data is null", () => {
    expect(checkLoginResponse(null)).toBe(false);
  });

  it("should return false when access_token is missing", () => {
    const response: LoginResponse = {
      access_token: "",
      refresh_token: "test_refresh_token",
      user: {
        id: 1,
        username: "testuser",
      },
    };

    expect(checkLoginResponse(response)).toBe(false);
  });

  it("should return false when refresh_token is missing", () => {
    const response: LoginResponse = {
      access_token: "test_token",
      refresh_token: "",
      user: {
        id: 1,
        username: "testuser",
      },
    };

    expect(checkLoginResponse(response)).toBe(false);
  });

  it("should return false when user is missing", () => {
    const response = {
      access_token: "test_token",
      refresh_token: "test_refresh_token",
    } as any;

    expect(checkLoginResponse(response)).toBe(false);
  });

  it("should return false when user id is null", () => {
    const response: LoginResponse = {
      access_token: "test_token",
      refresh_token: "test_refresh_token",
      user: {
        id: null as any,
        username: "testuser",
      },
    };

    expect(checkLoginResponse(response)).toBe(false);
  });

  it("should return false when user username is empty", () => {
    const response: LoginResponse = {
      access_token: "test_token",
      refresh_token: "test_refresh_token",
      user: {
        id: 1,
        username: "",
      },
    };

    expect(checkLoginResponse(response)).toBe(false);
  });

  it("should return false when user username is missing", () => {
    const response = {
      access_token: "test_token",
      refresh_token: "test_refresh_token",
      user: {
        id: 1,
      },
    } as any;

    expect(checkLoginResponse(response)).toBe(false);
  });
});
