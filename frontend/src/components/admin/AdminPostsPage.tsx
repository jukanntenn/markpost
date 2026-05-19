"use client";

import { useTranslations } from "next-intl";
import { adminApi, adminKeys } from "@/lib/api";
import { useAdminSearchTablePage } from "@/hooks/useAdminTablePage";
import { formatToLocalTime } from "@/utils/time";
import { buildPostUrl } from "@/utils/url";
import { SearchInput } from "@/components/ui/search-input";
import { TableHead, TableRow, TableCell } from "@/components/ui/table";
import { AdminTablePage } from "@/components/admin/AdminTablePage";

export function AdminPostsPage() {
  const t = useTranslations("admin");
  const { items: posts, search, setSearch, ...queryState } = useAdminSearchTablePage({
    queryKeyBuilder: adminKeys.posts.list,
    queryFn: adminApi.listPosts,
    itemKey: "posts",
    t,
  });

  return (
    <AdminTablePage
      title={t("posts.title")}
      toolbar={
        <SearchInput
          placeholder={t("posts.searchPlaceholder")}
          value={search}
          onChange={setSearch}
        />
      }
      {...queryState}
      emptyText={t("posts.empty")}
      headers={
        <>
          <TableHead>{t("posts.id")}</TableHead>
          <TableHead>{t("posts.titleCol")}</TableHead>
          <TableHead>{t("username")}</TableHead>
          <TableHead>{t("createdAt")}</TableHead>
        </>
      }
      colSpan={4}
      items={posts}
      renderRow={(post) => (
        <TableRow key={post.qid}>
          <TableCell>{post.qid.slice(0, 8)}...</TableCell>
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
      )}
    />
  );
}

export default AdminPostsPage;
