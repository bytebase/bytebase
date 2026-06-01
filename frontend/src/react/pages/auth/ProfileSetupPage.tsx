import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { UserAvatar } from "@/react/components/UserAvatar";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { useCurrentUser, useWorkspace } from "@/react/hooks/useAppState";
import { useAppStore } from "@/react/stores/app";
import { router } from "@/router";
import { pushNotification } from "@/store";
import { UpdateUserRequestSchema } from "@/types/proto-es/v1/user_service_pb";
import { WorkspaceSchema } from "@/types/proto-es/v1/workspace_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

export function ProfileSetupPage() {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const updateUser = useAppStore((state) => state.updateUser);
  const updateWorkspace = useAppStore((state) => state.updateWorkspace);
  const fetchWorkspaceIamPolicy = useAppStore(
    (state) => state.fetchWorkspaceIamPolicy
  );
  const workspace = useWorkspace();
  const workspacePolicy = useAppStore((state) => state.workspacePolicy);

  // The member count below gates the workspace-rename field, so make sure the
  // policy is loaded even when this page is reached outside the setup flow.
  useEffect(() => {
    void fetchWorkspaceIamPolicy();
  }, [fetchWorkspaceIamPolicy]);

  // Show workspace name field only if the user is the sole member of the
  // workspace (i.e. they just created it), not when they were invited.
  const canRenameWorkspace =
    hasWorkspacePermissionV2("bb.workspaces.update") &&
    (workspacePolicy?.bindings ?? []).reduce(
      (count, b) => count + b.members.length,
      0
    ) === 1;

  // `currentUser` loads asynchronously (unknownUser with an empty email until
  // then), so seed the name field once the real user is known rather than
  // freezing on the placeholder title. The ref keeps it a one-shot seed so
  // later edits aren't clobbered.
  const [name, setName] = useState(
    currentUser.email ? (currentUser.title ?? "") : ""
  );
  const nameSeededRef = useRef(Boolean(currentUser.email));
  useEffect(() => {
    if (!nameSeededRef.current && currentUser.email) {
      nameSeededRef.current = true;
      setName(currentUser.title ?? "");
    }
  }, [currentUser.email, currentUser.title]);
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
      await updateUser(
        create(UpdateUserRequestSchema, {
          user: { name: currentUser.name, title: name.trim() },
          updateMask: create(FieldMaskSchema, { paths: ["title"] }),
        })
      );
      if (canRenameWorkspace && workspaceTitle.trim() && workspace?.name) {
        await updateWorkspace(
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
