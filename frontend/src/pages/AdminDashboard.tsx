import { useMemo } from "react";
import { useLocation, useNavigate, Outlet } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";

type NavItem = {
  to: string;
  label: string;
};

export default function AdminDashboard() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();

  const items = useMemo<NavItem[]>(
    () => [
      { to: "/admin/users", label: t("admin.nav.users") },
      { to: "/admin/posts", label: t("admin.nav.posts") },
      { to: "/admin/channels", label: t("admin.nav.channels") },
    ],
    [t]
  );

  return (
    <div className="mx-auto max-w-6xl">
      <Card>
        <CardHeader className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <CardTitle>{t("admin.nav.title")}</CardTitle>
          <div className="flex flex-wrap items-center gap-2">
            {items.map((item) => {
              const active = location.pathname.startsWith(`/ui${item.to}`);
              return (
                <Button
                  key={item.to}
                  type="button"
                  variant={active ? "default" : "outline"}
                  className={cn(active ? "" : "bg-transparent")}
                  onClick={() => navigate(item.to)}
                >
                  {item.label}
                </Button>
              );
            })}
          </div>
        </CardHeader>
        <CardContent>
          <Outlet />
        </CardContent>
      </Card>
    </div>
  );
}

