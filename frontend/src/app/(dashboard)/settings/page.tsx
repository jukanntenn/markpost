import { Metadata } from "next";
import SettingsPage from "@/components/settings/SettingsPage";

export const metadata: Metadata = {
  title: "Settings - Markpost",
};

export default function Settings() {
  return <SettingsPage />;
}
