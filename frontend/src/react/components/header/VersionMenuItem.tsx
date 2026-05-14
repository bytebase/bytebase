import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Badge } from "@/react/components/ui/badge";
import {
  useServerInfo,
  useSubscription,
  useWorkspacePermission,
} from "@/react/hooks/useAppState";
import {
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
  useNavigate,
} from "@/react/router";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";

export function VersionMenuItem({ onCloseMenu }: { onCloseMenu: () => void }) {
  const { t } = useTranslation();
  const serverInfo = useServerInfo();
  const { subscription } = useSubscription();
  const canManageSettings = useWorkspacePermission("bb.settings.set");
  const navigate = useNavigate();

  const version = serverInfo?.version ?? "";
  const gitCommitBE = serverInfo?.gitCommit || "unknown";
  const gitCommitFE = import.meta.env.GIT_COMMIT || "unknown";
  const currentPlan = subscription?.plan ?? PlanType.FREE;

  const planLabel = useMemo(() => {
    switch (currentPlan) {
      case PlanType.TEAM:
        return t("subscription.plan.team.title");
      case PlanType.ENTERPRISE:
        return t("subscription.plan.enterprise.title");
      default:
        return t("subscription.plan.free.title");
    }
  }, [currentPlan, t]);

  const formattedVersion = useMemo(() => {
    if (version && version.split(".").length === 3) {
      return `v${version}`;
    }
    return version || "unknown";
  }, [version]);

  return (
    <>
      <div className="px-3 py-2">
        <div className="mb-2 flex items-center gap-x-2">
          {serverInfo?.demo ? (
            <Badge variant="secondary">{t("common.demo-mode")}</Badge>
          ) : canManageSettings ? (
            <button
              type="button"
              className="cursor-pointer text-sm text-accent hover:underline"
              onClick={() => {
                void navigate.push({
                  name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
                });
                onCloseMenu();
              }}
            >
              {planLabel}
            </button>
          ) : (
            <span className="text-sm text-control-light">{planLabel}</span>
          )}
        </div>

        {!serverInfo?.saas ? (
          <>
            <div className="flex w-full items-center justify-between gap-x-2 rounded-sm px-0 py-1 text-left text-sm text-control">
              <span className="flex items-center gap-x-2">
                {formattedVersion}
              </span>
            </div>

            <div className="mt-1 text-xs text-control-light">
              <div>BE Git hash: {gitCommitBE.slice(0, 7)}</div>
              <div>FE Git hash: {gitCommitFE.slice(0, 7)}</div>
            </div>
          </>
        ) : null}
      </div>
    </>
  );
}
