import { Metadata } from "next";
import PostsPage from "@/components/posts/PostsPage";

export const metadata: Metadata = {
  title: "Posts - Markpost",
};

export default function Posts() {
  return <PostsPage />;
}
