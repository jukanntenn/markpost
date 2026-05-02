"use client";

import { useState } from "react";
import Link from "next/link";
import { useTranslations } from "next-intl";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  FileTextIcon,
  Loader2Icon,
  TriangleAlertIcon,
} from "lucide-react";

import { buildPostUrl } from "@/utils/url";
import { usePosts } from "@/hooks/usePosts";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const formatToLocalTime = (utcString: string): string => {
  if (!utcString) return "";
  const date = new Date(utcString);
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  return `${year}-${month}-${day} ${hours}:${minutes}`;
};

export function PostsPage() {
  const [page, setPage] = useState(1);
  const limit = 10;

  const t = useTranslations("posts");
  const { data, isLoading, error } = usePosts(page, limit);

  const posts = data?.posts || [];
  const pagination = data?.pagination;

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-semibold">{t("title")}</h1>
      </div>

      <Card>
        <CardHeader className="flex-row items-center gap-2">
          <FileTextIcon className="size-4" />
          <CardTitle className="text-base">{t("title")}</CardTitle>
        </CardHeader>
        <CardContent>
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
          ) : posts.length === 0 ? (
            <p className="py-6 text-center text-sm text-muted-foreground">
              {t("empty")}
            </p>
          ) : (
            <>
              <ul className="divide-y">
                {posts.map((p) => (
                  <li key={p.id} className="py-3">
                    <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
                      <a
                        href={buildPostUrl(p.qid)}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="truncate text-sm font-medium underline-offset-4 hover:underline"
                      >
                        {p.title}
                      </a>
                      <span className="shrink-0 text-xs text-muted-foreground">
                        {formatToLocalTime(p.created_at)}
                      </span>
                    </div>
                  </li>
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
        </CardContent>
      </Card>
    </div>
  );
}

export default PostsPage;
