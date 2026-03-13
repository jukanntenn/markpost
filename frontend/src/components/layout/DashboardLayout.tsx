"use client";

import { useContext } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { UserInfoContext } from "@/components/UserInfoContext";
import { ThemeToggle } from "@/components/ThemeToggle";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  ChevronDownIcon,
  LogOutIcon,
  SettingsIcon,
  ShieldIcon,
  UserIcon,
} from "lucide-react";

export function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { logout, userInfo, isAuthenticated, isAdmin } = useContext(UserInfoContext);
  const router = useRouter();
  const pathname = usePathname();

  const handleLogout = () => {
    logout();
    router.replace("/login");
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
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button type="button" variant="ghost" className="gap-2">
                    <UserIcon className="size-4" />
                    <span className="hidden sm:inline">
                      {userInfo?.user?.username || "User"}
                    </span>
                    <ChevronDownIcon className="size-4 text-muted-foreground" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuLabel>
                    {userInfo?.user?.username || "User"}
                  </DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  {isAdmin() && (
                    <DropdownMenuItem onClick={() => router.push("/admin")}>
                      <ShieldIcon className="size-4" />
                      Admin
                    </DropdownMenuItem>
                  )}
                  <DropdownMenuItem onClick={() => router.push("/settings")}>
                    <SettingsIcon className="size-4" />
                    Settings
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem variant="destructive" onClick={handleLogout}>
                    <LogOutIcon className="size-4" />
                    Logout
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
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
