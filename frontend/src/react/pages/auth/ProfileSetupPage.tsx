import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { UserAvatar } from "@/react/components/UserAvatar";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import {
  pushNotification,
  useCurrentUserV1,
  useUserStore,
  useWorkspaceV1Store,
} from "@/store";
import { UpdateUserRequestSchema } from "@/types/proto-es/v1/user_service_pb";
import { WorkspaceSchema } from "@/types/proto-es/v1/workspace_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

export function ProfileSetupPage() {
  const { t } = useTranslation();
  const currentUser = useVueState(() => useCurrentUserV1().value);
  const userStore = useUserStore();
  const workspaceStore = useWorkspaceV1Store();
  const workspace = useVueState(() => workspaceStore.currentWorkspace);
  const workspacePolicy = useVueState(() => workspaceStore.workspaceIamPolicy);

  // Show workspace name field only if the user is the sole member of the
  // workspace (i.e. they just created it), not when they were invited.
  const canRenameWorkspace =
    hasWorkspacePermissionV2("bb.workspaces.update") &&
    workspacePolicy.bindings.reduce(
      (count, b) => count + b.members.length,
      0
    ) === 1;

  const [name, setName] = useState(currentUser?.title ?? "");
  const [workspaceTitle, setWorkspaceTitle] = useState(workspace?.title ?? "");
  const [saving, setSaving] = useState(false);

  const redirectUrl = () => {
    const q = new URLSearchParams(window.location.search);
    return q.get("redirect") || "/";
  };

  const handleSave = async () => {
    if (!currentUser?.name || !name.trim()) return;
    setSaving(true);
    try {
      await userStore.updateUser(
        create(UpdateUserRequestSchema, {
          user: { name: currentUser.name, title: name.trim() },
          updateMask: create(FieldMaskSchema, { paths: ["title"] }),
        })
      );
      if (canRenameWorkspace && workspaceTitle.trim() && workspace?.name) {
        await workspaceStore.updateWorkspace(
          create(WorkspaceSchema, {
            name: workspace.name,
            title: workspaceTitle.trim(),
          }),
          ["title"]
        );
      }
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("settings.profile.setup-success"),
      });
      router.replace(redirectUrl());
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("settings.profile.setup-failed"),
      });
    } finally {
      setSaving(false);
    }
  };

  const handleSkip = () => {
    router.replace(redirectUrl());
  };

  const displayName = name.trim() || currentUser?.email || "?";

  return (
    <div className="flex flex-col items-center justify-center min-h-screen px-4">
      <div className="w-full max-w-sm flex flex-col items-center gap-y-6">
        <UserAvatar
          title={displayName}
          colorSeed={currentUser?.email}
          size="md"
          className="!size-16 !text-2xl"
        />

        <div className="text-center">
          <h1 className="text-xl font-semibold">
            {t("settings.profile.setup-title")}
          </h1>
        </div>

        <div className="w-full flex flex-col gap-y-4">
          <div className="flex flex-col gap-y-1.5">
            <label className="text-sm font-medium">
              {t("settings.profile.display-name")}
            </label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t("settings.profile.display-name-placeholder")}
              autoFocus
              onKeyDown={(e) => {
                if (e.key === "Enter" && name.trim()) {
                  void handleSave();
                }
              }}
            />
          </div>

          {canRenameWorkspace && (
            <div className="flex flex-col gap-y-1.5">
              <label className="text-sm font-medium">
                {t("settings.profile.workspace-name")}
              </label>
              <Input
                value={workspaceTitle}
                onChange={(e) => setWorkspaceTitle(e.target.value)}
                placeholder={t("settings.profile.workspace-name-placeholder")}
              />
            </div>
          )}

          <div className="flex flex-col gap-y-2 mt-2">
            <Button
              onClick={() => void handleSave()}
              disabled={!name.trim() || saving}
              className="w-full"
            >
              {t("common.save")}
            </Button>
            <Button variant="ghost" onClick={handleSkip} className="w-full">
              {t("settings.profile.setup-skip")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
