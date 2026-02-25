import { useTranslation } from "react-i18next";

function LoginDivider() {
  const { t } = useTranslation();
  return (
    <div className="relative my-6">
      <div className="absolute inset-0 flex items-center">
        <span className="w-full border-t" />
      </div>
      <div className="relative flex justify-center text-xs uppercase">
        <span className="bg-background px-2 text-muted-foreground">
          {t("login.or")}
        </span>
      </div>
    </div>
  );
}

export default LoginDivider;
