import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { PROJECT_V1_ROUTE_DATABASES } from "@/app/router/handles";
import { ResourceIdField } from "@/components/ResourceIdField";
import { UserAvatar } from "@/components/UserAvatar";
import { Button } from "@/components/ui/button";
import { FormField } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  useCreateProject,
  useCurrentUser,
  useWorkspace,
  useWorkspacePermission,
} from "@/hooks/useAppState";
import { CONNECT_DATABASE_PRODUCT_INTRO } from "@/lib/productIntro";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import type { ValidatedMessage } from "@/types";
import { UpdateUserRequestSchema } from "@/types/proto-es/v1/user_service_pb";
import { WorkspaceSchema } from "@/types/proto-es/v1/workspace_service_pb";
import { extractProjectResourceName } from "@/utils";

export function ProfileSetupPage() {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const updateUser = useAppStore((state) => state.updateUser);
  const updateWorkspace = useAppStore((state) => state.updateWorkspace);
  const { createProject, setRecentProject } = useCreateProject();
  const workspace = useWorkspace();
  const workspacePolicy = useAppStore((state) => state.workspacePolicy);
  const canUpdateWorkspace = useWorkspacePermission("bb.workspaces.update");
  const canCreateProject = useWorkspacePermission("bb.projects.create");

  // Show workspace name field only if the user is the sole member of the
  // workspace (i.e. they just created it), not when they were invited.
  const canRenameWorkspace =
    canUpdateWorkspace &&
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
  const [projectTitle, setProjectTitle] = useState(() =>
    t("settings.profile.default-project-name")
  );
  const [projectResourceId, setProjectResourceId] = useState("");
  const [isProjectResourceIdValid, setIsProjectResourceIdValid] =
    useState(false);
  const [saving, setSaving] = useState(false);
  const projectTitleTrimmed = projectTitle.trim();
  const shouldCreateProject = canCreateProject && !!projectTitleTrimmed;

  const redirectUrl = () => {
    const q = new URLSearchParams(window.location.search);
    return q.get("redirect") || "/";
  };

  const validateProjectResourceId = useCallback(
    async (id: string): Promise<ValidatedMessage[]> => {
      try {
        const existing = await useAppStore
          .getState()
          .fetchProject(`${projectNamePrefix}${id}`, true);
        if (!existing) return [];
        return [
          {
            type: "error",
            message: t("resource-id.validation.duplicated", {
              resource: t("common.project"),
            }),
          },
        ];
      } catch {
        return [];
      }
    },
    [t]
  );

  const canSave =
    !!name.trim() &&
    !saving &&
    (!shouldCreateProject || isProjectResourceIdValid);

  const handleSave = async () => {
    if (!currentUser?.name || !canSave) return;
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
      let createdProjectName = "";
      if (shouldCreateProject) {
        const createdProject = await createProject(
          projectTitleTrimmed,
          projectResourceId
        );
        setRecentProject(createdProject.name);
        createdProjectName = createdProject.name;
      }
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("settings.profile.setup-success"),
      });
      if (createdProjectName) {
        router.replace({
          name: PROJECT_V1_ROUTE_DATABASES,
          params: {
            projectId: extractProjectResourceName(createdProjectName),
          },
          query: { intro: CONNECT_DATABASE_PRODUCT_INTRO },
        });
      } else {
        router.replace(redirectUrl());
      }
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
      <div className="w-full max-w-lg mx-4 flex flex-col items-center gap-y-6">
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
          <FormField title={t("settings.profile.display-name")}>
            <Input
              data-testid="profile-display-name"
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
          </FormField>

          {canRenameWorkspace && (
            <FormField title={t("settings.profile.workspace-name")}>
              <Input
                data-testid="profile-workspace-title"
                value={workspaceTitle}
                onChange={(e) => setWorkspaceTitle(e.target.value)}
                placeholder={t("settings.profile.workspace-name-placeholder")}
              />
            </FormField>
          )}

          {canCreateProject && (
            <FormField title={t("settings.profile.setup-first-project")}>
              <Input
                data-testid="profile-project-title"
                value={projectTitle}
                onChange={(e) => setProjectTitle(e.target.value)}
                placeholder={t("quick-action.new-project")}
              />
              {projectTitleTrimmed && (
                <ResourceIdField
                  suffix
                  value={projectResourceId}
                  resourceName={t("common.project")}
                  resourceTitle={projectTitle}
                  validate={validateProjectResourceId}
                  onChange={setProjectResourceId}
                  onValidationChange={setIsProjectResourceIdValid}
                />
              )}
            </FormField>
          )}

          <div className="flex flex-col gap-y-2 mt-2">
            <Button
              onClick={() => void handleSave()}
              disabled={!canSave}
              className="w-full"
            >
              {t("common.save")}
            </Button>
            <Button
              appearance="secondary"
              onClick={handleSkip}
              className="w-full"
            >
              {t("settings.profile.setup-skip")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
