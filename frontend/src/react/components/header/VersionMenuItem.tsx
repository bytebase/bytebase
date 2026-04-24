import { Volume2 } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { SETTING_ROUTE_WORKSPACE_SUBSCRIPTION } from "@/router/dashboard/workspaceSetting";
import { useActuatorV1Store, useSubscriptionV1Store } from "@/store";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

export function VersionMenuItem({ onCloseMenu }: { onCloseMenu: () => void }) {
  const { t } = useTranslation();
  const actuatorStore = useActuatorV1Store();
  const subscriptionStore = useSubscriptionV1Store();
  const [dialogOpen, setDialogOpen] = useState(false);

  const isDemo = useVueState(() => actuatorStore.isDemo);
  const isSaaSMode = useVueState(() => actuatorStore.isSaaSMode);
  const version = useVueState(() => actuatorStore.version);
  const gitCommitBE = useVueState(() => actuatorStore.gitCommitBE);
  const gitCommitFE = useVueState(() => actuatorStore.gitCommitFE);
  const hasNewRelease = useVueState(() => actuatorStore.hasNewRelease);
  const releaseLatest = useVueState(() => actuatorStore.releaseInfo.latest);
  const currentPlan = useVueState(() => subscriptionStore.currentPlan);
  const purchaseLicenseUrl = useVueState(
    () => subscriptionStore.purchaseLicenseUrl
  );
  const isSelfHostLicense = useVueState(
    () => subscriptionStore.isSelfHostLicense
  );

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

  const releaseLink = isSelfHostLicense
    ? "https://docs.bytebase.com/get-started/self-host-vs-cloud"
    : purchaseLicenseUrl;

  return (
    <>
      <div className="px-3 py-2">
        <div className="mb-2 flex items-center gap-x-2">
          {isDemo ? (
            <Badge variant="secondary">{t("common.demo-mode")}</Badge>
          ) : hasWorkspacePermissionV2("bb.settings.set") ? (
            <button
              type="button"
              className="cursor-pointer text-sm text-accent hover:underline"
              onClick={() => {
                void router.push({
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

        <button
          type="button"
          className="flex w-full items-center justify-between gap-x-2 rounded-sm px-0 py-1 text-left text-sm text-control hover:text-accent"
          onClick={() => {
            if (hasNewRelease) {
              onCloseMenu();
              setDialogOpen(true);
            }
          }}
        >
          <span className="flex items-center gap-x-2">
            {hasNewRelease ? (
              <Volume2 className="h-4 w-4 text-success" />
            ) : null}
            {formattedVersion}
          </span>
        </button>

        {!isSaaSMode ? (
          <div className="mt-1 text-xs text-control-light">
            <div>BE Git hash: {gitCommitBE.slice(0, 7)}</div>
            <div>FE Git hash: {gitCommitFE.slice(0, 7)}</div>
          </div>
        ) : null}
      </div>

      <Dialog
        open={dialogOpen}
        onOpenChange={(next) => {
          setDialogOpen(next);
        }}
      >
        <DialogContent className="max-w-lg p-6">
          <DialogTitle>
            {t("remind.release.new-version-available-with-tag", {
              tag: releaseLatest?.tag_name ?? "",
            })}
          </DialogTitle>
          <DialogDescription className="mt-2">
            {releaseLatest?.html_url ?? releaseLink}
          </DialogDescription>
          <div className="mt-6 flex justify-end gap-x-2">
            <Button variant="ghost" onClick={() => setDialogOpen(false)}>
              {t("common.dismiss")}
            </Button>
            <Button
              onClick={() => {
                window.open(releaseLatest?.html_url ?? releaseLink, "_blank");
                setDialogOpen(false);
              }}
            >
              {t("common.learn-more")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
