import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { server } from "../../mocks/server";
import { http, HttpResponse } from "msw";
import { setMockAuth, createWrapper } from "../../test/utils";
import { useDeliveryChannels } from "./useDeliveryChannels";
import { useCreateDeliveryChannel } from "./useCreateDeliveryChannel";
import type { CreateDeliveryChannelArgs } from "./useCreateDeliveryChannel";
import { useUpdateDeliveryChannel } from "./useUpdateDeliveryChannel";
import type { UpdateDeliveryChannelArgs } from "./useUpdateDeliveryChannel";
import { useDeleteDeliveryChannel } from "./useDeleteDeliveryChannel";

describe("delivery channel hooks", () => {
  beforeEach(() => {
    setMockAuth({
      access_token: "test_token",
      refresh_token: "test_refresh",
      user: { id: 1, username: "testuser" },
    });
  });

  it("fetches delivery channels successfully", async () => {
    server.use(
      http.get("/api/delivery/channels", () => {
        return HttpResponse.json({
          channels: [
            {
              id: 1,
              kind: "feishu",
              name: "team",
              enabled: true,
              webhook_url: "https://open.feishu.cn/open-apis/bot/v2/hook/abcdef",
              created_at: "2024-01-01T00:00:00Z",
              updated_at: "2024-01-01T00:00:00Z",
            },
          ],
        });
      })
    );

    const { result } = renderHook(() => useDeliveryChannels(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data?.channels?.length).toBe(1);
    });
  });

  it("creates a delivery channel", async () => {
    server.use(
      http.post("/api/delivery/channels", async ({ request }) => {
        const body = (await request.json()) as CreateDeliveryChannelArgs;
        expect(body.kind).toBe("feishu");
        expect(body.webhook_url).toContain("feishu");
        return HttpResponse.json({ ok: true });
      })
    );

    const { result } = renderHook(() => useCreateDeliveryChannel(), {
      wrapper: createWrapper(),
    });

    await result.current.trigger({
      kind: "feishu",
      webhook_url: "https://open.feishu.cn/open-apis/bot/v2/hook/abcdef",
    });
  });

  it("updates a delivery channel", async () => {
    server.use(
      http.put("/api/delivery/channels/:id", async ({ params, request }) => {
        expect(params.id).toBe("1");
        const body = (await request.json()) as Omit<UpdateDeliveryChannelArgs, "id">;
        expect(body.enabled).toBe(false);
        return HttpResponse.json({ ok: true });
      })
    );

    const { result } = renderHook(() => useUpdateDeliveryChannel(), {
      wrapper: createWrapper(),
    });

    await result.current.trigger({ id: 1, enabled: false });
  });

  it("deletes a delivery channel", async () => {
    server.use(
      http.delete("/api/delivery/channels/:id", ({ params }) => {
        expect(params.id).toBe("1");
        return HttpResponse.json({ ok: true });
      })
    );

    const { result } = renderHook(() => useDeleteDeliveryChannel(), {
      wrapper: createWrapper(),
    });

    await result.current.trigger({ id: 1 });
  });
});
