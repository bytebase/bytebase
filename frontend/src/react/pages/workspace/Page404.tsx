import { ChevronLeft } from "lucide-react";
import { useTranslation } from "react-i18next";
import logoIcon from "@/assets/logo-icon.svg";
import { Button } from "@/react/components/ui/button";
import { router } from "@/router";
import { WORKSPACE_ROOT_MODULE } from "@/router/dashboard/workspaceRoutes";

export function Page404() {
  const { t } = useTranslation();

  const goHome = () => {
    router.push({ name: WORKSPACE_ROOT_MODULE });
  };

  return (
    <div className="w-full px-4 grid place-items-center py-24">
      <img className="w-16 h-auto opacity-80" src={logoIcon} alt="Bytebase" />
      <p className="mt-4 text-balance text-center">
        {t("common.resource-not-found")}
      </p>
      <div className="mt-12">
        <Button variant="outline" size="sm" onClick={goHome}>
          <ChevronLeft className="h-4 w-4" />
          {t("error-page.go-back-home")}
        </Button>
      </div>
    </div>
  );
}
