import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useCreateTestPost } from "../hooks/swr/useCreateTestPost";
import * as api from "../utils/api";
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

interface CreateTestPostModalProps {
  show: boolean;
  postKey: string;
  onHide: () => void;
  onSuccess: () => void;
}

function CreateTestPostModal({ show, postKey, onHide, onSuccess }: CreateTestPostModalProps) {
  const { t } = useTranslation();
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [error, setError] = useState<string>("");
  const titleRef = useRef<HTMLInputElement | null>(null);

  const { trigger, isMutating, reset } = useCreateTestPost(postKey);

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
      setError(t("createTestPost.errorEmptyBody"));
      return;
    }
    try {
      await trigger({ title: title.trim(), body });
      toast.success(t("createTestPost.successHeader"), {
        description: t("createTestPost.successBody"),
      });
      onSuccess();
      reset();
    } catch (err: unknown) {
      const msg = api.getErrorMessage(err, t("createTestPost.errorServer"));
      setError(msg);
    }
  };

  return (
    <Dialog open={show} onOpenChange={(open) => (!open ? onHide() : undefined)}>
      <DialogContent className="sm:max-w-2xl">
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>{t("createTestPost.title")}</DialogTitle>
            <DialogDescription className="sr-only">
              {t("createTestPost.bodyLabel")}
            </DialogDescription>
          </DialogHeader>

          {error && (
            <Alert variant="destructive">
              <TriangleAlertIcon />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <div className="space-y-2">
            <Label htmlFor="test-post-title">{t("createTestPost.titleLabel")}</Label>
            <Input
              id="test-post-title"
              ref={titleRef}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t("createTestPost.titlePlaceholder")}
              disabled={isMutating}
              autoComplete="off"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="test-post-body">{t("createTestPost.bodyLabel")}</Label>
            <Textarea
              id="test-post-body"
              value={body}
              onChange={(e) => setBody(e.target.value)}
              placeholder={t("createTestPost.bodyPlaceholder")}
              disabled={isMutating}
              rows={8}
            />
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onHide} disabled={isMutating}>
              {t("createTestPost.cancel")}
            </Button>
            <Button type="submit" disabled={isMutating || !body.trim()}>
              {isMutating ? (
                <span className="inline-flex items-center gap-2">
                  <Loader2Icon className="size-4 animate-spin" />
                  {t("createTestPost.creating")}
                </span>
              ) : (
                t("createTestPost.create")
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export default CreateTestPostModal;
