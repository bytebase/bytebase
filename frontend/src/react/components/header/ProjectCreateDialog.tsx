import { Code, ConnectError } from "@connectrpc/connect";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  ResourceIdField,
  type ResourceIdFieldRef,
} from "@/react/components/ResourceIdField";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { router } from "@/router";
import { pushNotification, useProjectV1Store, useUIStateStore } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { unknownProject } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

export interface ProjectCreateDialogProps {
  open: boolean;
  onClose: () => void;
  onCreated?: (project: Project) => void;
}

export function ProjectCreateDialog({
  open,
  onClose,
  onCreated,
}: ProjectCreateDialogProps) {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();
  const uiStateStore = useUIStateStore();
  const defaultProjectTitle = t("quick-action.new-project");
  const [title, setTitle] = useState(defaultProjectTitle);
  const [resourceId, setResourceId] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const [isResourceIdValid, setIsResourceIdValid] = useState(false);
  const resourceIdFieldRef = useRef<ResourceIdFieldRef>(null);

  useEffect(() => {
    if (!open) return;
    setTitle(defaultProjectTitle);
    setResourceId("");
    setIsCreating(false);
    setIsResourceIdValid(false);
  }, [defaultProjectTitle, open]);

  const allowCreate = useMemo(() => {
    if (!title.trim()) return false;
    if (!isResourceIdValid) return false;
    if (!hasWorkspacePermissionV2("bb.projects.create")) return false;
    return true;
  }, [isResourceIdValid, title]);

  const validate = useCallback(
    async (id: string) => {
      try {
        await projectStore.getOrFetchProjectByName(
          `${projectNamePrefix}${id}`,
          true
        );
        return [
          {
            type: "error" as const,
            message: t("resource-id.validation.duplicated", {
              resource: t("common.project"),
            }),
          },
        ];
      } catch {
        return [];
      }
    },
    [projectStore, t]
  );

  const handleCreate = useCallback(async () => {
    if (!allowCreate || isCreating) return;

    try {
      setIsCreating(true);
      const createdProject = await projectStore.createProject(
        { ...unknownProject(), title },
        resourceId
      );

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("project.create-modal.success-prompt", {
          name: createdProject.title,
        }),
      });

      if (onCreated) {
        onCreated(createdProject);
      } else {
        void uiStateStore.saveIntroStateByKey({
          key: "project.visit",
          newState: true,
        });
        void router.push({ path: `/${createdProject.name}` });
      }

      onClose();
    } catch (error) {
      if (error instanceof ConnectError && error.code === Code.AlreadyExists) {
        resourceIdFieldRef.current?.addValidationError(error.message);
      } else {
        throw error;
      }
    } finally {
      setIsCreating(false);
    }
  }, [
    allowCreate,
    isCreating,
    onClose,
    onCreated,
    projectStore,
    resourceId,
    t,
    title,
    uiStateStore,
  ]);

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="standard">
        <SheetHeader>
          <SheetTitle>{t("quick-action.create-project")}</SheetTitle>
        </SheetHeader>

        <SheetBody>
          <div className="flex flex-col gap-y-6">
            <div>
              <label className="text-base leading-6 font-medium text-control">
                {t("project.create-modal.project-name")}
                <span className="ml-0.5 text-error">*</span>
              </label>
              <Input
                className="mt-2 mb-1"
                value={title}
                maxLength={200}
                placeholder={t("project.create-modal.project-name")}
                onChange={(e) => setTitle(e.target.value)}
              />
              <ResourceIdField
                ref={resourceIdFieldRef}
                value={resourceId}
                resourceType="project"
                resourceName={title}
                resourceTitle={title}
                validate={validate}
                onChange={setResourceId}
                onValidationChange={setIsResourceIdValid}
              />
            </div>
          </div>

          {isCreating && (
            <div className="absolute inset-0 flex items-center justify-center bg-background/50">
              <div className="size-6 animate-spin rounded-full border-2 border-accent border-t-transparent" />
            </div>
          )}
        </SheetBody>

        <SheetFooter>
          <Button variant="ghost" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button disabled={!allowCreate} onClick={handleCreate}>
            {t("common.create")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
