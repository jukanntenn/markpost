"use client";

import { useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { useQuery } from "@tanstack/react-query";
import { ChevronLeftIcon } from "lucide-react";

import { deliveryApi, deliveryKeys } from "@/lib/api";
import { DEFAULT_PAGE_SIZE } from "@/lib/constants";
import { truncate } from "@/lib/utils";
import { formatToLocalTime } from "@/utils/time";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import { QueryState } from "@/components/ui/query-state";
import { PaginationControls } from "@/components/ui/pagination-controls";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { DeliveryChannel } from "@/types/delivery";
import { DeliveryChannelDialog } from "./DeliveryChannelDialog";
import { DeliveryHistoryTable } from "./DeliveryHistoryTable";

export function DeliveryChannelDetailPage() {
  const searchParams = useSearchParams();
  const t = useTranslations("delivery");
  const [page, setPage] = useState(1);
  const limit = DEFAULT_PAGE_SIZE;

  const channelID = Number(searchParams.get("id"));
  const channelsQuery = useQuery({
    queryKey: deliveryKeys.channels(),
    queryFn: deliveryApi.list,
  });

  const channel: DeliveryChannel | undefined = channelsQuery.data?.items.find(
    (c) => c.id === channelID,
  );

  const historyQuery = useQuery({
    queryKey: deliveryKeys.history(page, limit, channelID),
    queryFn: () => deliveryApi.listHistory(page, limit, channelID),
    enabled: !!channel,
    refetchOnWindowFocus: false,
  });

  const [dialogOpen, setDialogOpen] = useState(false);

  const history = historyQuery.data?.items ?? [];
  const pagination = historyQuery.data
    ? {
        page: historyQuery.data.page,
        limit: historyQuery.data.limit,
        total: historyQuery.data.total,
        total_pages: historyQuery.data.total_pages,
      }
    : undefined;

  if (channelsQuery.isLoading) {
    return (
      <div className="flex items-center justify-center gap-2 py-16">
        <Spinner className="size-5" />
        <span className="text-sm text-muted-foreground">{t("channelDetail.loading")}</span>
      </div>
    );
  }

  if (!channel) {
    return (
      <div className="space-y-6">
        <BackLink />
        <p className="py-16 text-center text-sm text-muted-foreground">
          {t("channelDetail.notFound")}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <BackLink />

      <div className="flex items-start justify-between gap-4">
        <div className="space-y-1">
          <h1 className="font-display text-2xl font-bold tracking-tight">
            {channel.name || t("channels.unnamed")}
          </h1>
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <span>{channel.kind}</span>
            <span>·</span>
            <Badge variant={channel.enabled ? "secondary" : "outline"}>
              {channel.enabled ? t("channelDetail.enabled") : t("channelDetail.disabled")}
            </Badge>
          </div>
        </div>
        <Button variant="outline" onClick={() => setDialogOpen(true)}>
          {t("channelDetail.edit")}
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("channelDetail.configTitle")}</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <ConfigField label={t("channelDetail.webhookURL")}>
              <code className="break-all text-sm">{truncate(channel.configuration?.webhook_url ?? "", 60)}</code>
            </ConfigField>
            <ConfigField label={t("channelDetail.cardLinkURL")}>
              <code className="break-all text-sm">
                {channel.configuration?.card_link_url || "—"}
              </code>
            </ConfigField>
            <ConfigField label={t("channelDetail.keywords")}>
              <span className="text-sm">{channel.keywords || "—"}</span>
            </ConfigField>
            <ConfigField label={t("channelDetail.createdAt")}>
              <span className="text-sm">{formatToLocalTime(channel.created_at)}</span>
            </ConfigField>
          </dl>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("channelDetail.historyTitle")}</CardTitle>
        </CardHeader>
        <CardContent>
          <QueryState
            isLoading={historyQuery.isLoading}
            error={historyQuery.error}
            loadingText={t("history.loading")}
            errorText={t("history.loadFailed")}
            loadingClassName="flex items-center justify-center gap-2 py-8"
          >
            <DeliveryHistoryTable items={history} />
            {pagination && (
              <PaginationControls
                page={page}
                totalPages={pagination.total_pages}
                onPageChange={setPage}
                prevLabel={t("history.prev")}
                nextLabel={t("history.next")}
              />
            )}
          </QueryState>
        </CardContent>
      </Card>

      {dialogOpen && (
        <DeliveryChannelDialog
          open={dialogOpen}
          onOpenChange={setDialogOpen}
          editingChannel={channel}
        />
      )}
    </div>
  );
}

function BackLink() {
  const t = useTranslations("delivery");
  return (
    <Link
      href="/delivery/channels"
      className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
    >
      <ChevronLeftIcon className="size-4" />
      {t("channelDetail.back")}
    </Link>
  );
}

function ConfigField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1">
      <dt className="text-xs font-medium text-muted-foreground">{label}</dt>
      <dd>{children}</dd>
    </div>
  );
}

export default DeliveryChannelDetailPage;
