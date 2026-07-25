"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { useQuery } from "@tanstack/react-query";

import { deliveryApi, deliveryKeys } from "@/lib/api";
import { DEFAULT_PAGE_SIZE } from "@/lib/constants";
import { PageHeading } from "@/components/ui/page-heading";
import { QueryState } from "@/components/ui/query-state";
import { PaginationControls } from "@/components/ui/pagination-controls";
import {
  Card,
  CardContent,
} from "@/components/ui/card";
import { DeliveryHistoryTable } from "./DeliveryHistoryTable";

export function DeliveryHistoryPage() {
  const t = useTranslations("delivery");
  const [page, setPage] = useState(1);
  const limit = DEFAULT_PAGE_SIZE;

  const { data, isLoading, error } = useQuery({
    queryKey: deliveryKeys.history(page, limit),
    queryFn: () => deliveryApi.listHistory(page, limit),
    refetchOnWindowFocus: false,
  });

  const items = data?.items ?? [];
  const pagination = data
    ? { page: data.page, limit: data.limit, total: data.total, total_pages: data.total_pages }
    : undefined;

  return (
    <div className="space-y-6">
      <PageHeading>{t("history.title")}</PageHeading>

      <Card>
        <CardContent>
          <QueryState
            isLoading={isLoading}
            error={error}
            loadingText={t("history.loading")}
            errorText={t("history.loadFailed")}
            loadingClassName="flex items-center justify-center gap-2 py-8"
          >
            <DeliveryHistoryTable items={items} />
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
    </div>
  );
}

export default DeliveryHistoryPage;
