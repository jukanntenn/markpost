import { Metadata } from "next";
import AdminChannelsPage from "@/components/admin/AdminChannelsPage";

export const metadata: Metadata = {
  title: "Channels - Admin - Markpost",
};

export default function AdminChannels() {
  return <AdminChannelsPage />;
}
