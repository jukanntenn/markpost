import { useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Loader2Icon } from "lucide-react";
import { toast } from "sonner";

import { useUsers, type User } from "../hooks/swr/useUsers";
import { UserInfoContext } from "@/components/UserInfoContext";
import { auth, getErrorMessage } from "../utils/api";

import { Badge } from "@/components/ui/badge";
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import ConfirmDialog from "@/components/admin/ConfirmDialog";

type Role = "admin" | "user";

export default function AdminUsers() {
  const { t } = useTranslation();
  const { userInfo } = useContext(UserInfoContext);
  const currentUserID = userInfo?.user?.id;

  const [page, setPage] = useState(1);
  const limit = 10;

  const { data, error, isLoading, mutate } = useUsers(page, limit);
  const totalPages = data ? Math.max(1, Math.ceil(data.total / data.page_size)) : 1;

  const [roleDialogOpen, setRoleDialogOpen] = useState(false);
  const [roleDialogUser, setRoleDialogUser] = useState<User | null>(null);
  const [roleSubmitting, setRoleSubmitting] = useState(false);

  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteUser, setDeleteUser] = useState<User | null>(null);

  const formatDate = (dateString: string) => new Date(dateString).toLocaleString();

  const canModifyUser = (u: User) => (typeof currentUserID === "number" ? u.id !== currentUserID : true);

  const roleLabel = useMemo(
    () => ({
      admin: t("admin.roleAdmin"),
      user: t("admin.roleUser"),
    }),
    [t]
  );

  const openRoleDialog = (u: User) => {
    setRoleDialogUser(u);
    setRoleDialogOpen(true);
  };

  const submitRoleChange = async (role: Role) => {
    if (!roleDialogUser) return;
    try {
      setRoleSubmitting(true);
      await auth.put(`/api/admin/users/${roleDialogUser.id}/role`, { role });
      toast.success(t("admin.users.roleUpdated"));
      await mutate();
      setRoleDialogOpen(false);
      setRoleDialogUser(null);
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t("admin.errors.updateFailed")));
    } finally {
      setRoleSubmitting(false);
    }
  };

  const openDeleteDialog = (u: User) => {
    setDeleteUser(u);
    setDeleteOpen(true);
  };

  const confirmDelete = async () => {
    if (!deleteUser) return;
    try {
      await auth.delete(`/api/admin/users/${deleteUser.id}`);
      toast.success(t("admin.users.deleted"));
      if (data?.users?.length === 1 && page > 1) {
        setPage((p) => Math.max(1, p - 1));
      } else {
        await mutate();
      }
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t("admin.errors.deleteFailed")));
    } finally {
      setDeleteUser(null);
    }
  };

  const renderRoleBadge = (user: User) =>
    user.role === "admin" ? (
      <Badge variant="destructive">{t("admin.roleAdmin")}</Badge>
    ) : (
      <Badge variant="secondary">{t("admin.roleUser")}</Badge>
    );

  const renderActions = (user: User) => {
    const disabled = !canModifyUser(user) || roleSubmitting;
    return (
      <div className="flex items-center justify-end gap-2">
        <Button type="button" variant="outline" size="sm" onClick={() => openRoleDialog(user)} disabled={disabled}>
          {t("admin.users.changeRole")}
        </Button>
        <Button
          type="button"
          variant="destructive"
          size="sm"
          onClick={() => openDeleteDialog(user)}
          disabled={disabled}
        >
          {t("admin.users.delete")}
        </Button>
      </div>
    );
  };

  const renderTableRow = (user: User) => (
    <TableRow key={user.id}>
      <TableCell>{user.id}</TableCell>
      <TableCell className="font-medium">{user.username}</TableCell>
      <TableCell>{renderRoleBadge(user)}</TableCell>
      <TableCell>{user.github_id ?? "-"}</TableCell>
      <TableCell className="text-muted-foreground">{formatDate(user.created_at)}</TableCell>
      <TableCell className="text-muted-foreground">{formatDate(user.updated_at)}</TableCell>
      <TableCell className="text-right">{renderActions(user)}</TableCell>
    </TableRow>
  );

  const renderMobileCard = (user: User) => (
    <div key={user.id} className="rounded-lg border p-4">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="truncate text-sm font-medium">{user.username}</div>
          <div className="text-xs text-muted-foreground">ID: {user.id}</div>
        </div>
        {renderRoleBadge(user)}
      </div>
      <div className="mt-3 space-y-1 text-sm">
        <div>
          <span className="font-medium">{t("admin.githubId")}:</span> {user.github_id ?? "-"}
        </div>
        <div>
          <span className="font-medium">{t("admin.createdAt")}:</span> {formatDate(user.created_at)}
        </div>
        <div>
          <span className="font-medium">{t("admin.updatedAt")}:</span> {formatDate(user.updated_at)}
        </div>
      </div>
      <div className="mt-4 flex items-center justify-end gap-2">{renderActions(user)}</div>
    </div>
  );

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
      return <div className="py-10 text-center text-sm text-destructive">{t("admin.error")}</div>;
    }

    if (!data || data.users.length === 0) {
      return <div className="py-10 text-center text-sm text-muted-foreground">{t("admin.noUsers")}</div>;
    }

    return (
      <>
        <div className="hidden md:block">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("admin.id")}</TableHead>
                <TableHead>{t("admin.username")}</TableHead>
                <TableHead>{t("admin.role")}</TableHead>
                <TableHead>{t("admin.githubId")}</TableHead>
                <TableHead>{t("admin.createdAt")}</TableHead>
                <TableHead>{t("admin.updatedAt")}</TableHead>
                <TableHead className="text-right">{t("admin.users.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>{data.users.map(renderTableRow)}</TableBody>
          </Table>
        </div>

        <div className="space-y-3 md:hidden">{data.users.map(renderMobileCard)}</div>

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

        <div className="mt-3 text-center text-xs text-muted-foreground">
          {t("admin.totalUsers", { count: data.total })}
        </div>
      </>
    );
  };

  return (
    <>
      <Card className="border-0 shadow-none">
        <CardHeader className="px-0">
          <CardTitle>{t("admin.users.title")}</CardTitle>
        </CardHeader>
        <CardContent className="px-0">{renderContent()}</CardContent>
      </Card>

      <Dialog
        open={roleDialogOpen}
        onOpenChange={(open) => (!roleSubmitting ? setRoleDialogOpen(open) : undefined)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("admin.users.changeRole")}</DialogTitle>
            <DialogDescription>
              {roleDialogUser
                ? t("admin.users.changeRoleFor", { username: roleDialogUser.username })
                : t("admin.users.changeRole")}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
            <Button type="button" variant="outline" onClick={() => setRoleDialogOpen(false)} disabled={roleSubmitting}>
              {t("common.cancel")}
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={() => submitRoleChange("user")}
              disabled={roleSubmitting || !roleDialogUser || roleDialogUser.role === "user"}
            >
              {roleLabel.user}
            </Button>
            <Button
              type="button"
              onClick={() => submitRoleChange("admin")}
              disabled={roleSubmitting || !roleDialogUser || roleDialogUser.role === "admin"}
            >
              {roleSubmitting ? (
                <span className="inline-flex items-center gap-2">
                  <Loader2Icon className="size-4 animate-spin" />
                  {t("common.processing")}
                </span>
              ) : (
                roleLabel.admin
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t("admin.users.deleteTitle")}
        description={
          deleteUser ? t("admin.users.deleteConfirm", { username: deleteUser.username }) : t("admin.users.deleteConfirmEmpty")
        }
        confirmLabel={t("admin.users.delete")}
        confirmVariant="destructive"
        onConfirm={confirmDelete}
      />
    </>
  );
}

