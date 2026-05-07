import { Volume2 } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import semver from "semver";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/react/components/ui/dialog";
import {
  useServerInfo,
  useSubscription,
  useWorkspacePermission,
} from "@/react/hooks/useAppState";
import {
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
  useNavigate,
} from "@/react/router";
import type { ReleaseInfo } from "@/types/actuator";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { STORAGE_KEY_RELEASE } from "@/utils/storage-keys";

function readReleaseInfo(): ReleaseInfo {
  try {
    const raw = localStorage.getItem(STORAGE_KEY_RELEASE);
    return raw
      ? (JSON.parse(raw) as ReleaseInfo)
      : {
          ignoreRemindModalTillNextRelease: false,
          nextCheckTs: 0,
        };
  } catch {
    return {
      ignoreRemindModalTillNextRelease: false,
      nextCheckTs: 0,
    };
  }
}

function hasNewerRelease(latest: string | undefined, current: string) {
  if (!latest) return false;
  if (current === "development") return true;
  const latestVersion = semver.coerce(latest);
  const currentVersion = semver.coerce(current);
  if (!latestVersion || !currentVersion) return false;
  return semver.gt(latestVersion, currentVersion);
}

export function VersionMenuItem({ onCloseMenu }: { onCloseMenu: () => void }) {
  const { t } = useTranslation();
  const serverInfo = useServerInfo();
  const { subscription } = useSubscription();
  const canManageSettings = useWorkspacePermission("bb.settings.set");
  const navigate = useNavigate();
  const [dialogOpen, setDialogOpen] = useState(false);
  const releaseInfo = useMemo(readReleaseInfo, []);

  const version = serverInfo?.version ?? "";
  const gitCommitBE = serverInfo?.gitCommit || "unknown";
  const gitCommitFE = import.meta.env.GIT_COMMIT || "unknown";
  const currentPlan = subscription?.plan ?? PlanType.FREE;
  const hasNewRelease = hasNewerRelease(releaseInfo.latest?.tag_name, version);

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

  const isSelfHostLicense =
    import.meta.env.MODE.toLowerCase() !== "release-aws";
  const purchaseLicenseUrl = import.meta.env.BB_PURCHASE_LICENSE_URL as string;
  const releaseLink = isSelfHostLicense
    ? "https://docs.bytebase.com/get-started/self-host-vs-cloud"
    : purchaseLicenseUrl;

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

            <div className="mt-1 text-xs text-control-light">
              <div>BE Git hash: {gitCommitBE.slice(0, 7)}</div>
              <div>FE Git hash: {gitCommitFE.slice(0, 7)}</div>
            </div>
          </>
        ) : null}
      </div>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-lg p-6">
          <DialogTitle>
            {t("remind.release.new-version-available-with-tag", {
              tag: releaseInfo.latest?.tag_name ?? "",
            })}
          </DialogTitle>
          <DialogDescription className="mt-2">
            {releaseInfo.latest?.html_url ?? releaseLink}
          </DialogDescription>
          <div className="mt-6 flex justify-end gap-x-2">
            <Button variant="ghost" onClick={() => setDialogOpen(false)}>
              {t("common.dismiss")}
            </Button>
            <Button
              onClick={() => {
                window.open(
                  releaseInfo.latest?.html_url ?? releaseLink,
                  "_blank"
                );
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
