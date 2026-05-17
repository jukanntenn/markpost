"use client";

import { useTranslations } from "next-intl";
import { UserPlusIcon } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { formatToLocalTime } from "@/lib/utils";

import { adminApi } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { QueryState } from "@/components/ui/query-state";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export function AdminUsersPage() {
  const t = useTranslations("admin");

  const { data, isLoading, error } = useQuery({
    queryKey: ["admin", "users"],
    queryFn: () => adminApi.listUsers(),
  });

  const users = data?.users || [];

  return (
    <div>
      <div className="mb-6 flex items-center justify-between md:mb-8 lg:mb-12">
        <h1 className="font-display text-[28px] font-bold tracking-tight">{t("users.title")}</h1>
        <Button>
          <UserPlusIcon className="mr-2 size-4" />
          {t("users.addUser")}
        </Button>
      </div>

      <QueryState isLoading={isLoading} error={error} loadingText={t("loading")} errorText={t("error")}>
        <div className="rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("id")}</TableHead>
                <TableHead>{t("username")}</TableHead>
                <TableHead>{t("role")}</TableHead>
                <TableHead>{t("createdAt")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={4} className="text-center text-muted-foreground">
                    {t("noUsers")}
                  </TableCell>
                </TableRow>
              ) : (
                users.map((user) => (
                  <TableRow key={user.id}>
                    <TableCell>{user.id}</TableCell>
                    <TableCell>{user.username}</TableCell>
                    <TableCell>{user.role}</TableCell>
                    <TableCell>{formatToLocalTime(user.created_at)}</TableCell>
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

export default AdminUsersPage;
