"use client";

import { useEffect, useRef, useState } from "react";
import { useTranslations } from "next-intl";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";
import { Loader2Icon, TriangleAlertIcon } from "lucide-react";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import type { CreateTestPostRequest, CreateTestPostResponse } from "@/types/posts";

interface CreateTestPostModalProps {
  show: boolean;
  postKey: string;
  onHide: () => void;
  onSuccess: () => void;
}

async function createTestPost(postKey: string, data: CreateTestPostRequest): Promise<CreateTestPostResponse> {
  const response = await fetch(`/${postKey}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(data),
  });

  if (!response.ok) {
    let message = "Failed to create post";
    try {
      const text = await response.text();
      const error = JSON.parse(text);
      message = error.message || message;
    } catch {
    }
    throw new Error(message);
  }

  return response.json();
}

function CreateTestPostModal({ show, postKey, onHide, onSuccess }: CreateTestPostModalProps) {
  const t = useTranslations("createTestPost");
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [error, setError] = useState<string>("");
  const titleRef = useRef<HTMLInputElement | null>(null);

  const { mutate, isPending, reset } = useMutation({
    mutationFn: (data: CreateTestPostRequest) => createTestPost(postKey, data),
    onSuccess: () => {
      toast.success(t("successHeader"), {
        description: t("successBody"),
      });
      onSuccess();
      reset();
    },
    onError: (err: Error) => {
      setError(err.message);
    },
  });

  useEffect(() => {
    if (show) {
      setError("");
      setTimeout(() => titleRef.current?.focus(), 0);
    } else {
      setTitle("");
      setBody("");
      setError("");
    }
  }, [show]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!body.trim()) {
      setError(t("errorEmptyBody"));
      return;
    }
    mutate({ title: title.trim(), body });
  };

  return (
    <Dialog open={show} onOpenChange={(open) => (!open ? onHide() : undefined)}>
      <DialogContent className="sm:max-w-2xl">
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>{t("title")}</DialogTitle>
            <DialogDescription className="sr-only">
              {t("title")}
            </DialogDescription>
          </DialogHeader>

          {error && (
            <Alert variant="destructive">
              <TriangleAlertIcon />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <div className="space-y-2">
            <Label htmlFor="test-post-title">{t("titleLabel")}</Label>
            <Input
              id="test-post-title"
              ref={titleRef}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t("titlePlaceholder")}
              disabled={isPending}
              autoComplete="off"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="test-post-body">{t("bodyLabel")}</Label>
            <Textarea
              id="test-post-body"
              value={body}
              onChange={(e) => setBody(e.target.value)}
              placeholder={t("bodyPlaceholder")}
              disabled={isPending}
              rows={8}
            />
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onHide} disabled={isPending}>
              {t("cancel")}
            </Button>
            <Button type="submit" disabled={isPending || !body.trim()}>
              {isPending ? (
                <span className="inline-flex items-center gap-2">
                  <Loader2Icon className="size-4 animate-spin" />
                  {t("creating")}
                </span>
              ) : (
                t("create")
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export default CreateTestPostModal;
