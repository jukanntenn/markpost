"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";
import { UsersIcon, FileTextIcon, RadioIcon, SendIcon } from "lucide-react";
import type { LucideIcon } from "lucide-react";

interface AdminNavItem {
  href: string;
  labelKey: string;
  icon: LucideIcon;
}

const adminNavItems: AdminNavItem[] = [
  { href: "/admin/users", labelKey: "nav.users", icon: UsersIcon },
  { href: "/admin/posts", labelKey: "nav.posts", icon: FileTextIcon },
  { href: "/admin/delivery/channels", labelKey: "nav.channels", icon: RadioIcon },
  { href: "/admin/delivery/history", labelKey: "nav.history", icon: SendIcon },
];

export function AdminLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const t = useTranslations("admin");

  return (
    <div className="flex gap-6">
      <aside className="w-64 shrink-0 border-r border-[#E7E5E4]">
        <nav className="space-y-1">
          {adminNavItems.map((item) => {
            const Icon = item.icon;
            const isActive = pathname === item.href || pathname?.startsWith(`${item.href}/`);
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  "flex items-center gap-2 rounded-md px-3 py-2 text-sm font-semibold transition-colors",
                  isActive
                    ? "border-l-[3px] border-l-primary bg-muted text-foreground"
                    : "text-muted-foreground hover:bg-muted hover:text-foreground"
                )}
              >
                <Icon className="size-4" />
                {t(item.labelKey)}
              </Link>
            );
          })}
        </nav>
      </aside>
      <div className="flex-1">{children}</div>
    </div>
  );
}
