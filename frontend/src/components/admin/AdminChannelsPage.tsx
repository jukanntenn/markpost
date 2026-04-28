"use client";

import { Loader2Icon, TriangleAlertIcon } from "lucide-react";
import { useQuery } from "@tanstack/react-query";

import { request } from "@/lib/api";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface Channel {
  id: number;
  name: string;
  type: string;
  enabled: boolean;
  user_id: number;
  webhook_url: string;
  created_at: string;
}

interface ChannelsResponse {
  channels: Channel[];
}

export function AdminChannelsPage() {
  const { data, isLoading, error } = useQuery<ChannelsResponse>({
    queryKey: ["admin", "channels"],
    queryFn: () => request<ChannelsResponse>("/api/v1/admin/channels"),
  });

  const channels = data?.channels || [];

  return (
    <div>
      <h1 className="mb-6 text-2xl font-semibold">Channels</h1>

      {isLoading ? (
        <div className="flex flex-col items-center justify-center gap-2 py-6 text-center">
          <Loader2Icon className="size-5 animate-spin" />
          <p className="text-sm text-muted-foreground">Loading channels...</p>
        </div>
      ) : error ? (
        <Alert variant="destructive">
          <TriangleAlertIcon />
          <AlertDescription>Error loading channels</AlertDescription>
        </Alert>
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Enabled</TableHead>
                <TableHead>Created At</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {channels.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-muted-foreground">
                    No channels found
                  </TableCell>
                </TableRow>
              ) : (
                channels.map((channel) => (
                  <TableRow key={channel.id}>
                    <TableCell>{channel.id}</TableCell>
                    <TableCell>{channel.name}</TableCell>
                    <TableCell>{channel.type}</TableCell>
                    <TableCell>{channel.enabled ? "Yes" : "No"}</TableCell>
                    <TableCell>{new Date(channel.created_at).toLocaleString()}</TableCell>
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

export default AdminChannelsPage;
