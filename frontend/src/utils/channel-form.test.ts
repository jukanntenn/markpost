import { describe, it, expect } from "vitest";
import {
  channelToForm,
  formToCreatePayload,
  formToUpdatePayload,
  EMPTY_FORM,
  validateConfiguration,
} from "./channel-form";
import type { DeliveryChannel } from "@/types/delivery";

const channel: DeliveryChannel = {
  id: 1,
  kind: "feishu",
  name: "My Channel",
  enabled: true,
  configuration: {
    webhook_url: "https://example.com/hook",
    card_link_url: "",
  },
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
      configuration: {
        webhook_url: "https://example.com/hook",
        card_link_url: "",
      },
      keywords: "go,rust",
    });
  });
});

describe("formToCreatePayload", () => {
  it("maps FormState to CreateChannelPayload", () => {
    const form = {
      ...EMPTY_FORM,
      name: "Test",
      configuration: { webhook_url: "https://hook.test", card_link_url: "" },
    };
    const result = formToCreatePayload(form);
    expect(result).toEqual({
      kind: "feishu",
      name: "Test",
      configuration: { webhook_url: "https://hook.test", card_link_url: "" },
      keywords: "",
    });
  });
});

describe("formToUpdatePayload", () => {
  it("maps editingId and FormState to UpdateChannelMutationVars", () => {
    const form = {
      ...EMPTY_FORM,
      name: "Updated",
      configuration: { webhook_url: "https://new.test", card_link_url: "" },
      keywords: "ts",
    };
    const result = formToUpdatePayload(42, form);
    expect(result).toEqual({
      id: 42,
      data: {
        name: "Updated",
        configuration: { webhook_url: "https://new.test", card_link_url: "" },
        keywords: "ts",
      },
    });
  });

  it("excludes kind from the update data", () => {
    const result = formToUpdatePayload(1, { ...EMPTY_FORM, kind: "slack" });
    expect("kind" in result.data).toBe(false);
  });
});

describe("validateConfiguration", () => {
  it("validates a valid feishu configuration", () => {
    const result = validateConfiguration("feishu", {
      webhook_url: "https://example.com/hook",
      card_link_url: "",
    });
    expect(result.valid).toBe(true);
    expect(result.errors).toEqual({});
  });

  it("rejects empty webhook URL", () => {
    const result = validateConfiguration("feishu", {
      webhook_url: "",
      card_link_url: "",
    });
    expect(result.valid).toBe(false);
    expect(result.errors["webhook_url"]).toBeDefined();
  });

  it("rejects invalid URL", () => {
    const result = validateConfiguration("feishu", {
      webhook_url: "not-a-url",
      card_link_url: "",
    });
    expect(result.valid).toBe(false);
    expect(result.errors["webhook_url"]).toBeDefined();
  });

  it("rejects unsupported channel kind", () => {
    const result = validateConfiguration("slack", {
      webhook_url: "https://example.com",
      card_link_url: "",
    });
    expect(result.valid).toBe(false);
    expect(result.errors["kind"]).toBeDefined();
  });

  it("accepts card_link_url with template variables", () => {
    const result = validateConfiguration("feishu", {
      webhook_url: "https://example.com/hook",
      card_link_url: "https://custom.com/{{.QID}}",
    });
    expect(result.valid).toBe(true);
  });
});
