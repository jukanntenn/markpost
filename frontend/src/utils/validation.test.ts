import { describe, expect, it } from "vitest";
import { validatePasswordChange } from "./validation";

describe("validatePasswordChange", () => {
  const messages = { notMatch: "Passwords do not match", minLength: "Password too short" };

  it("returns null when passwords match and meet minimum length", () => {
    expect(validatePasswordChange({ newPassword: "secret123", confirmPassword: "secret123" }, messages)).toBeNull();
  });

  it("returns the notMatch message when passwords differ", () => {
    expect(validatePasswordChange({ newPassword: "secret123", confirmPassword: "different" }, messages)).toBe("Passwords do not match");
  });

  it("returns the minLength message when password is too short", () => {
    expect(validatePasswordChange({ newPassword: "abc", confirmPassword: "abc" }, messages)).toBe("Password too short");
  });

  it("checks notMatch before minLength", () => {
    expect(validatePasswordChange({ newPassword: "abc", confirmPassword: "xyz" }, messages)).toBe("Passwords do not match");
  });
});
