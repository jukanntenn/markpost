import { Metadata } from "next";
import AdminPostsPage from "@/components/admin/AdminPostsPage";

export const metadata: Metadata = {
  title: "Posts - Admin - Markpost",
};

export default function AdminPosts() {
  return <AdminPostsPage />;
}
