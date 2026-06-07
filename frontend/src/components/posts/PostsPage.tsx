"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { FileTextIcon } from "lucide-react";

import { usePosts } from "@/hooks/usePosts";
import { DEFAULT_PAGE_SIZE } from "@/lib/constants";
import { PostListItemRow } from "./PostListItemRow";
import { PostListEmptyState } from "./PostListEmptyState";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PaginationControls } from "@/components/ui/pagination-controls";
import { PageHeading } from "@/components/ui/page-heading";
import { QueryState } from "@/components/ui/query-state";

export function PostsPage() {
  const [page, setPage] = useState(1);
  const limit = DEFAULT_PAGE_SIZE;

  const t = useTranslations("posts");
  const { posts, pagination, isLoading, error } = usePosts(page, limit, { refetchInterval: page === 1 ? 3000 : undefined });

  return (
    <div>
      <PageHeading>{t("title")}</PageHeading>

      <Card>
        <CardHeader className="flex-row items-center gap-2">
          <FileTextIcon className="size-4" />
          <CardTitle className="text-base">{t("title")}</CardTitle>
        </CardHeader>
        <CardContent>
          <QueryState isLoading={isLoading} error={error} loadingText={t("loading")} errorText={t("error")}>
            {posts.length === 0 ? (
              <PostListEmptyState message={t("empty")} />
            ) : (
              <>
                <ul className="divide-y">
                  {posts.map((p) => (
                    <PostListItemRow key={p.id} post={p} showSeconds={false} />
                  ))}
                </ul>

                {pagination && (
                  <PaginationControls
                    page={page}
                    totalPages={pagination.total_pages}
                    onPageChange={setPage}
                    prevLabel={t("pagination.prev")}
                    nextLabel={t("pagination.next")}
                  />
                )}
              </>
            )}
          </QueryState>
        </CardContent>
      </Card>
    </div>
  );
}

export default PostsPage;
