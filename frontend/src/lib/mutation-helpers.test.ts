import { describe, it, expect, vi, beforeEach } from "vitest";

import { mutationOptions, mutationSuccess, setErrorOnError } from "./mutation-helpers";

vi.mock("@/stores/toast", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    warning: vi.fn(),
  },
}));

import { toast } from "@/stores/toast";

describe("mutationOptions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns options with onError that calls toast.error", () => {
    const mutationFn = vi.fn();
    const result = mutationOptions({ mutationFn });
    result.onError!(new Error("test error"), undefined, undefined);
    expect(toast.error).toHaveBeenCalledWith("test error");
    expect(result.mutationFn).toBe(mutationFn);
  });

  it("uses custom onError when provided instead of toast", () => {
    const customOnError = vi.fn();
    const result = mutationOptions({ mutationFn: vi.fn(), onError: customOnError });
    result.onError!(new Error("custom error"), undefined, undefined);
    expect(customOnError).toHaveBeenCalledWith(new Error("custom error"), undefined, undefined);
    expect(toast.error).not.toHaveBeenCalled();
  });
});

describe("setErrorOnError", () => {
  it("calls the setter with the error message", () => {
    const setter = vi.fn();
    setErrorOnError(setter)(new Error("fail"));
    expect(setter).toHaveBeenCalledWith("fail");
  });
});

describe("mutationSuccess", () => {
  it("calls toast.success with the message", async () => {
    const result = mutationSuccess("done");
    await result.onSuccess();
    expect(toast.success).toHaveBeenCalledWith("done");
  });

  it("calls all invalidation functions", async () => {
    const inv1 = vi.fn().mockResolvedValue(undefined);
    const inv2 = vi.fn().mockResolvedValue(undefined);
    const result = mutationSuccess("saved", inv1, inv2);
    await result.onSuccess();
    expect(inv1).toHaveBeenCalled();
    expect(inv2).toHaveBeenCalled();
  });

  it("runs invalidations concurrently", async () => {
    const order: string[] = [];
    const slow = vi.fn().mockImplementation(async () => { order.push("slow"); });
    const fast = vi.fn().mockResolvedValue(undefined);
    const result = mutationSuccess("ok", slow, fast);
    await result.onSuccess();
    expect(slow).toHaveBeenCalled();
    expect(fast).toHaveBeenCalled();
  });

  it("works with zero invalidations", async () => {
    const result = mutationSuccess("created");
    await result.onSuccess();
    expect(toast.success).toHaveBeenCalledWith("created");
  });
});
