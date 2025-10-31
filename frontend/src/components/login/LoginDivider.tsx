import { useTranslation } from "react-i18next";

function LoginDivider() {
  const { t } = useTranslation();
  return (
    <div className="position-relative mb-4">
      <hr className="text-muted" />
      <span className="position-absolute top-50 start-50 translate-middle bg-body px-3 text-muted small">
        {t("login.or")}
      </span>
    </div>
  );
}

export default LoginDivider;
