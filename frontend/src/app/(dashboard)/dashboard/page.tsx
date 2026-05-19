import { buildPageMetadata } from "@/lib/metadata";
import DashboardPage from "@/components/dashboard/DashboardPage";

export const generateMetadata = buildPageMetadata("dashboard");

export default function Dashboard() {
  return <DashboardPage />;
}
