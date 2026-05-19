import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useDebouncedValue } from "./useDebouncedValue";

describe("useDebouncedValue", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns the initial value immediately", () => {
    const { result } = renderHook(
      ({ value, delay }) => useDebouncedValue(value, delay),
      { initialProps: { value: "hello", delay: 500 } },
    );
    expect(result.current).toBe("hello");
  });

  it("updates after the delay elapses", () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useDebouncedValue(value, delay),
      { initialProps: { value: "hello", delay: 500 } },
    );

    rerender({ value: "world", delay: 500 });
    act(() => {
      vi.advanceTimersByTime(500);
    });

    expect(result.current).toBe("world");
  });

  it("does not update before the delay elapses", () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useDebouncedValue(value, delay),
      { initialProps: { value: "hello", delay: 500 } },
    );

    rerender({ value: "world", delay: 500 });
    act(() => {
      vi.advanceTimersByTime(499);
    });

    expect(result.current).toBe("hello");
  });

  it("resets the timer on rapid changes", () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useDebouncedValue(value, delay),
      { initialProps: { value: "hello", delay: 500 } },
    );

    rerender({ value: "a", delay: 500 });
    act(() => {
      vi.advanceTimersByTime(300);
    });

    rerender({ value: "b", delay: 500 });
    act(() => {
      vi.advanceTimersByTime(300);
    });

    expect(result.current).toBe("hello");

    act(() => {
      vi.advanceTimersByTime(200);
    });
    expect(result.current).toBe("b");
  });
});
