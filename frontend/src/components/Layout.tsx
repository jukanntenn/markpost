import { useContext } from "react";
import { UserInfoContext } from "./UserInfoContext";
import { Outlet, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import ThemeToggle from "./ThemeToggle";
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

const Layout = () => {
  const { logout, userInfo, isAuthenticated, isAdmin } = useContext(UserInfoContext);
  const navigate = useNavigate();
  const { t } = useTranslation();

  const handleLogout = () => {
    logout();
    navigate("/login", { replace: true });
  };

  return (
    <>
      <header className="sticky top-0 z-50 w-full border-b bg-background/80 backdrop-blur">
        <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-4">
          <Button
            type="button"
            variant="ghost"
            className="h-9 px-2"
            onClick={() => navigate("/dashboard")}
          >
            <img src="markpost.svg" alt="Markpost" className="h-6 w-auto" />
          </Button>
          <div className="flex items-center gap-2">
            <ThemeToggle />
            {isAuthenticated && (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button type="button" variant="ghost" className="gap-2">
                    <UserIcon className="size-4" />
                    <span className="hidden sm:inline">
                      {userInfo?.user?.username || t("common.user")}
                    </span>
                    <ChevronDownIcon className="size-4 text-muted-foreground" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuLabel>
                    {userInfo?.user?.username || t("common.user")}
                  </DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  {isAdmin() && (
                    <DropdownMenuItem onClick={() => navigate("/admin")}>
                      <ShieldIcon className="size-4" />
                      {t("navigation.userMenu.admin")}
                    </DropdownMenuItem>
                  )}
                  <DropdownMenuItem onClick={() => navigate("/settings")}>
                    <SettingsIcon className="size-4" />
                    {t("navigation.userMenu.settings")}
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem variant="destructive" onClick={handleLogout}>
                    <LogOutIcon className="size-4" />
                    {t("navigation.userMenu.logout")}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            )}
          </div>
        </div>
      </header>
      <main className="mx-auto w-full max-w-7xl px-4 py-6">
        <Outlet />
      </main>
    </>
  );
};

export default Layout;
