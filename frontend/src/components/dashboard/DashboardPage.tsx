"use client";

import { useState } from "react";
import Link from "next/link";
import { useTranslations } from "next-intl";
import {
  BookIcon,
  CopyIcon,
  EyeIcon,
  EyeOffIcon,
  FilePlusIcon,
  FileTextIcon,
  KeyIcon,
  Link2Icon,
} from "lucide-react";

import { buildPostUrl } from "@/utils/url";
import { formatToLocalTime } from "@/lib/utils";
import CreateTestPostModal from "@/components/CreateTestPostModal";
import { PostListItemRow } from "@/components/posts/PostListItemRow";
import { usePostKey } from "@/hooks/usePostKey";
import { usePosts } from "@/hooks/usePosts";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Menu } from "@/components/ui/menu";
import { QueryState } from "@/components/ui/query-state";

export function DashboardPage() {
  const [showKey, setShowKey] = useState(false);
  const [copySuccess, setCopySuccess] = useState(false);
  const [showCreateModal, setShowCreateModal] = useState(false);

  const t = useTranslations("dashboard");
  const tPostKey = useTranslations("dashboard.postKey");
  const tRecentPosts = useTranslations("dashboard.recentPosts");
  const tDocs = useTranslations("dashboard.documentation");

  const { data: postKeyData, isLoading: keyLoading, error: keyError } = usePostKey();
  const { data: postsData, isLoading: postsLoading, error: postsError, refetch: refetchPosts } = usePosts(1, 10);

  const postKey = postKeyData?.post_key || "";
  const createdAt = postKeyData?.created_at || "";
  const recentPosts = postsData?.posts || [];

  const handleCopy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopySuccess(true);
      setTimeout(() => setCopySuccess(false), 2000);
    } catch (err) {
      console.error("Failed to copy: ", err);
    }
  };

  return (
    <>
      <div className="grid gap-4 xl:grid-cols-2 xl:gap-6">
        <div className="flex flex-col gap-4 xl:gap-6">
          <Card>
            <CardHeader className="flex-row items-center justify-between space-y-0">
              <div className="flex items-center gap-2">
                <KeyIcon className="size-4" />
                <CardTitle className="text-base">{tPostKey("title")}</CardTitle>
              </div>
            </CardHeader>
            <CardContent className="space-y-3">
              <QueryState
                isLoading={keyLoading}
                error={keyError}
                loadingText={tPostKey("loadingKey")}
                errorText={tPostKey("errorLoadingKey")}
                loadingClassName="flex flex-col items-center justify-center gap-2 rounded-lg border bg-muted/40 p-6 text-center"
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0 flex-1 space-y-2">
                    <div className="font-mono text-base">
                      {showKey ? postKey : "•".repeat(postKey.length)}
                    </div>
                    {copySuccess && (
                      <Badge variant="accent">{tPostKey("copied")}</Badge>
                    )}
                    <div className="text-xs text-muted-foreground">
                      {tPostKey("createdAt")}: {formatToLocalTime(createdAt)}
                    </div>
                  </div>
                  <div className="flex shrink-0 items-center gap-1">
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => setShowKey(!showKey)}
                      title={showKey ? tPostKey("hideKey") : tPostKey("showKey")}
                    >
                      {showKey ? (
                        <EyeOffIcon className="size-4" />
                      ) : (
                        <EyeIcon className="size-4" />
                      )}
                    </Button>
                    <Menu>
                      <Menu.Trigger
                        render={
                          <Button type="button" variant="ghost" size="icon" title={tPostKey("copyKey")} />
                        }
                      >
                        <CopyIcon className="size-4" />
                      </Menu.Trigger>
                      <Menu.Popup>
                        <Menu.Item onClick={() => postKey && handleCopy(postKey)}>
                          <KeyIcon className="size-4" />
                          {tPostKey("copyPostKey")}
                        </Menu.Item>
                        <Menu.Item
                          onClick={() => postKey && handleCopy(`${window.location.origin}${buildPostUrl(postKey)}`)}
                        >
                          <Link2Icon className="size-4" />
                          {tPostKey("copyPostUrl")}
                        </Menu.Item>
                      </Menu.Popup>
                    </Menu>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => setShowCreateModal(true)}
                      title={tPostKey("createTestPost")}
                    >
                      <FilePlusIcon className="size-4" />
                    </Button>
                  </div>
                </div>
              </QueryState>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex-row items-center justify-between space-y-0">
              <div className="flex items-center gap-2">
                <BookIcon className="size-4" />
                <CardTitle className="text-base">
                  {tDocs("title")}
                </CardTitle>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                {tDocs("content")}{" "}
                <a
                  href="https://github.com/jukanntenn/markpost?tab=readme-ov-file#apis"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-primary underline underline-offset-4"
                >
                  {tDocs("apiLink")}
                </a>{" "}
                {tDocs("content2")}
              </p>
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader className="flex-row items-center justify-between space-y-0">
            <div className="flex items-center gap-2">
              <FileTextIcon className="size-4" />
              <CardTitle className="text-base">{tRecentPosts("title")}</CardTitle>
            </div>
            <Button
              type="button"
              variant="link"
              className="h-auto p-0"
              asChild
            >
              <Link href="/posts">{tRecentPosts("viewAll")}</Link>
            </Button>
          </CardHeader>
          <CardContent>
            <QueryState isLoading={postsLoading} error={postsError} loadingText={tRecentPosts("loading")} errorText={tRecentPosts("error")}>
              {recentPosts.length === 0 ? (
                <p className="py-6 text-center text-sm text-muted-foreground">
                  {tRecentPosts("empty")}
                </p>
              ) : (
                <ul className="-mx-2 divide-y">
                  {recentPosts.map((p) => (
                    <PostListItemRow key={p.id} post={p} className="px-2 py-3" />
                  ))}
                </ul>
              )}
            </QueryState>
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
