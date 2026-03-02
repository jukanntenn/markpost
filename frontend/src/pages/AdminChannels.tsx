import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Loader2Icon, PencilIcon, Trash2Icon } from "lucide-react";
import { toast } from "sonner";

import { auth, getErrorMessage } from "../utils/api";
import { useAdminChannels, type AdminChannel } from "../hooks/swr/useAdminChannels";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import ConfirmDialog from "@/components/admin/ConfirmDialog";

export default function AdminChannels() {
  const { t } = useTranslation();
  const { data, error, isLoading, mutate } = useAdminChannels();

  const [editOpen, setEditOpen] = useState(false);
  const [editChannel, setEditChannel] = useState<AdminChannel | null>(null);
  const [editName, setEditName] = useState("");
  const [editWebhookURL, setEditWebhookURL] = useState("");
  const [editKeywords, setEditKeywords] = useState("");
  const [editEnabled, setEditEnabled] = useState(true);
  const [editSubmitting, setEditSubmitting] = useState(false);

  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteChannel, setDeleteChannel] = useState<AdminChannel | null>(null);

  const rows = useMemo(() => data?.channels ?? [], [data]);

  const formatDate = (dateString: string) => new Date(dateString).toLocaleString();

  const openEdit = (c: AdminChannel) => {
    setEditChannel(c);
    setEditName(c.name ?? "");
    setEditWebhookURL(c.webhook_url ?? "");
    setEditKeywords(c.keywords ?? "");
    setEditEnabled(Boolean(c.enabled));
    setEditOpen(true);
  };

  const submitEdit = async () => {
    if (!editChannel) return;
    try {
      setEditSubmitting(true);
      await auth.put(`/api/admin/channels/${editChannel.id}`, {
        name: editName,
        webhook_url: editWebhookURL,
        keywords: editKeywords,
        enabled: editEnabled,
      });
      toast.success(t("admin.channels.updated"));
      await mutate();
      setEditOpen(false);
      setEditChannel(null);
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t("admin.errors.updateFailed")));
    } finally {
      setEditSubmitting(false);
    }
  };

  const openDelete = (c: AdminChannel) => {
    setDeleteChannel(c);
    setDeleteOpen(true);
  };

  const confirmDelete = async () => {
    if (!deleteChannel) return;
    try {
      await auth.delete(`/api/admin/channels/${deleteChannel.id}`);
      toast.success(t("admin.channels.deleted"));
      await mutate();
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t("admin.errors.deleteFailed")));
    } finally {
      setDeleteChannel(null);
    }
  };

  const renderContent = () => {
    if (isLoading) {
      return (
        <div className="flex items-center justify-center py-10 text-sm text-muted-foreground">
          <Loader2Icon className="mr-2 size-4 animate-spin" />
          {t("admin.loading")}
        </div>
      );
    }

    if (error) {
      return <div className="py-10 text-center text-sm text-destructive">{t("admin.channels.error")}</div>;
    }

    if (!rows.length) {
      return <div className="py-10 text-center text-sm text-muted-foreground">{t("admin.channels.empty")}</div>;
    }

    return (
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("admin.channels.id")}</TableHead>
              <TableHead>{t("admin.channels.user")}</TableHead>
              <TableHead>{t("admin.channels.kind")}</TableHead>
              <TableHead>{t("admin.channels.name")}</TableHead>
              <TableHead>{t("admin.channels.enabled")}</TableHead>
              <TableHead>{t("admin.channels.createdAt")}</TableHead>
              <TableHead>{t("admin.channels.updatedAt")}</TableHead>
              <TableHead className="text-right">{t("admin.channels.actions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((ch) => (
              <TableRow key={ch.id}>
                <TableCell>{ch.id}</TableCell>
                <TableCell className="min-w-40 font-medium">{ch.username || String(ch.user_id)}</TableCell>
                <TableCell className="min-w-24">{ch.kind}</TableCell>
                <TableCell className="min-w-40">{ch.name || "-"}</TableCell>
                <TableCell className="min-w-24">{ch.enabled ? t("common.yes") : t("common.no")}</TableCell>
                <TableCell className="min-w-40 text-muted-foreground">{formatDate(ch.created_at)}</TableCell>
                <TableCell className="min-w-40 text-muted-foreground">{formatDate(ch.updated_at)}</TableCell>
                <TableCell className="text-right">
                  <div className="flex items-center justify-end gap-2">
                    <Button type="button" variant="outline" size="sm" onClick={() => openEdit(ch)}>
                      <PencilIcon className="mr-2 size-4" />
                      {t("admin.channels.edit")}
                    </Button>
                    <Button type="button" variant="destructive" size="sm" onClick={() => openDelete(ch)}>
                      <Trash2Icon className="mr-2 size-4" />
                      {t("admin.channels.delete")}
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    );
  };

  return (
    <>
      <Card className="border-0 shadow-none">
        <CardHeader className="px-0">
          <CardTitle>{t("admin.channels.title")}</CardTitle>
        </CardHeader>
        <CardContent className="px-0">{renderContent()}</CardContent>
      </Card>

      <Dialog
        open={editOpen}
        onOpenChange={(open) => (!editSubmitting ? setEditOpen(open) : undefined)}
      >
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>{t("admin.channels.editTitle")}</DialogTitle>
            <DialogDescription className="sr-only">{t("admin.channels.editTitle")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="admin-channel-name">{t("admin.channels.name")}</Label>
                <Input
                  id="admin-channel-name"
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                  disabled={editSubmitting}
                  autoComplete="off"
                />
              </div>
              <div className="flex items-center justify-between rounded-md border px-3 py-2">
                <div className="flex flex-col">
                  <span className="text-sm font-medium">{t("admin.channels.enabled")}</span>
                  <span className="text-xs text-muted-foreground">{t("admin.channels.enabledHelp")}</span>
                </div>
                <Switch checked={editEnabled} onCheckedChange={setEditEnabled} disabled={editSubmitting} />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="admin-channel-webhook">{t("admin.channels.webhookUrl")}</Label>
              <Textarea
                id="admin-channel-webhook"
                value={editWebhookURL}
                onChange={(e) => setEditWebhookURL(e.target.value)}
                disabled={editSubmitting}
                rows={3}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="admin-channel-keywords">{t("admin.channels.keywords")}</Label>
              <Input
                id="admin-channel-keywords"
                value={editKeywords}
                onChange={(e) => setEditKeywords(e.target.value)}
                disabled={editSubmitting}
                placeholder={t("admin.channels.keywordsPlaceholder")}
                autoComplete="off"
              />
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setEditOpen(false)} disabled={editSubmitting}>
              {t("common.cancel")}
            </Button>
            <Button type="button" onClick={submitEdit} disabled={editSubmitting}>
              {editSubmitting ? (
                <span className="inline-flex items-center gap-2">
                  <Loader2Icon className="size-4 animate-spin" />
                  {t("common.processing")}
                </span>
              ) : (
                t("common.save")
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t("admin.channels.deleteTitle")}
        description={
          deleteChannel
            ? t("admin.channels.deleteConfirm", { name: deleteChannel.name || "-" })
            : t("admin.channels.deleteConfirmEmpty")
        }
        confirmLabel={t("admin.channels.delete")}
        onConfirm={confirmDelete}
      />
    </>
  );
}

