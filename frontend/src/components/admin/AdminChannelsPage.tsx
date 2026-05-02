"use client";

import { useTranslations } from "next-intl";
import { Loader2Icon, TriangleAlertIcon } from "lucide-react";
import { useQuery } from "@tanstack/react-query";

import { request } from "@/lib/api";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface Channel {
  id: number;
  name: string;
  type: string;
  enabled: boolean;
  user_id: number;
  webhook_url: string;
  created_at: string;
}

interface ChannelsResponse {
  channels: Channel[];
}

export function AdminChannelsPage() {
  const t = useTranslations("admin");

  const { data, isLoading, error } = useQuery<ChannelsResponse>({
    queryKey: ["admin", "channels"],
    queryFn: () => request<ChannelsResponse>("/api/v1/admin/channels"),
  });

  const channels = data?.channels || [];

  return (
    <div>
      <h1 className="mb-6 text-2xl font-semibold">{t("channels.title")}</h1>

      {isLoading ? (
        <div className="flex flex-col items-center justify-center gap-2 py-6 text-center">
          <Loader2Icon className="size-5 animate-spin" />
          <p className="text-sm text-muted-foreground">{t("loading")}</p>
        </div>
      ) : error ? (
        <Alert variant="destructive">
          <TriangleAlertIcon />
          <AlertDescription>{t("error")}</AlertDescription>
        </Alert>
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("channels.id")}</TableHead>
                <TableHead>{t("channels.name")}</TableHead>
                <TableHead>{t("channels.kind")}</TableHead>
                <TableHead>{t("channels.enabled")}</TableHead>
                <TableHead>{t("createdAt")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {channels.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-muted-foreground">
                    {t("channels.empty")}
                  </TableCell>
                </TableRow>
              ) : (
                channels.map((channel) => (
                  <TableRow key={channel.id}>
                    <TableCell>{channel.id}</TableCell>
                    <TableCell>{channel.name}</TableCell>
                    <TableCell>{channel.type}</TableCell>
                    <TableCell>{channel.enabled ? t("channels.enabled") : "-"}</TableCell>
                    <TableCell>{new Date(channel.created_at).toLocaleString()}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}

export default AdminChannelsPage;
