import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { Copy, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Alert, AlertDescription } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import { useVueState } from "@/react/hooks/useVueState";
import {
  pushNotification,
  useActuatorV1Store,
  useSettingV1Store,
} from "@/store";
import { hasWorkspacePermissionV2 } from "@/utils";

// ============================================================
// AADSyncDrawer
// ============================================================

export function AADSyncDrawer({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const actuatorStore = useActuatorV1Store();
  const settingV1Store = useSettingV1Store();

  const externalUrl = useVueState(
    () => actuatorStore.serverInfo?.externalUrl ?? ""
  );
  const workspaceResourceName = useVueState(
    () => actuatorStore.workspaceResourceName
  );
  const directorySyncToken = useVueState(
    () => settingV1Store.workspaceProfile.directorySyncToken
  );

  useEscapeKey(true, onClose);

  const scimUrl =
    externalUrl && workspaceResourceName
      ? `${externalUrl}/hook/scim/${workspaceResourceName}`
      : "";

  const copyToClipboard = async (value: string) => {
    try {
      if (navigator.clipboard) {
        await navigator.clipboard.writeText(value);
      } else {
        // Fallback for non-secure contexts (e.g. self-hosted HTTP)
        const textarea = document.createElement("textarea");
        textarea.value = value;
        textarea.style.position = "fixed";
        textarea.style.opacity = "0";
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand("copy");
        document.body.removeChild(textarea);
      }
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.copied"),
      });
    } catch {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: t("common.copy-failed"),
      });
    }
  };

  const handleResetToken = async () => {
    const confirmed = window.confirm(
      t("settings.members.entra-sync.reset-token-warning")
    );
    if (!confirmed) return;

    try {
      await settingV1Store.updateWorkspaceProfile({
        payload: { directorySyncToken: "" },
        updateMask: create(FieldMaskSchema, {
          paths: ["value.workspace_profile.directory_sync_token"],
        }),
      });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch {
      // error already shown by store
    }
  };

  return (
    <>
      {/* Backdrop */}
      <div className="fixed inset-0 z-40 bg-black/30" onClick={onClose} />

      {/* Drawer */}
      <div
        role="dialog"
        aria-modal="true"
        className="fixed inset-y-0 right-0 z-50 w-[40rem] max-w-[100vw] bg-white shadow-xl flex flex-col"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b">
          <h2 className="text-lg font-medium">
            {t("settings.members.entra-sync.self")}
          </h2>
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="h-5 w-5" />
          </Button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto px-6 py-6">
          <div className="flex flex-col gap-y-6">
            {/* Description */}
            <p className="text-sm text-control-light">
              {t("settings.members.entra-sync.description")}{" "}
              <a
                href="https://docs.bytebase.com/administration/scim/overview?source=console"
                target="_blank"
                rel="noopener noreferrer"
                className="text-accent hover:underline"
              >
                {t("common.learn-more")}
              </a>
            </p>

            {/* Missing external URL warning */}
            {!externalUrl && (
              <Alert variant="warning">
                <AlertDescription>{t("banner.external-url")}</AlertDescription>
              </Alert>
            )}

            {/* SCIM Endpoint URL */}
            <div className="flex flex-col gap-y-2">
              <label className="block text-sm font-medium text-control">
                {t("settings.members.entra-sync.endpoint")}
              </label>
              <span className="textinfolabel text-sm">
                {t("settings.members.entra-sync.endpoint-tip")}
              </span>
              <div className="flex items-center gap-x-2">
                <Input readOnly value={scimUrl} className="flex-1 text-sm" />
                <Button
                  variant="outline"
                  size="sm"
                  disabled={!scimUrl}
                  onClick={() => copyToClipboard(scimUrl)}
                >
                  <Copy className="h-4 w-4" />
                </Button>
              </div>
            </div>

            {/* Secret Token */}
            <div className="flex flex-col gap-y-2">
              <label className="block text-sm font-medium text-control">
                {t("settings.members.entra-sync.secret-token")}
              </label>
              <span className="textinfolabel text-sm">
                {t("settings.members.entra-sync.secret-token-tip")}
              </span>
              <div className="flex items-center gap-x-2">
                <Input
                  readOnly
                  type="password"
                  value={directorySyncToken}
                  className="flex-1 text-sm"
                />
                <Button
                  variant="outline"
                  size="sm"
                  disabled={!directorySyncToken}
                  onClick={() => copyToClipboard(directorySyncToken)}
                >
                  <Copy className="h-4 w-4" />
                </Button>
              </div>
              {hasWorkspacePermissionV2("bb.settings.setWorkspaceProfile") && (
                <Button
                  variant="outline"
                  size="sm"
                  className="self-start text-error border-error hover:bg-error/10"
                  onClick={handleResetToken}
                >
                  {t("settings.members.entra-sync.reset-token")}
                </Button>
              )}
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-x-2 px-6 py-4 border-t">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
        </div>
      </div>
    </>
  );
}
