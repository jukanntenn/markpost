import type { Metadata } from "next";

// buildPageMetadata returns a static build-time title for a route. Under static
// export the locale is resolved client-side (no server getTranslations), so the
// page title is a fixed English string at build time; client components update
// document.title per-locale via useTranslations + useEffect where localized
// titles matter (e.g. LoginPage).
const PAGE_TITLES: Record<string, string> = {
  login: "Login | Markpost",
  dashboard: "Dashboard | Markpost",
  settings: "Settings | Markpost",
  allPosts: "All Posts | Markpost",
  adminUsers: "Users - Admin | Markpost",
  adminPosts: "Posts - Admin | Markpost",
  adminChannels: "Channels - Admin | Markpost",
  adminDeliveryHistory: "Delivery History - Admin | Markpost",
};

export function buildPageMetadata(key: string) {
  const title = PAGE_TITLES[key] ?? "Markpost";
  return (): Metadata => ({ title });
}
