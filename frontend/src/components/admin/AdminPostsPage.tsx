"use client";

import { Loader2Icon, TriangleAlertIcon } from "lucide-react";
import { useQuery } from "@tanstack/react-query";

import { authFetcher } from "@/lib/api/fetcher";
import { buildPostUrl } from "@/utils/url";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface Post {
  id: string;
  qid: string;
  title: string;
  user_id: number;
  created_at: string;
}

interface PostsResponse {
  posts: Post[];
}

export function AdminPostsPage() {
  const { data, isLoading, error } = useQuery<PostsResponse>({
    queryKey: ["admin", "posts"],
    queryFn: () => authFetcher("/api/admin/posts"),
  });

  const posts = data?.posts || [];

  return (
    <div>
      <h1 className="mb-6 text-2xl font-semibold">Posts</h1>

      {isLoading ? (
        <div className="flex flex-col items-center justify-center gap-2 py-6 text-center">
          <Loader2Icon className="size-5 animate-spin" />
          <p className="text-sm text-muted-foreground">Loading posts...</p>
        </div>
      ) : error ? (
        <Alert variant="destructive">
          <TriangleAlertIcon />
          <AlertDescription>Error loading posts</AlertDescription>
        </Alert>
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>Title</TableHead>
                <TableHead>User ID</TableHead>
                <TableHead>Created At</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {posts.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={4} className="text-center text-muted-foreground">
                    No posts found
                  </TableCell>
                </TableRow>
              ) : (
                posts.map((post) => (
                  <TableRow key={post.id}>
                    <TableCell>{post.id.slice(0, 8)}...</TableCell>
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
                    <TableCell>{post.user_id}</TableCell>
                    <TableCell>{new Date(post.created_at).toLocaleString()}</TableCell>
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

export default AdminPostsPage;
