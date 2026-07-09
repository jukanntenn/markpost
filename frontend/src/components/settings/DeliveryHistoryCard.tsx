"use client";

import { useState } from "react";
import Link from "next/link";
import { useTranslations } from "next-intl";

import { useQuery } from "@tanstack/react-query";

import { deliveryApi, deliveryKeys } from "@/lib/api";
import { DEFAULT_PAGE_SIZE } from "@/lib/constants";
import { truncate, cn } from "@/lib/utils";
import { buildPostUrl } from "@/utils/url";
import { formatToLocalTime } from "@/utils/time";
import { PaginationControls } from "@/components/ui/pagination-controls";
import { QueryState } from "@/components/ui/query-state";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { DeliveryStatus } from "@/types/delivery";

const statusVariant: Record<DeliveryStatus, "secondary" | "destructive" | "outline"> = {
  delivered: "secondary",
  failed: "destructive",
  expired: "outline",
};

export function DeliveryHistoryCard() {
  const t = useTranslations("settings");
  const [page, setPage] = useState(1);
  const limit = DEFAULT_PAGE_SIZE;

  const { data, isLoading, error } = useQuery({
    queryKey: deliveryKeys.history(page, limit),
    queryFn: () => deliveryApi.listHistory(page, limit),
    refetchOnWindowFocus: false,
  });

  const history = data?.history ?? [];
  const pagination = data?.pagination;

  return (
    <Card data-testid="delivery-history-card">
      <CardHeader>
        <CardTitle>{t("deliveryHistory")}</CardTitle>
        <CardDescription>{t("deliveryHistoryList")}</CardDescription>
      </CardHeader>
      <CardContent>
        <QueryState
          isLoading={isLoading}
          error={error}
          loadingText={t("deliveryHistoryLoading")}
          errorText={t("deliveryHistoryLoadFailed")}
        >
          {history.length === 0 ? (
            <p className="py-4 text-center text-sm text-muted-foreground">
              {t("deliveryHistoryEmpty")}
            </p>
          ) : (
            <>
              <ul className="divide-y">
                {history.map((item) => {
                  const showError = (item.status === "failed" || item.status === "expired") && item.last_error;
                  return (
                    <li key={item.id} className="flex flex-col gap-1 py-3">
                      <div className="flex items-center justify-between gap-3">
                        <div className="min-w-0 flex-1">
                          {item.post_qid ? (
                            <Link
                              href={buildPostUrl(item.post_qid)}
                              className="truncate text-sm font-medium hover:underline"
                            >
                              {item.post_title ?? item.post_qid}
                            </Link>
                          ) : (
                            <span className="text-sm text-muted-foreground italic">
                              {t("deliveryHistoryPostDeleted")}
                            </span>
                          )}
                          <p className="truncate text-xs text-muted-foreground">
                            {item.channel_name ?? t("deliveryHistoryChannelDeleted")}
                          </p>
                        </div>
                        <Badge variant={statusVariant[item.status]}>
                          {t(`deliveryHistoryStatus_${item.status}`)}
                        </Badge>
                      </div>
                      <div className="flex items-center justify-between gap-3">
                        <span className="text-xs text-muted-foreground">
                          {formatToLocalTime(item.created_at, { includeSeconds: false })}
                        </span>
                        {showError && (
                          <span
                            className={cn("max-w-[60%] truncate text-xs text-destructive")}
                            title={item.last_error}
                          >
                            {truncate(item.last_error, 60)}
                          </span>
                        )}
                      </div>
                    </li>
                  );
                })}
              </ul>

              {pagination && (
                <PaginationControls
                  page={page}
                  totalPages={pagination.total_pages}
                  onPageChange={setPage}
                  prevLabel={t("deliveryHistoryPrev")}
                  nextLabel={t("deliveryHistoryNext")}
                />
              )}
            </>
          )}
        </QueryState>
      </CardContent>
    </Card>
  );
}

export default DeliveryHistoryCard;
