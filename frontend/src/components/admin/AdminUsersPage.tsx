"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Loader2Icon, TriangleAlertIcon, UserPlusIcon } from "lucide-react";
import { useQuery } from "@tanstack/react-query";

import { request } from "@/lib/api";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface User {
  id: number;
  username: string;
  role: string;
  created_at: string;
}

interface UsersResponse {
  users: User[];
}

export function AdminUsersPage() {
  const t = useTranslations("admin");

  const { data, isLoading, error } = useQuery<UsersResponse>({
    queryKey: ["admin", "users"],
    queryFn: () => request<UsersResponse>("/api/v1/admin/users"),
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
                    <TableCell>{new Date(user.created_at).toLocaleString()}</TableCell>
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

export default AdminUsersPage;
