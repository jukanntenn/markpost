import { renderHook, act, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { useCopyToClipboard } from "./useCopyToClipboard";

describe("useCopyToClipboard", () => {
  let writeTextSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    Object.assign(navigator, {
      clipboard: { writeText: vi.fn().mockResolvedValue(undefined) },
    });
    writeTextSpy = vi.mocked(navigator.clipboard.writeText);
  });

  it("returns initial state with copied=false and a copy function", () => {
    const { result } = renderHook(() => useCopyToClipboard());
    expect(result.current.copied).toBe(false);
    expect(typeof result.current.copy).toBe("function");
  });

  it("sets copied to true after successful copy", async () => {
    const { result } = renderHook(() => useCopyToClipboard());

    await act(async () => {
      await result.current.copy("hello");
    });

    await waitFor(() => {
      expect(result.current.copied).toBe(true);
    });
    expect(writeTextSpy).toHaveBeenCalledWith("hello");
  });

  it("resets copied to false after default delay", async () => {
    vi.useFakeTimers();
    const { result } = renderHook(() => useCopyToClipboard());

    await act(async () => {
      await result.current.copy("hello");
    });
    expect(result.current.copied).toBe(true);

    act(() => {
      vi.advanceTimersByTime(2000);
    });
    expect(result.current.copied).toBe(false);

    vi.useRealTimers();
  });

  it("respects custom resetDelay", async () => {
    vi.useFakeTimers();
    const { result } = renderHook(() => useCopyToClipboard(1000));

    await act(async () => {
      await result.current.copy("hello");
    });
    expect(result.current.copied).toBe(true);

    act(() => {
      vi.advanceTimersByTime(999);
    });
    expect(result.current.copied).toBe(true);

    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(result.current.copied).toBe(false);

    vi.useRealTimers();
  });

  it("keeps copied false when clipboard write fails", async () => {
    writeTextSpy.mockRejectedValue(new Error("denied"));

    const { result } = renderHook(() => useCopyToClipboard());

    await act(async () => {
      await result.current.copy("hello");
    });

    expect(result.current.copied).toBe(false);
  });
});
