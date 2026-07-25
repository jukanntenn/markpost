import "@testing-library/jest-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { DeliveryChannelDialog } from "./DeliveryChannelDialog";
import { ThemeProvider } from "@/components/theme-provider";
import { renderWithProviders, mockMatchMedia } from "@/test/utils";
import { mockDeliveryChannels, resetDeliveryMocks } from "@/mocks/handlers";
import type { DeliveryChannel } from "@/types/delivery";

vi.mock("@/stores/toast", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  },
}));

const channel: DeliveryChannel = {
  id: 1,
  kind: "feishu",
  name: "Existing Channel",
  enabled: true,
  configuration: { webhook_url: "https://example.com/hook", card_link_url: "" },
  keywords: "alert",
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
};

beforeEach(() => {
  vi.clearAllMocks();
  resetDeliveryMocks();
  mockMatchMedia();
});

describe("DeliveryChannelDialog", () => {
  it("shows create title when editingChannel is null", () => {
    renderWithProviders(
      <DeliveryChannelDialog open={true} onOpenChange={vi.fn()} editingChannel={null} />,
      { wrapper: ThemeProvider },
    );

    expect(screen.getByRole("heading", { name: /add delivery channel/i })).toBeInTheDocument();
  });

  it("shows edit title and prefills fields when editingChannel is set", () => {
    renderWithProviders(
      <DeliveryChannelDialog open={true} onOpenChange={vi.fn()} editingChannel={channel} />,
      { wrapper: ThemeProvider },
    );

    expect(screen.getByRole("heading", { name: /edit delivery channel/i })).toBeInTheDocument();
    expect(screen.getByDisplayValue("Existing Channel")).toBeInTheDocument();
    expect(screen.getByDisplayValue("https://example.com/hook")).toBeInTheDocument();
    expect(screen.getByDisplayValue("alert")).toBeInTheDocument();
  });

  it("shows test button only when editing", () => {
    const { unmount } = renderWithProviders(
      <DeliveryChannelDialog open={true} onOpenChange={vi.fn()} editingChannel={null} />,
      { wrapper: ThemeProvider },
    );
    expect(screen.queryByRole("button", { name: /^test$/i })).not.toBeInTheDocument();

    unmount();

    renderWithProviders(
      <DeliveryChannelDialog open={true} onOpenChange={vi.fn()} editingChannel={channel} />,
      { wrapper: ThemeProvider },
    );
    expect(screen.getByRole("button", { name: /^test$/i })).toBeInTheDocument();
  });

  it("sends a test message and toasts success", async () => {
    const user = userEvent.setup();
    const { toast } = await import("@/stores/toast");

    renderWithProviders(
      <DeliveryChannelDialog open={true} onOpenChange={vi.fn()} editingChannel={channel} />,
      { wrapper: ThemeProvider },
    );

    await user.click(screen.getByRole("button", { name: /^test$/i }));

    await waitFor(() => {
      expect(toast.success).toHaveBeenCalledWith("Test message sent");
    });
  });

  it("reveals delete confirmation and deletes the channel", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    mockDeliveryChannels.push({ ...channel });

    renderWithProviders(
      <DeliveryChannelDialog open={true} onOpenChange={onOpenChange} editingChannel={channel} />,
      { wrapper: ThemeProvider },
    );

    await user.click(screen.getByRole("button", { name: /delete/i }));

    expect(screen.getByRole("button", { name: /confirm delete/i })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /confirm delete/i }));

    await waitFor(() => {
      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });
});
