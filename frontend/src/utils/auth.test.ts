import { describe, it, expect } from "vitest";
import { checkLoginResponse } from "./auth";
import type { LoginResponse } from "@/lib/api/auth";

describe("checkLoginResponse", () => {
  it("should return true when all fields are valid", () => {
    const validResponse: LoginResponse = {
      token: "test_token",
      refresh_token: "test_refresh_token",
      expires_in: 86400,
      user: {
        id: 1,
        username: "testuser",
        email: "test@example.com",
      },
    };

    expect(checkLoginResponse(validResponse)).toBe(true);
  });

  it("should return false when data is null", () => {
    expect(checkLoginResponse(null)).toBe(false);
  });

  it("should return false when token is missing", () => {
    const response: LoginResponse = {
      token: "",
      refresh_token: "test_refresh_token",
      expires_in: 86400,
      user: {
        id: 1,
        username: "testuser",
        email: "test@example.com",
      },
    };

    expect(checkLoginResponse(response)).toBe(false);
  });

  it("should return false when refresh_token is missing", () => {
    const response: LoginResponse = {
      token: "test_token",
      refresh_token: "",
      expires_in: 86400,
      user: {
        id: 1,
        username: "testuser",
        email: "test@example.com",
      },
    };

    expect(checkLoginResponse(response)).toBe(false);
  });

  it("should return false when user is missing", () => {
    const response = {
      token: "test_token",
      refresh_token: "test_refresh_token",
    } as unknown as LoginResponse;

    expect(checkLoginResponse(response)).toBe(false);
  });

  it("should return false when user id is null", () => {
    const response = {
      token: "test_token",
      refresh_token: "test_refresh_token",
      user: {
        id: null,
        username: "testuser",
        email: "test@example.com",
      },
    } as unknown as LoginResponse;

    expect(checkLoginResponse(response)).toBe(false);
  });

  it("should return false when user username is empty", () => {
    const response: LoginResponse = {
      token: "test_token",
      refresh_token: "test_refresh_token",
      expires_in: 86400,
      user: {
        id: 1,
        username: "",
        email: "test@example.com",
      },
    };

    expect(checkLoginResponse(response)).toBe(false);
  });

  it("should return false when user username is missing", () => {
    const response = {
      token: "test_token",
      refresh_token: "test_refresh_token",
      user: {
        id: 1,
      },
    } as unknown as LoginResponse;

    expect(checkLoginResponse(response)).toBe(false);
  });
});
