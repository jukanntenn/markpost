"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Loader2Icon, PlusIcon, Trash2Icon, TriangleAlertIcon, PencilIcon } from "lucide-react";

import { request } from "@/lib/api";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";

interface Channel {
  id: number;
  kind: string;
  name: string;
  enabled: boolean;
  webhook_url: string;
  keywords: string;
  created_at: string;
  updated_at: string;
}

interface ChannelsResponse {
  channels: Channel[];
}

async function fetchChannels(): Promise<Channel[]> {
  const data = await request<ChannelsResponse>("/api/v1/delivery/channels");
  return data.channels;
}

async function createChannel(data: {
  kind: string;
  name: string;
  webhook_url: string;
  keywords?: string;
}): Promise<void> {
  await request("/api/v1/delivery/channels", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

async function updateChannel(
  id: number,
  data: {
    name?: string;
    webhook_url?: string;
    keywords?: string;
    enabled?: boolean;
  }
): Promise<void> {
  await request(`/api/v1/delivery/channels/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

async function deleteChannel(id: number): Promise<void> {
  await request(`/api/v1/delivery/channels/${id}`, {
    method: "DELETE",
  });
}

export function DeliveryChannelsCard() {
  const queryClient = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null);

  // Form state
  const [formName, setFormName] = useState("");
  const [formWebhookUrl, setFormWebhookUrl] = useState("");
  const [formKeywords, setFormKeywords] = useState("");

  const {
    data: channels,
    isLoading,
    error,
  } = useQuery<Channel[]>({
    queryKey: ["delivery", "channels"],
    queryFn: fetchChannels,
  });

  const createMutation = useMutation({
    mutationFn: createChannel,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["delivery", "channels"] });
      resetForm();
      toast.success("Channel created successfully");
    },
    onError: (err: Error) => {
      toast.error(err.message || "Failed to create channel");
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof updateChannel>[1] }) =>
      updateChannel(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["delivery", "channels"] });
      resetForm();
      toast.success("Channel updated successfully");
    },
    onError: (err: Error) => {
      toast.error(err.message || "Failed to update channel");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteChannel,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["delivery", "channels"] });
      setDeleteConfirmId(null);
      toast.success("Channel deleted successfully");
    },
    onError: (err: Error) => {
      toast.error(err.message || "Failed to delete channel");
    },
  });

  function resetForm() {
    setShowForm(false);
    setEditingId(null);
    setFormName("");
    setFormWebhookUrl("");
    setFormKeywords("");
  }

  function startEdit(channel: Channel) {
    setEditingId(channel.id);
    setFormName(channel.name);
    setFormWebhookUrl(channel.webhook_url);
    setFormKeywords(channel.keywords);
    setShowForm(true);
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    if (editingId) {
      updateMutation.mutate({
        id: editingId,
        data: {
          name: formName,
          webhook_url: formWebhookUrl,
          keywords: formKeywords,
        },
      });
    } else {
      createMutation.mutate({
        kind: "feishu",
        name: formName,
        webhook_url: formWebhookUrl,
        keywords: formKeywords,
      });
    }
  }

  const isSubmitting = createMutation.isPending || updateMutation.isPending;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>Delivery Channels</CardTitle>
            <CardDescription>
              Manage your notification delivery channels
            </CardDescription>
          </div>
          {!showForm && (
            <Button
              size="sm"
              onClick={() => {
                resetForm();
                setShowForm(true);
              }}
            >
              <PlusIcon className="mr-1 size-4" />
              Add Channel
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {isLoading ? (
          <div className="flex items-center justify-center gap-2 py-4">
            <Loader2Icon className="size-4 animate-spin" />
            <span className="text-sm text-muted-foreground">Loading channels...</span>
          </div>
        ) : error ? (
          <Alert variant="destructive">
            <TriangleAlertIcon />
            <AlertDescription>Error loading channels</AlertDescription>
          </Alert>
        ) : (
          <>
            {channels && channels.length === 0 && !showForm && (
              <p className="py-4 text-center text-sm text-muted-foreground">
                No delivery channels configured. Add one to get started.
              </p>
            )}

            {channels && channels.length > 0 && (
              <div className="space-y-3">
                {channels.map((channel) => (
                  <div
                    key={channel.id}
                    className="flex items-center justify-between rounded-lg border p-3"
                  >
                    <div className="flex items-center gap-3">
                      <Switch
                        size="sm"
                        checked={channel.enabled}
                        onCheckedChange={(checked) =>
                          updateMutation.mutate({
                            id: channel.id,
                            data: { enabled: checked },
                          })
                        }
                      />
                      <div>
                        <p className="text-sm font-medium">{channel.name}</p>
                        <p className="text-xs text-muted-foreground">
                          {channel.kind} · {channel.webhook_url.slice(0, 40)}...
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => startEdit(channel)}
                      >
                        <PencilIcon className="size-4" />
                      </Button>
                      {deleteConfirmId === channel.id ? (
                        <div className="flex items-center gap-1">
                          <Button
                            variant="destructive"
                            size="sm"
                            onClick={() => deleteMutation.mutate(channel.id)}
                            disabled={deleteMutation.isPending}
                          >
                            {deleteMutation.isPending ? (
                              <Loader2Icon className="size-4 animate-spin" />
                            ) : (
                              "Confirm"
                            )}
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setDeleteConfirmId(null)}
                          >
                            Cancel
                          </Button>
                        </div>
                      ) : (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setDeleteConfirmId(channel.id)}
                        >
                          <Trash2Icon className="size-4" />
                        </Button>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}

            {showForm && (
              <>
                <Separator />
                <form onSubmit={handleSubmit} className="space-y-3">
                  <div className="space-y-2">
                    <Label htmlFor="channel-name">Channel Name</Label>
                    <Input
                      id="channel-name"
                      value={formName}
                      onChange={(e) => setFormName(e.target.value)}
                      placeholder="My Feishu Channel"
                      required
                      disabled={isSubmitting}
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="channel-webhook">Feishu Webhook URL</Label>
                    <Input
                      id="channel-webhook"
                      value={formWebhookUrl}
                      onChange={(e) => setFormWebhookUrl(e.target.value)}
                      placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/..."
                      required
                      disabled={isSubmitting}
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="channel-keywords">Keywords (optional)</Label>
                    <Input
                      id="channel-keywords"
                      value={formKeywords}
                      onChange={(e) => setFormKeywords(e.target.value)}
                      placeholder="Comma-separated keywords"
                      disabled={isSubmitting}
                    />
                  </div>

                  <div className="flex gap-2">
                    <Button type="submit" disabled={isSubmitting}>
                      {isSubmitting ? (
                        <span className="inline-flex items-center gap-2">
                          <Loader2Icon className="size-4 animate-spin" />
                          Saving...
                        </span>
                      ) : editingId ? (
                        "Update Channel"
                      ) : (
                        "Create Channel"
                      )}
                    </Button>
                    <Button
                      type="button"
                      variant="outline"
                      onClick={resetForm}
                      disabled={isSubmitting}
                    >
                      Cancel
                    </Button>
                  </div>
                </form>
              </>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}

export default DeliveryChannelsCard;
