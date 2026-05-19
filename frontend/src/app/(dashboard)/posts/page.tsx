import { buildPageMetadata } from "@/lib/metadata";
import PostsPage from "@/components/posts/PostsPage";

export const generateMetadata = buildPageMetadata("allPosts");

export default function Posts() {
  return <PostsPage />;
}
