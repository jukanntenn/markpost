"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { useAuthStore } from "@/stores/auth";
import { useAuthReady } from "@/hooks/useAuthReady";
import { authApi } from "@/lib/api";
import { ThemeToggle } from "@/components/ThemeToggle";
import { Button, buttonVariants } from "@/components/ui/button";
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
  const t = useTranslations("navigation");
  const tCommon = useTranslations("common");
  const user = useAuthStore((state) => state.user);
  const { isAuthenticated, isAdmin } = useAuthReady();
  const logout = useAuthStore((state) => state.logout);

  const [scrolled, setScrolled] = useState(false);
  const mainRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleScroll = () => {
      setScrolled(window.scrollY > 0);
    };
    window.addEventListener("scroll", handleScroll, { passive: true });
    handleScroll();
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

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
      <header
        className={`sticky top-0 z-50 w-full bg-background/80 backdrop-blur transition-[border-color] duration-150 ${scrolled ? "border-b" : ""}`}
      >
        <div className="mx-auto flex h-14 max-w-[1200px] items-center justify-between px-6">
          <Link href="/dashboard" className={buttonVariants({ variant: "ghost", className: "h-9 px-2" })}>
            <Image src="/markpost.svg" alt="Markpost" className="h-6 w-auto" width={24} height={24} />
          </Link>
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
      <main ref={mainRef} className="mx-auto w-full max-w-[1200px] px-6 py-6 md:py-8 lg:py-12">
        {children}
      </main>
    </>
  );
}
