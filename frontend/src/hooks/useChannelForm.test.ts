import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { DeliveryChannel } from "@/types/delivery";
import type { CreateChannelPayload } from "@/types/delivery";
import type { UpdateChannelMutationVars } from "@/utils/channel-form";
import { EMPTY_FORM } from "@/utils/channel-form";
import { useChannelForm } from "./useChannelForm";

function createMockMutation<T>() {
  return {
    mutate: vi.fn(),
    isPending: false,
  } as unknown as {
    mutate: ReturnType<typeof vi.fn>;
    isPending: boolean;
  };
}

const mockChannel: DeliveryChannel = {
  id: 5,
  kind: "feishu",
  name: "My Channel",
  enabled: true,
  configuration: {
    webhook_url: "https://example.com",
    card_link_url: "",
  },
  keywords: "test",
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
};

describe("useChannelForm", () => {
  it("returns correct initial state", () => {
    const createMutation = createMockMutation<CreateChannelPayload>();
    const updateMutation = createMockMutation<UpdateChannelMutationVars>();
    const { result } = renderHook(() =>
      useChannelForm({ createMutation, updateMutation }),
    );

    expect(result.current.showForm).toBe(false);
    expect(result.current.editingId).toBe(null);
    expect(result.current.form).toEqual(EMPTY_FORM);
    expect(result.current.isSubmitting).toBe(false);
  });

  it("openNewForm shows form with empty state", () => {
    const createMutation = createMockMutation<CreateChannelPayload>();
    const updateMutation = createMockMutation<UpdateChannelMutationVars>();
    const { result } = renderHook(() =>
      useChannelForm({ createMutation, updateMutation }),
    );

    act(() => {
      result.current.openNewForm();
    });

    expect(result.current.showForm).toBe(true);
    expect(result.current.editingId).toBe(null);
    expect(result.current.form).toEqual(EMPTY_FORM);
  });

  it("startEdit populates form with channel data", () => {
    const createMutation = createMockMutation<CreateChannelPayload>();
    const updateMutation = createMockMutation<UpdateChannelMutationVars>();
    const { result } = renderHook(() =>
      useChannelForm({ createMutation, updateMutation }),
    );

    act(() => {
      result.current.startEdit(mockChannel);
    });

    expect(result.current.showForm).toBe(true);
    expect(result.current.editingId).toBe(5);
    expect(result.current.form).toEqual({
      kind: "feishu",
      name: "My Channel",
      configuration: {
        webhook_url: "https://example.com",
        card_link_url: "",
      },
      keywords: "test",
    });
  });

  it("resetForm hides form and clears state", () => {
    const createMutation = createMockMutation<CreateChannelPayload>();
    const updateMutation = createMockMutation<UpdateChannelMutationVars>();
    const { result } = renderHook(() =>
      useChannelForm({ createMutation, updateMutation }),
    );

    act(() => {
      result.current.openNewForm();
    });
    expect(result.current.showForm).toBe(true);

    act(() => {
      result.current.resetForm();
    });

    expect(result.current.showForm).toBe(false);
    expect(result.current.editingId).toBe(null);
    expect(result.current.form).toEqual(EMPTY_FORM);
  });

  it("handleSubmit dispatches create when not editing", () => {
    const createMutation = createMockMutation<CreateChannelPayload>();
    const updateMutation = createMockMutation<UpdateChannelMutationVars>();
    const { result } = renderHook(() =>
      useChannelForm({ createMutation, updateMutation }),
    );

    act(() => {
      result.current.openNewForm();
    });

    act(() => {
      result.current.handleSubmit({
        preventDefault: vi.fn(),
      } as unknown as React.FormEvent);
    });

    expect(createMutation.mutate).toHaveBeenCalledWith({
      kind: EMPTY_FORM.kind,
      name: EMPTY_FORM.name,
      configuration: EMPTY_FORM.configuration,
      keywords: EMPTY_FORM.keywords,
    });
    expect(updateMutation.mutate).not.toHaveBeenCalled();
  });

  it("handleSubmit dispatches update when editing", () => {
    const createMutation = createMockMutation<CreateChannelPayload>();
    const updateMutation = createMockMutation<UpdateChannelMutationVars>();
    const { result } = renderHook(() =>
      useChannelForm({ createMutation, updateMutation }),
    );

    act(() => {
      result.current.startEdit(mockChannel);
    });

    act(() => {
      result.current.handleSubmit({
        preventDefault: vi.fn(),
      } as unknown as React.FormEvent);
    });

    expect(updateMutation.mutate).toHaveBeenCalledWith({
      id: 5,
      data: {
        name: "My Channel",
        configuration: {
          webhook_url: "https://example.com",
          card_link_url: "",
        },
        keywords: "test",
      },
    });
    expect(createMutation.mutate).not.toHaveBeenCalled();
  });

  it("isSubmitting is true when createMutation is pending", () => {
    const createMutation = createMockMutation<CreateChannelPayload>();
    const updateMutation = createMockMutation<UpdateChannelMutationVars>();
    createMutation.isPending = true;

    const { result } = renderHook(() =>
      useChannelForm({ createMutation, updateMutation }),
    );

    expect(result.current.isSubmitting).toBe(true);
  });

  it("isSubmitting is true when updateMutation is pending", () => {
    const createMutation = createMockMutation<CreateChannelPayload>();
    const updateMutation = createMockMutation<UpdateChannelMutationVars>();
    updateMutation.isPending = true;

    const { result } = renderHook(() =>
      useChannelForm({ createMutation, updateMutation }),
    );

    expect(result.current.isSubmitting).toBe(true);
  });

  it("isSubmitting is false when neither mutation is pending", () => {
    const createMutation = createMockMutation<CreateChannelPayload>();
    const updateMutation = createMockMutation<UpdateChannelMutationVars>();

    const { result } = renderHook(() =>
      useChannelForm({ createMutation, updateMutation }),
    );

    expect(result.current.isSubmitting).toBe(false);
  });
});
