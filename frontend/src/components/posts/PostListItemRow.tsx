"use client";

import { buildPostUrl } from "@/utils/url";
import { formatToLocalTime } from "@/lib/utils";
import type { PostListItem } from "@/types/posts";

interface PostListItemRowProps {
  post: PostListItem;
  className?: string;
  showSeconds?: boolean;
}

export function PostListItemRow({ post, className, showSeconds = true }: PostListItemRowProps) {
  return (
    <li className={className ?? "py-3"}>
      <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
        <a
          href={buildPostUrl(post.qid)}
          target="_blank"
          rel="noopener noreferrer"
          className="truncate text-sm font-medium text-primary underline-offset-4 hover:underline"
        >
          {post.title}
        </a>
        <span className="shrink-0 text-xs text-muted-foreground">
          {formatToLocalTime(post.created_at, showSeconds)}
        </span>
      </div>
    </li>
  );
}
