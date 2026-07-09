"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";

import { adminApi, adminKeys } from "@/lib/api";
import { useAdminTablePage } from "@/hooks/useAdminTablePage";
import { formatToLocalTime } from "@/utils/time";
import { truncate, cn } from "@/lib/utils";
import { buildPostUrl } from "@/utils/url";
import { TableHead, TableRow, TableCell } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { AdminTablePage } from "@/components/admin/AdminTablePage";
import type { DeliveryStatus } from "@/types/delivery";

const statusVariant: Record<DeliveryStatus, "secondary" | "destructive" | "outline"> = {
  delivered: "secondary",
  failed: "destructive",
  expired: "outline",
};

export function AdminDeliveryHistoryPage() {
  const t = useTranslations("admin");

  const { items: history, ...queryState } = useAdminTablePage({
    queryKey: adminKeys.history.all(),
    queryFn: () => adminApi.listDeliveryHistory(),
    itemKey: "history",
    t,
  });

  return (
    <AdminTablePage
      title={t("history.title")}
      {...queryState}
      emptyText={t("history.empty")}
      headers={
        <>
          <TableHead>{t("history.createdAt")}</TableHead>
          <TableHead>{t("history.post")}</TableHead>
          <TableHead>{t("history.channel")}</TableHead>
          <TableHead>{t("history.user")}</TableHead>
          <TableHead>{t("history.status")}</TableHead>
          <TableHead>{t("history.error")}</TableHead>
        </>
      }
      colSpan={6}
      items={history}
      renderRow={(item) => {
        const showError = (item.status === "failed" || item.status === "expired") && item.last_error;
        return (
          <TableRow key={item.id}>
            <TableCell className="whitespace-nowrap">
              {formatToLocalTime(item.created_at, { includeSeconds: false })}
            </TableCell>
            <TableCell>
              {item.post_qid ? (
                <Link href={buildPostUrl(item.post_qid)} className="truncate font-medium hover:underline">
                  {item.post_title ?? item.post_qid}
                </Link>
              ) : (
                <span className="text-muted-foreground italic">{t("history.postDeleted")}</span>
              )}
            </TableCell>
            <TableCell>
              {item.channel_name ?? (
                <span className="text-muted-foreground italic">{t("history.channelDeleted")}</span>
              )}
            </TableCell>
            <TableCell>
              {item.username ?? (
                <span className="text-muted-foreground italic">{t("history.userDeleted")}</span>
              )}
            </TableCell>
            <TableCell>
              <Badge variant={statusVariant[item.status]}>
                {t(`history.status_${item.status}`)}
              </Badge>
            </TableCell>
            <TableCell>
              {showError ? (
                <span className={cn("block max-w-[200px] truncate text-destructive")} title={item.last_error}>
                  {truncate(item.last_error, 50)}
                </span>
              ) : (
                <span className="text-muted-foreground">—</span>
              )}
            </TableCell>
          </TableRow>
        );
      }}
    />
  );
}

export default AdminDeliveryHistoryPage;
