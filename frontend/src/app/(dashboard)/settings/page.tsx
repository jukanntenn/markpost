import { buildPageMetadata } from "@/lib/metadata";
import SettingsPage from "@/components/settings/SettingsPage";

export const generateMetadata = buildPageMetadata("settings");

export default function Settings() {
  return <SettingsPage />;
}
