"use client";

import { useTranslations } from "next-intl";
import { adminApi, adminKeys } from "@/lib/api";
import { useAdminTablePage } from "@/hooks/useAdminTablePage";
import { formatToLocalTime } from "@/utils/time";
import { TableHead, TableRow, TableCell } from "@/components/ui/table";
import { AdminTablePage } from "@/components/admin/AdminTablePage";

export function AdminChannelsPage() {
  const t = useTranslations("admin");

  const { items: channels, ...queryState } = useAdminTablePage({
    queryKey: adminKeys.channels.all(),
    queryFn: () => adminApi.listChannels(),
    itemKey: "channels",
    t,
  });

  return (
    <AdminTablePage
      title={t("channels.title")}
      {...queryState}
      emptyText={t("channels.empty")}
      headers={
        <>
          <TableHead>{t("channels.id")}</TableHead>
          <TableHead>{t("channels.name")}</TableHead>
          <TableHead>{t("channels.kind")}</TableHead>
          <TableHead>{t("channels.enabled")}</TableHead>
          <TableHead>{t("createdAt")}</TableHead>
        </>
      }
      colSpan={5}
      items={channels}
      renderRow={(channel) => (
        <TableRow key={channel.id}>
          <TableCell>{channel.id}</TableCell>
          <TableCell>{channel.name}</TableCell>
          <TableCell>{channel.kind}</TableCell>
          <TableCell>{channel.enabled ? t("channels.enabled") : "-"}</TableCell>
          <TableCell>{formatToLocalTime(channel.created_at)}</TableCell>
        </TableRow>
      )}
    />
  );
}

export default AdminChannelsPage;
