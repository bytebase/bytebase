import { create } from "@bufbuild/protobuf";
import { Copy } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { workspaceServiceClientConnect } from "@/api";
import { ExternalUrlAlert } from "@/components/ExternalUrlAlert";
import { LearnMoreLink } from "@/components/LearnMoreLink";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { writeTextToClipboard } from "@/lib/clipboard";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { RotateDirectorySyncTokenRequestSchema } from "@/types/proto-es/v1/workspace_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

// ============================================================
// AADSyncSheet
// ============================================================

export function AADSyncSheet({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const { t } = useTranslation();

  const externalUrl = useAppStore((s) => s.serverInfo?.externalUrl ?? "");
  const workspaceResourceName = useAppStore((s) => s.workspaceResourceName());
  const tokenConfigured = useAppStore(
    (s) => s.getWorkspaceProfile().directorySyncTokenConfigured
  );

  const scimUrl =
    externalUrl && workspaceResourceName
      ? `${externalUrl}/hook/scim/${workspaceResourceName}`
      : "";

  const copyToClipboard = async (value: string) => {
    if (await writeTextToClipboard(value)) {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.copied"),
      });
    } else {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: t("common.copy-failed"),
      });
    }
  };

  // The plaintext token is returned exactly once by the rotate RPC, so it lives
  // in local state only. The sheet stays mounted when closed, so this must be
  // cleared explicitly — otherwise reopening would redisplay a token that is
  // supposed to have been shown once.
  const [mintedToken, setMintedToken] = useState("");
  // Each rotation invalidates the previous token, so two in flight can resolve
  // out of order and leave the admin copying a token the server has already
  // replaced. One at a time.
  const [rotating, setRotating] = useState(false);

  const handleClose = () => {
    setMintedToken("");
    onClose();
  };

  const handleRotateToken = async () => {
    if (rotating) return;

    if (tokenConfigured) {
      const confirmed = window.confirm(
        t("settings.members.entra-sync.regenerate-token-warning")
      );
      if (!confirmed) return;
    }

    setRotating(true);
    try {
      const resp = await workspaceServiceClientConnect.rotateDirectorySyncToken(
        create(RotateDirectorySyncTokenRequestSchema, {
          name: workspaceResourceName,
        })
      );
      setMintedToken(resp.token);
      await useAppStore.getState().loadWorkspaceProfile(true);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch {
      // error already shown by the client interceptor
    } finally {
      setRotating(false);
    }
  };

  return (
    <Sheet open={open} onOpenChange={(next) => !next && handleClose()}>
      <SheetContent width="standard">
        <SheetHeader>
          <SheetTitle>{t("settings.members.entra-sync.self")}</SheetTitle>
        </SheetHeader>

        <SheetBody>
          <div className="flex flex-col gap-y-6">
            {/* Description */}
            <p className="text-sm text-control-light">
              {t("settings.members.entra-sync.description")}{" "}
              <LearnMoreLink
                href="https://docs.bytebase.com/administration/scim/overview?source=console"
                className="text-accent"
              />
            </p>

            {/* Missing external URL warning */}
            <ExternalUrlAlert variant="warning" actionAppearance="outline" />

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
                  appearance="outline"
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
              {mintedToken ? (
                <>
                  <div className="flex items-center gap-x-2">
                    <Input
                      readOnly
                      value={mintedToken}
                      className="flex-1 text-sm font-mono"
                    />
                    <Button
                      appearance="outline"
                      size="sm"
                      onClick={() => copyToClipboard(mintedToken)}
                    >
                      <Copy className="size-4" />
                    </Button>
                  </div>
                  <Alert
                    variant="warning"
                    description={t(
                      "settings.members.entra-sync.token-shown-once"
                    )}
                  />
                </>
              ) : (
                <span className="textinfolabel text-sm">
                  {tokenConfigured
                    ? t("settings.members.entra-sync.token-configured")
                    : t("settings.members.entra-sync.token-not-configured")}
                </span>
              )}
              {hasWorkspacePermissionV2(
                "bb.workspaces.rotateDirectorySyncToken"
              ) && (
                <Button
                  appearance="outline"
                  size="sm"
                  className="self-start"
                  disabled={rotating}
                  onClick={handleRotateToken}
                >
                  {tokenConfigured
                    ? t("settings.members.entra-sync.regenerate-token")
                    : t("settings.members.entra-sync.generate-token")}
                </Button>
              )}
            </div>
          </div>
        </SheetBody>

        <SheetFooter>
          <Button appearance="outline" onClick={handleClose}>
            {t("common.cancel")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
