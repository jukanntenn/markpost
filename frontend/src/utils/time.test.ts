import { describe, expect, it } from "vitest";
import { formatToLocalTime } from "./time";

describe("formatToLocalTime", () => {
  it("formats a UTC ISO string with seconds by default", () => {
    const result = formatToLocalTime("2025-06-15T14:30:45Z");
    expect(result).toMatch(/\d{2}\/\d{2}\/\d{4}, \d{2}:\d{2}:\d{2} [AP]M$/);
  });

  it("formats without seconds when includeSeconds is false", () => {
    const result = formatToLocalTime("2025-06-15T14:30:45Z", { includeSeconds: false });
    expect(result).toMatch(/\d{2}\/\d{2}\/\d{4}, \d{2}:\d{2} [AP]M$/);
    const timePart = result.split(", ")[1];
    expect(timePart.split(":")).toHaveLength(2);
  });

  it("defaults to en locale", () => {
    const result = formatToLocalTime("2025-06-15T14:30:45Z");
    expect(result).toMatch(/\d{2}\/\d{2}\/\d{4}, \d{2}:\d{2}:\d{2} [AP]M$/);
  });

  it("uses the provided locale", () => {
    const result = formatToLocalTime("2025-06-15T14:30:45Z", { locale: "en-US" });
    expect(result).toMatch(/\d{2}\/\d{2}\/\d{4}/);
  });

  it("defaults to en when locale is not provided in object form", () => {
    const result = formatToLocalTime("2025-06-15T14:30:45Z", { includeSeconds: true });
    expect(result).toMatch(/\d{2}\/\d{2}\/\d{4}, \d{2}:\d{2}:\d{2} [AP]M$/);
  });

  it("returns empty string for empty input", () => {
    expect(formatToLocalTime("")).toBe("");
  });

  it("zero-pads single-digit month, day, hour, minute, second", () => {
    const result = formatToLocalTime("2025-01-05T09:05:03Z");
    const [datePart, timePart] = result.split(", ");
    expect(datePart).toMatch(/^\d{2}\/\d{2}\/\d{4}$/);
    expect(timePart).toMatch(/^\d{2}:\d{2}:\d{2} [AP]M$/);
  });

  it("returns empty string for malformed input", () => {
    expect(formatToLocalTime("not-a-date")).toBe("");
  });

  it("handles a date at midnight local time", () => {
    const date = new Date();
    date.setHours(0, 0, 0, 0);
    const result = formatToLocalTime(date.toISOString());
    const timePart = result.split(", ")[1];
    expect(timePart).toBe("12:00:00 AM");
  });

  it("handles includeSeconds as object", () => {
    const result = formatToLocalTime("2025-06-15T14:30:45Z", { includeSeconds: true });
    expect(result).toMatch(/\d{2}\/\d{2}\/\d{4}, \d{2}:\d{2}:\d{2} [AP]M$/);
  });
});
