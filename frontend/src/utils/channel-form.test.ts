import { describe, it, expect } from "vitest";
import { channelToForm, formToCreatePayload, formToUpdatePayload, EMPTY_FORM } from "./channel-form";
import type { DeliveryChannel } from "@/types/delivery";

const channel: DeliveryChannel = {
  id: 1,
  kind: "feishu",
  name: "My Channel",
  enabled: true,
  webhook_url: "https://example.com/hook",
  keywords: "go,rust",
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

describe("channelToForm", () => {
  it("maps DeliveryChannel fields to FormState", () => {
    const result = channelToForm(channel);
    expect(result).toEqual({
      kind: "feishu",
      name: "My Channel",
      webhookUrl: "https://example.com/hook",
      keywords: "go,rust",
    });
  });
});

describe("formToCreatePayload", () => {
  it("maps FormState to CreateChannelPayload", () => {
    const form = { ...EMPTY_FORM, name: "Test", webhookUrl: "https://hook.test" };
    const result = formToCreatePayload(form);
    expect(result).toEqual({
      kind: "feishu",
      name: "Test",
      webhook_url: "https://hook.test",
      keywords: "",
    });
  });
});

describe("formToUpdatePayload", () => {
  it("maps editingId and FormState to UpdateChannelMutationVars", () => {
    const form = { ...EMPTY_FORM, name: "Updated", webhookUrl: "https://new.test", keywords: "ts" };
    const result = formToUpdatePayload(42, form);
    expect(result).toEqual({
      id: 42,
      data: {
        name: "Updated",
        webhook_url: "https://new.test",
        keywords: "ts",
      },
    });
  });

  it("excludes kind from the update data", () => {
    const result = formToUpdatePayload(1, { ...EMPTY_FORM, kind: "slack" });
    expect("kind" in result.data).toBe(false);
  });
});
