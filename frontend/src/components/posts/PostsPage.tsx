"use client";

import { useState } from "react";
import Link from "next/link";
import { useTranslations } from "next-intl";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  FileTextIcon,
} from "lucide-react";

import { usePosts } from "@/hooks/usePosts";
import { PostListItemRow } from "./PostListItemRow";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { QueryState } from "@/components/ui/query-state";

export function PostsPage() {
  const [page, setPage] = useState(1);
  const limit = 10;

  const t = useTranslations("posts");
  const { data, isLoading, error } = usePosts(page, limit);

  const posts = data?.posts || [];
  const pagination = data?.pagination;

  return (
    <div>
      <div className="mb-6 flex items-center justify-between md:mb-8 lg:mb-12">
        <h1 className="font-display text-[28px] font-bold tracking-tight">{t("title")}</h1>
      </div>

      <Card>
        <CardHeader className="flex-row items-center gap-2">
          <FileTextIcon className="size-4" />
          <CardTitle className="text-base">{t("title")}</CardTitle>
        </CardHeader>
        <CardContent>
          <QueryState isLoading={isLoading} error={error} loadingText={t("loading")} errorText={t("error")}>
            {posts.length === 0 ? (
              <p className="py-6 text-center text-sm text-muted-foreground">
                {t("empty")}
              </p>
            ) : (
              <>
                <ul className="divide-y">
                  {posts.map((p) => (
                    <PostListItemRow key={p.id} post={p} showSeconds={false} />
                  ))}
                </ul>

                {pagination && pagination.total_pages > 1 && (
                  <div className="mt-4 flex items-center justify-between border-t pt-4">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setPage((p) => Math.max(1, p - 1))}
                      disabled={page === 1}
                    >
                      <ChevronLeftIcon className="mr-1 size-4" />
                      {t("pagination.prev")}
                    </Button>
                    <span className="text-sm text-muted-foreground">
                      {page} / {pagination.total_pages}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() =>
                        setPage((p) => Math.min(pagination.total_pages, p + 1))
                      }
                      disabled={page === pagination.total_pages}
                    >
                      {t("pagination.next")}
                      <ChevronRightIcon className="ml-1 size-4" />
                    </Button>
                  </div>
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
