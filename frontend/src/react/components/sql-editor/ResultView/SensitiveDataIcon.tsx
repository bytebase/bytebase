import { EyeOffIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { WORKSPACE_ROUTE_GLOBAL_MASKING } from "@/router/dashboard/workspaceRoutes";
import { hasWorkspacePermissionV2 } from "@/utils/iam/permission";

export function SensitiveDataIcon() {
  const { t } = useTranslation();
  const clickable = hasWorkspacePermissionV2("bb.policies.update");

  const handleClick = () => {
    if (!clickable) return;
    const url = router.resolve({ name: WORKSPACE_ROUTE_GLOBAL_MASKING });
    window.open(url.href, "_BLANK");
  };

  return (
    <Tooltip content={t("sensitive-data.self")} delayDuration={250}>
      <EyeOffIcon
        className={cn("size-3 -mb-0.5", clickable && "cursor-pointer")}
        onClick={handleClick}
      />
    </Tooltip>
  );
}
