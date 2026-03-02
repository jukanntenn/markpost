import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Loader2Icon, PencilIcon, Trash2Icon } from "lucide-react";
import { toast } from "sonner";

import { auth, getErrorMessage } from "../utils/api";
import { useAdminPosts, type AdminPost } from "../hooks/swr/useAdminPosts";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import ConfirmDialog from "@/components/admin/ConfirmDialog";

export default function AdminPosts() {
  const { t } = useTranslation();

  const [page, setPage] = useState(1);
  const limit = 10;

  const [searchInput, setSearchInput] = useState("");
  const [search, setSearch] = useState("");

  useEffect(() => {
    const id = window.setTimeout(() => setSearch(searchInput), 300);
    return () => window.clearTimeout(id);
  }, [searchInput]);

  useEffect(() => setPage(1), [search]);

  const { data, error, isLoading, mutate } = useAdminPosts(page, limit, search);
  const totalPages = data ? Math.max(1, Math.ceil(data.total / data.page_size)) : 1;

  const [editOpen, setEditOpen] = useState(false);
  const [editPost, setEditPost] = useState<AdminPost | null>(null);
  const [editTitle, setEditTitle] = useState("");
  const [editBody, setEditBody] = useState("");
  const [editSubmitting, setEditSubmitting] = useState(false);

  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deletePost, setDeletePost] = useState<AdminPost | null>(null);

  const formatDate = (dateString: string) => new Date(dateString).toLocaleString();

  const openEdit = (p: AdminPost) => {
    setEditPost(p);
    setEditTitle(p.title ?? "");
    setEditBody(p.body ?? "");
    setEditOpen(true);
  };

  const submitEdit = async () => {
    if (!editPost) return;
    try {
      setEditSubmitting(true);
      await auth.put(`/api/admin/posts/${editPost.id}`, { title: editTitle, body: editBody });
      toast.success(t("admin.posts.updated"));
      await mutate();
      setEditOpen(false);
      setEditPost(null);
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t("admin.errors.updateFailed")));
    } finally {
      setEditSubmitting(false);
    }
  };

  const openDelete = (p: AdminPost) => {
    setDeletePost(p);
    setDeleteOpen(true);
  };

  const confirmDelete = async () => {
    if (!deletePost) return;
    try {
      await auth.delete(`/api/admin/posts/${deletePost.id}`);
      toast.success(t("admin.posts.deleted"));
      if (data?.posts?.length === 1 && page > 1) {
        setPage((p) => Math.max(1, p - 1));
      } else {
        await mutate();
      }
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t("admin.errors.deleteFailed")));
    } finally {
      setDeletePost(null);
    }
  };

  const rows = useMemo(() => data?.posts ?? [], [data]);

  const renderContent = () => {
    if (isLoading) {
      return (
        <div className="flex items-center justify-center py-10 text-sm text-muted-foreground">
          <Loader2Icon className="mr-2 size-4 animate-spin" />
          {t("admin.loading")}
        </div>
      );
    }

    if (error) {
      return <div className="py-10 text-center text-sm text-destructive">{t("admin.posts.error")}</div>;
    }

    if (!rows.length) {
      return <div className="py-10 text-center text-sm text-muted-foreground">{t("admin.posts.empty")}</div>;
    }

    return (
      <>
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("admin.posts.id")}</TableHead>
                <TableHead>{t("admin.posts.titleCol")}</TableHead>
                <TableHead>{t("admin.posts.user")}</TableHead>
                <TableHead>{t("admin.posts.createdAt")}</TableHead>
                <TableHead>{t("admin.posts.updatedAt")}</TableHead>
                <TableHead className="text-right">{t("admin.posts.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((p) => (
                <TableRow key={p.id}>
                  <TableCell>{p.id}</TableCell>
                  <TableCell className="min-w-48 font-medium">{p.title || "-"}</TableCell>
                  <TableCell className="min-w-32">{p.user?.username ?? String(p.user_id)}</TableCell>
                  <TableCell className="min-w-40 text-muted-foreground">{formatDate(p.created_at)}</TableCell>
                  <TableCell className="min-w-40 text-muted-foreground">{formatDate(p.updated_at)}</TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-2">
                      <Button type="button" variant="outline" size="sm" onClick={() => openEdit(p)}>
                        <PencilIcon className="mr-2 size-4" />
                        {t("admin.posts.edit")}
                      </Button>
                      <Button type="button" variant="destructive" size="sm" onClick={() => openDelete(p)}>
                        <Trash2Icon className="mr-2 size-4" />
                        {t("admin.posts.delete")}
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>

        {totalPages > 1 && (
          <div className="mt-4 flex justify-center">
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
              >
                {t("admin.previous")}
              </Button>
              <Button type="button" variant="outline" size="sm" disabled>
                {t("admin.page", { current: page, total: totalPages })}
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page === totalPages}
              >
                {t("admin.next")}
              </Button>
            </div>
          </div>
        )}

        <div className="mt-3 text-center text-xs text-muted-foreground">{t("admin.posts.total", { count: data?.total ?? 0 })}</div>
      </>
    );
  };

  return (
    <>
      <Card className="border-0 shadow-none">
        <CardHeader className="flex flex-col gap-3 px-0 md:flex-row md:items-center md:justify-between">
          <CardTitle>{t("admin.posts.title")}</CardTitle>
          <div className="w-full md:w-80">
            <Input
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              placeholder={t("admin.posts.searchPlaceholder")}
              autoComplete="off"
            />
          </div>
        </CardHeader>
        <CardContent className="px-0">{renderContent()}</CardContent>
      </Card>

      <Dialog
        open={editOpen}
        onOpenChange={(open) => (!editSubmitting ? setEditOpen(open) : undefined)}
      >
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>{t("admin.posts.editTitle")}</DialogTitle>
            <DialogDescription className="sr-only">{t("admin.posts.editTitle")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="admin-post-title">{t("admin.posts.titleCol")}</Label>
              <Input
                id="admin-post-title"
                value={editTitle}
                onChange={(e) => setEditTitle(e.target.value)}
                disabled={editSubmitting}
                autoComplete="off"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="admin-post-body">{t("admin.posts.body")}</Label>
              <Textarea
                id="admin-post-body"
                value={editBody}
                onChange={(e) => setEditBody(e.target.value)}
                disabled={editSubmitting}
                rows={10}
              />
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setEditOpen(false)} disabled={editSubmitting}>
              {t("common.cancel")}
            </Button>
            <Button type="button" onClick={submitEdit} disabled={editSubmitting || !editBody.trim()}>
              {editSubmitting ? (
                <span className="inline-flex items-center gap-2">
                  <Loader2Icon className="size-4 animate-spin" />
                  {t("common.processing")}
                </span>
              ) : (
                t("common.save")
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t("admin.posts.deleteTitle")}
        description={
          deletePost ? t("admin.posts.deleteConfirm", { title: deletePost.title || "-" }) : t("admin.posts.deleteConfirmEmpty")
        }
        confirmLabel={t("admin.posts.delete")}
        onConfirm={confirmDelete}
      />
    </>
  );
}

