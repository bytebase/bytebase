import { useTranslation } from "react-i18next";
import { RouterLink } from "@/react/components/RouterLink";
import { Alert, type AlertProps } from "@/react/components/ui/alert";
import { type ButtonProps, buttonVariants } from "@/react/components/ui/button";
import { useServerState } from "@/react/hooks/useAppState";
import {
  EXTERNAL_URL_PRODUCT_INTRO,
  PRODUCT_INTRO_QUERY_KEY,
} from "@/react/lib/productIntro";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/react/router/handles";
import { hasWorkspacePermissionV2 } from "@/utils";

type ExternalUrlAlertProps = Omit<AlertProps, "title" | "description"> & {
  actionAppearance?: ButtonProps["appearance"];
  actionClassName?: string;
};

export function ExternalUrlAlert({
  variant = "error",
  actionAppearance,
  actionClassName,
  children,
  ...props
}: ExternalUrlAlertProps) {
  const { t } = useTranslation();
  const { needConfigureExternalUrl } = useServerState();
  const canConfigure = hasWorkspacePermissionV2(
    "bb.settings.setWorkspaceProfile"
  );

  if (!needConfigureExternalUrl) {
    return null;
  }

  return (
    <Alert
      variant={variant}
      title={t("banner.external-url")}
      description={t("settings.general.workspace.external-url.description")}
      {...props}
    >
      {children}
      {canConfigure && (
        <div className="mt-2">
          <RouterLink
            to={{
              name: SETTING_ROUTE_WORKSPACE_GENERAL,
              query: {
                [PRODUCT_INTRO_QUERY_KEY]: EXTERNAL_URL_PRODUCT_INTRO,
              },
            }}
            className={buttonVariants({
              appearance: actionAppearance,
              size: "sm",
              className: actionClassName ?? "w-fit",
            })}
          >
            {t("common.configure-now")}
          </RouterLink>
        </div>
      )}
    </Alert>
  );
}
