import { Metadata } from "next";
import DashboardPage from "@/components/dashboard/DashboardPage";

export const metadata: Metadata = {
  title: "Dashboard - Markpost",
};

export default function Dashboard() {
  return <DashboardPage />;
}
