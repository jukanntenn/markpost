"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useAuthStore } from "@/stores/auth";
import { authApi } from "@/lib/api/auth";
import { ThemeToggle } from "@/components/ThemeToggle";
import { Button } from "@/components/ui/button";
import { Menu } from "@/components/ui/menu";
import {
  ChevronDownIcon,
  LogOutIcon,
  SettingsIcon,
  ShieldIcon,
  UserIcon,
} from "lucide-react";

export function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const t = useTranslations("navigation");
  const tCommon = useTranslations("common");
  const user = useAuthStore((state) => state.user);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated());
  const isAdmin = useAuthStore((state) => state.isAdmin());
  const logout = useAuthStore((state) => state.logout);

  const handleLogout = async () => {
    try {
      await authApi.logout();
    } catch {
    } finally {
      logout();
      router.replace("/login");
    }
  };

  return (
    <>
      <header className="sticky top-0 z-50 w-full border-b bg-background/80 backdrop-blur">
        <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-4">
          <Button
            type="button"
            variant="ghost"
            className="h-9 px-2"
            asChild
          >
            <Link href="/dashboard">
              <img src="/markpost.svg" alt="Markpost" className="h-6 w-auto" />
            </Link>
          </Button>
          <div className="flex items-center gap-2">
            <ThemeToggle />
            {isAuthenticated && (
              <Menu.Root>
                <Menu.Trigger
                  render={
                    <Button type="button" variant="ghost" className="gap-2" />
                  }
                >
                  <UserIcon className="size-4" />
                  <span className="hidden sm:inline">
                    {user?.username || tCommon("user")}
                  </span>
                  <ChevronDownIcon className="size-4 text-muted-foreground" />
                </Menu.Trigger>
                <Menu.Popup>
                  <Menu.Group>
                    <Menu.Label>
                      {user?.username || tCommon("user")}
                    </Menu.Label>
                  </Menu.Group>
                  <Menu.Separator />
                  {isAdmin && (
                    <Menu.Item onClick={() => router.push("/admin")}>
                      <ShieldIcon className="size-4" />
                      {t("userMenu.admin")}
                    </Menu.Item>
                  )}
                  <Menu.Item onClick={() => router.push("/settings")}>
                    <SettingsIcon className="size-4" />
                    {t("userMenu.settings")}
                  </Menu.Item>
                  <Menu.Separator />
                  <Menu.Item variant="destructive" onClick={handleLogout}>
                    <LogOutIcon className="size-4" />
                    {t("userMenu.logout")}
                  </Menu.Item>
                </Menu.Popup>
              </Menu.Root>
            )}
          </div>
        </div>
      </header>
      <main className="mx-auto w-full max-w-7xl px-4 py-6">
        {children}
      </main>
    </>
  );
}
