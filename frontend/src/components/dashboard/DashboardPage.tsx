"use client";

import { useState } from "react";
import Link from "next/link";
import {
  BookIcon,
  CopyIcon,
  EyeIcon,
  EyeOffIcon,
  FilePlusIcon,
  FileTextIcon,
  KeyIcon,
  Loader2Icon,
  TriangleAlertIcon,
} from "lucide-react";

import { buildPostUrl } from "@/utils/url";
import CreateTestPostModal from "@/components/CreateTestPostModal";
import { usePostKey } from "@/hooks/usePostKey";
import { usePosts } from "@/hooks/usePosts";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const formatToLocalTime = (utcString: string): string => {
  if (!utcString) return "";

  const date = new Date(utcString);

  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  const seconds = String(date.getSeconds()).padStart(2, "0");

  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
};

export function DashboardPage() {
  const [showKey, setShowKey] = useState(false);
  const [copySuccess, setCopySuccess] = useState(false);
  const [showCreateModal, setShowCreateModal] = useState(false);

  const { data: postKeyData, isLoading: keyLoading, error: keyError } = usePostKey();
  const { data: postsData, isLoading: postsLoading, error: postsError, refetch: refetchPosts } = usePosts(1, 10);

  const postKey = postKeyData?.post_key || "";
  const createdAt = postKeyData?.created_at || "";
  const recentPosts = postsData?.posts || [];

  const handleCopyKey = async () => {
    if (postKey) {
      try {
        await navigator.clipboard.writeText(postKey);
        setCopySuccess(true);
        setTimeout(() => setCopySuccess(false), 2000);
      } catch (err) {
        console.error("Failed to copy text: ", err);
      }
    }
  };

  return (
    <>
      <div className="grid gap-4 xl:grid-cols-2">
        <div className="flex flex-col gap-4">
          <Card>
            <CardHeader className="flex-row items-center justify-between space-y-0">
              <div className="flex items-center gap-2">
                <KeyIcon className="size-4" />
                <CardTitle className="text-base">Post Key</CardTitle>
              </div>
            </CardHeader>
            <CardContent className="space-y-3">
              {keyLoading ? (
                <div className="flex flex-col items-center justify-center gap-2 rounded-lg border bg-muted/40 p-6 text-center">
                  <Loader2Icon className="size-5 animate-spin" />
                  <p className="text-sm text-muted-foreground">
                    Loading post key...
                  </p>
                </div>
              ) : keyError ? (
                <Alert variant="destructive">
                  <TriangleAlertIcon />
                  <AlertDescription>
                    Error loading post key
                  </AlertDescription>
                </Alert>
              ) : (
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0 flex-1 space-y-2">
                    <div className="font-mono text-base">
                      {showKey ? postKey : "•".repeat(postKey.length)}
                    </div>
                    {copySuccess && (
                      <Badge variant="secondary">Copied!</Badge>
                    )}
                    <div className="text-xs text-muted-foreground">
                      Created at: {formatToLocalTime(createdAt)}
                    </div>
                  </div>
                  <div className="flex shrink-0 items-center gap-1">
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => setShowKey(!showKey)}
                      title={showKey ? "Hide key" : "Show key"}
                    >
                      {showKey ? (
                        <EyeOffIcon className="size-4" />
                      ) : (
                        <EyeIcon className="size-4" />
                      )}
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={handleCopyKey}
                      title="Copy key"
                    >
                      <CopyIcon className="size-4" />
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => setShowCreateModal(true)}
                      title="Create test post"
                    >
                      <FilePlusIcon className="size-4" />
                    </Button>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex-row items-center justify-between space-y-0">
              <div className="flex items-center gap-2">
                <BookIcon className="size-4" />
                <CardTitle className="text-base">
                  Documentation
                </CardTitle>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                Check out the{" "}
                <a
                  href="https://github.com/jukanntenn/markpost?tab=readme-ov-file#apis"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-foreground underline underline-offset-4"
                >
                  API documentation
                </a>{" "}
                for more information.
              </p>
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader className="flex-row items-center justify-between space-y-0">
            <div className="flex items-center gap-2">
              <FileTextIcon className="size-4" />
              <CardTitle className="text-base">Recent Posts</CardTitle>
            </div>
            <Button
              type="button"
              variant="link"
              className="h-auto p-0"
              asChild
            >
              <Link href="/posts">View all</Link>
            </Button>
          </CardHeader>
          <CardContent>
            {postsLoading ? (
              <div className="flex flex-col items-center justify-center gap-2 py-6 text-center">
                <Loader2Icon className="size-5 animate-spin" />
                <p className="text-sm text-muted-foreground">
                  Loading posts...
                </p>
              </div>
            ) : postsError ? (
              <Alert variant="destructive">
                <TriangleAlertIcon />
                <AlertDescription>Error loading posts</AlertDescription>
              </Alert>
            ) : recentPosts.length === 0 ? (
              <p className="py-6 text-center text-sm text-muted-foreground">
                No posts yet
              </p>
            ) : (
              <ul className="-mx-2 divide-y">
                {recentPosts.map((p) => (
                  <li key={p.id} className="px-2 py-3">
                    <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
                      <a
                        href={buildPostUrl(p.qid)}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="truncate text-sm font-medium underline-offset-4 hover:underline"
                      >
                        {p.title}
                      </a>
                      <span className="shrink-0 text-xs text-muted-foreground">
                        {formatToLocalTime(p.created_at)}
                      </span>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>
    <CreateTestPostModal
      show={showCreateModal}
      postKey={postKey}
      onHide={() => setShowCreateModal(false)}
      onSuccess={() => {
        setShowCreateModal(false);
        refetchPosts();
      }}
    />
    </>
  );
}

export default DashboardPage;
