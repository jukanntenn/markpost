import { useNavigate } from "react-router-dom";
import { useContext } from "react";
import { UserInfoContext } from "../components/UserInfoContext";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

function NotFound() {
  const navigate = useNavigate();
  const { isAuthenticated } = useContext(UserInfoContext);
  const { t } = useTranslation();

  return (
    <div className="flex min-h-[60svh] items-center justify-center">
      <Card className="w-full max-w-md">
        <CardContent className="space-y-4 text-center">
          <h1 className="text-3xl font-semibold tracking-tight">
            {t("notFound.title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {isAuthenticated
              ? t("notFound.pageNotFound")
              : t("notFound.pageNotFoundLoginRequired")}
          </p>
          {isAuthenticated ? (
            <Button type="button" onClick={() => navigate("/dashboard")}>
              {t("notFound.backToDashboard")}
            </Button>
          ) : (
            <Button type="button" onClick={() => navigate("/login")}>
              {t("notFound.goToLogin")}
            </Button>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

export default NotFound;
