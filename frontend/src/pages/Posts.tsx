import { useEffect, useState } from "react";
import { FileTextIcon, Loader2Icon, TriangleAlertIcon } from "lucide-react";
import { buildPostUrl } from "../utils/url";
import { usePosts } from "../hooks/swr/usePosts";
import { useSearchParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const formatToLocalTime = (utcString: string): string => {
  if (!utcString) return "";
  const date = new Date(utcString);
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  const seconds = String(date.getSeconds()).padStart(2, "0");
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
};

function Posts() {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const [page, setPage] = useState<number>(() => parseInt(searchParams.get("page") || "1", 10));
  const limit = 20;

  const { data, isLoading, error } = usePosts(page, limit, { refreshInterval: 3000 });

  const items = data?.posts || [];
  const totalPages = data?.pagination?.total_pages || 1;

  useEffect(() => {
    document.title = t("common.pageTitle.allPosts");
  }, [t]);

  const handlePageChange = (nextPage: number) => {
    if (nextPage < 1 || nextPage > totalPages) return;
    setPage(nextPage);
    setSearchParams({ page: String(nextPage) });
  };

  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between space-y-0">
        <div className="flex items-center gap-2">
          <FileTextIcon className="size-4" />
          <CardTitle className="text-base">{t("posts.title")}</CardTitle>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {isLoading ? (
          <div className="flex flex-col items-center justify-center gap-2 rounded-lg border bg-muted/40 p-6 text-center">
            <Loader2Icon className="size-5 animate-spin" />
            <p className="text-sm text-muted-foreground">{t("posts.loading")}</p>
          </div>
        ) : error ? (
          <Alert variant="destructive">
            <TriangleAlertIcon />
            <AlertDescription>{t("posts.error")}</AlertDescription>
          </Alert>
        ) : items.length === 0 ? (
          <div className="rounded-lg border bg-muted/40 p-6 text-center text-sm text-muted-foreground">
            {t("posts.empty")}
          </div>
        ) : (
          <>
            <div className="hidden md:block">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[60%]">{t("posts.table.title")}</TableHead>
                    <TableHead>{t("posts.table.createdAt")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((p) => (
                    <TableRow key={p.id}>
                      <TableCell className="max-w-0">
                        <a
                          href={buildPostUrl(p.qid)}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="block truncate font-medium underline-offset-4 hover:underline"
                        >
                          {p.title}
                        </a>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {formatToLocalTime(p.created_at)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            <div className="md:hidden">
              <ul className="-mx-2 divide-y">
                {items.map((p) => (
                  <li key={p.id} className="px-2 py-3">
                    <div className="flex flex-col gap-1">
                      <a
                        href={buildPostUrl(p.qid)}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-sm font-medium underline-offset-4 hover:underline"
                      >
                        {p.title}
                      </a>
                      <span className="text-xs text-muted-foreground">
                        {formatToLocalTime(p.created_at)}
                      </span>
                    </div>
                  </li>
                ))}
              </ul>
            </div>

            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => handlePageChange(page - 1)}
                >
                  {t("posts.pagination.prev")}
                </Button>
                <span className="text-xs text-muted-foreground">
                  {page} / {totalPages}
                </span>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={page >= totalPages}
                  onClick={() => handlePageChange(page + 1)}
                >
                  {t("posts.pagination.next")}
                </Button>
              </div>
              <span className="text-xs text-muted-foreground">
                {page} / {totalPages}
              </span>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

export default Posts;
