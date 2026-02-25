import React, { useState } from "react";
import { useUsers, type User } from "../hooks/swr/useUsers";
import { useTranslation } from "react-i18next";
import { Loader2Icon } from "lucide-react";

import { Badge } from "@/components/ui/badge";
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

const Admin: React.FC = () => {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const limit = 10;

  const { data, error, isLoading } = useUsers(page, limit);

  const totalPages = data ? Math.ceil(data.total / data.page_size) : 1;

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const renderTableRow = (user: User) => (
    <TableRow key={user.id}>
      <TableCell>{user.id}</TableCell>
      <TableCell className="font-medium">{user.username}</TableCell>
      <TableCell>
        {user.role === "admin" ? (
          <Badge variant="destructive">{t("admin.roleAdmin")}</Badge>
        ) : (
          <Badge variant="secondary">{t("admin.roleUser")}</Badge>
        )}
      </TableCell>
      <TableCell>{user.github_id ?? "-"}</TableCell>
      <TableCell className="text-muted-foreground">{formatDate(user.created_at)}</TableCell>
      <TableCell className="text-muted-foreground">{formatDate(user.updated_at)}</TableCell>
    </TableRow>
  );

  const renderMobileCard = (user: User) => (
    <div key={user.id} className="rounded-lg border p-4">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="truncate text-sm font-medium">{user.username}</div>
          <div className="text-xs text-muted-foreground">ID: {user.id}</div>
        </div>
        {user.role === "admin" ? (
          <Badge variant="destructive">{t("admin.roleAdmin")}</Badge>
        ) : (
          <Badge variant="secondary">{t("admin.roleUser")}</Badge>
        )}
      </div>
      <div className="mt-3 space-y-1 text-sm">
        <div>
          <span className="font-medium">{t("admin.githubId")}:</span>{" "}
          {user.github_id ?? "-"}
        </div>
        <div>
          <span className="font-medium">{t("admin.createdAt")}:</span>{" "}
          {formatDate(user.created_at)}
        </div>
        <div>
          <span className="font-medium">{t("admin.updatedAt")}:</span>{" "}
          {formatDate(user.updated_at)}
        </div>
      </div>
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
      return (
        <div className="py-10 text-center text-sm text-destructive">
          {t("admin.error")}
        </div>
      );
    }

    if (!data || data.users.length === 0) {
      return (
        <div className="py-10 text-center text-sm text-muted-foreground">
          {t("admin.noUsers")}
        </div>
      );
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
    <div className="mx-auto max-w-5xl">
      <Card>
        <CardHeader>
          <CardTitle>{t("admin.title")}</CardTitle>
        </CardHeader>
        <CardContent>{renderContent()}</CardContent>
      </Card>
    </div>
  );
};

export default Admin;
