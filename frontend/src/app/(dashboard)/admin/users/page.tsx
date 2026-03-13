import { Metadata } from "next";
import AdminUsersPage from "@/components/admin/AdminUsersPage";

export const metadata: Metadata = {
  title: "Users - Admin - Markpost",
};

export default function AdminUsers() {
  return <AdminUsersPage />;
}
