"use client";

import { useState, useCallback } from "react";
import { useTranslations } from "next-intl";
import { Loader2Icon, SearchIcon, TriangleAlertIcon } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { request } from "@/lib/api";
import { buildPostUrl } from "@/utils/url";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, } from "@/components/ui/table";

interface Post {
  id: string;
  qid: string;
  title: string;
  user_id: number;
  username: string;
  created_at: string;
}

interface PostsResponse {
  posts: Post[];
  total: number;
}

export function AdminPostsPage() {
  const t = useTranslations("admin");
  const [search, setSearch] = useState("");

  const { data, isLoading, error } = useQuery<PostsResponse>({
    queryKey: ["admin", "posts", search],
    queryFn: () => {
      const params = new URLSearchParams();
      if (search) params.set("search", search);
      const query = params.toString();
      return request<PostsResponse>(`/api/v1/admin/posts${query ? `?${query}` : ""}`);
    },
  });

  const posts = data?.posts || [];

  const handleSearch = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value);
  }, []);

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-semibold">{t("posts.title")}</h1>
        <div className="relative w-64">
          <SearchIcon className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder={t("posts.searchPlaceholder")}
            value={search}
            onChange={handleSearch}
            className="pl-9"
          />
        </div>
      </div>
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
      ) : (
        <div className="rounded-md border">
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
                    <TableCell>{new Date(post.created_at).toLocaleString()}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}

export default AdminPostsPage;
