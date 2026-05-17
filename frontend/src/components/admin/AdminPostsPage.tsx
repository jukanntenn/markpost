"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { SearchIcon } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { adminApi } from "@/lib/api";
import { formatToLocalTime } from "@/lib/utils";
import { buildPostUrl } from "@/utils/url";
import { Input } from "@/components/ui/input";
import { QueryState } from "@/components/ui/query-state";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, } from "@/components/ui/table";

export function AdminPostsPage() {
  const t = useTranslations("admin");
  const [search, setSearch] = useState("");

  const { data, isLoading, error } = useQuery({
    queryKey: ["admin", "posts", search],
    queryFn: () => adminApi.listPosts(search),
  });

  const posts = data?.posts || [];

  return (
    <div>
      <div className="mb-6 flex items-center justify-between md:mb-8 lg:mb-12">
        <h1 className="font-display text-[28px] font-bold tracking-tight">{t("posts.title")}</h1>
        <div className="relative w-64">
          <SearchIcon className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder={t("posts.searchPlaceholder")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
      </div>
      <QueryState isLoading={isLoading} error={error} loadingText={t("loading")} errorText={t("error")}>
        <div className="rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("posts.id")}</TableHead>
                <TableHead>{t("posts.titleCol")}</TableHead>
                <TableHead>{t("username")}</TableHead>
                <TableHead>{t("createdAt")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {posts.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={4} className="text-center text-muted-foreground">
                    {t("posts.empty")}
                  </TableCell>
                </TableRow>
              ) : (
                posts.map((post) => (
                  <TableRow key={post.id}>
                    <TableCell>{post.id.slice(0, 8)}...</TableCell>
                    <TableCell>
                      <a
                        href={buildPostUrl(post.qid)}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-primary underline-offset-4 hover:underline"
                      >
                        {post.title}
                      </a>
                    </TableCell>
                    <TableCell>{post.username}</TableCell>
                    <TableCell>{formatToLocalTime(post.created_at)}</TableCell>
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

export default AdminPostsPage;
