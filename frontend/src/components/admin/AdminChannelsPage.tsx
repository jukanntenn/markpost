"use client";

import { useTranslations } from "next-intl";
import { useQuery } from "@tanstack/react-query";

import { adminApi } from "@/lib/api";
import { formatToLocalTime } from "@/lib/utils";
import { QueryState } from "@/components/ui/query-state";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export function AdminChannelsPage() {
  const t = useTranslations("admin");

  const { data, isLoading, error } = useQuery({
    queryKey: ["admin", "channels"],
    queryFn: () => adminApi.listChannels(),
  });

  const channels = data?.channels || [];

  return (
    <div>
      <h1 className="mb-6 font-display text-[28px] font-bold tracking-tight md:mb-8 lg:mb-12">{t("channels.title")}</h1>

      <QueryState isLoading={isLoading} error={error} loadingText={t("loading")} errorText={t("error")}>
        <div className="rounded-lg border">
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
                    <TableCell>{channel.kind}</TableCell>
                    <TableCell>{channel.enabled ? t("channels.enabled") : "-"}</TableCell>
                    <TableCell>{formatToLocalTime(channel.created_at)}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </QueryState>
    </div>
  );
}

export default AdminChannelsPage;
