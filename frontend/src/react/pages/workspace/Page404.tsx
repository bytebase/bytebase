import { ChevronLeft } from "lucide-react";
import { useTranslation } from "react-i18next";
import logoIcon from "@/assets/logo-icon.svg";
import { RouterLink } from "@/react/components/RouterLink";
import { buttonVariants } from "@/react/components/ui/button";
import { WORKSPACE_ROUTE_LANDING } from "@/react/router/handles";

export function Page404() {
  const { t } = useTranslation();

  return (
    <div className="w-full px-4 grid place-items-center py-24">
      <img className="w-16 h-auto opacity-80" src={logoIcon} alt="Bytebase" />
      <p className="mt-4 text-balance text-center">
        {t("common.resource-not-found")}
      </p>
      <div className="mt-12">
        <RouterLink
          to={{ name: WORKSPACE_ROUTE_LANDING }}
          className={buttonVariants({ variant: "outline", size: "sm" })}
        >
          <ChevronLeft className="h-4 w-4" />
          {t("error-page.go-back-home")}
        </RouterLink>
      </div>
    </div>
  );
}
