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
import {
  isConnectAlreadyExists,
  useCreateProject,
  useNotify,
  useWorkspacePermission,
} from "@/react/hooks/useAppState";
import { projectNamePrefix } from "@/react/lib/resourceName";
import { useNavigate } from "@/react/router";
import { useAppStore } from "@/react/stores/app";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

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
  const navigate = useNavigate();
  const { createProject, setRecentProject } = useCreateProject();
  const notify = useNotify();
  const hasCreatePermission = useWorkspacePermission("bb.projects.create");
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
    if (!hasCreatePermission) return false;
    return true;
  }, [hasCreatePermission, isResourceIdValid, title]);

  const validate = useCallback(
    async (id: string) => {
      try {
        const existing = await useAppStore
          .getState()
          .fetchProject(`${projectNamePrefix}${id}`);
        if (!existing) {
          return [];
        }
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
    [t]
  );

  const handleCreate = useCallback(async () => {
    if (!allowCreate || isCreating) return;

    try {
      setIsCreating(true);
      const createdProject = await createProject(title, resourceId);

      notify({
        module: "bytebase",
        style: "SUCCESS",
        title: t("project.create-modal.success-prompt", {
          name: createdProject.title,
        }),
      });

      if (onCreated) {
        onCreated(createdProject);
      } else {
        setRecentProject(createdProject.name);
        void navigate.push({ path: `/${createdProject.name}` });
      }

      onClose();
    } catch (error) {
      if (
        isConnectAlreadyExists(error) ||
        (error instanceof ConnectError && error.code === Code.AlreadyExists)
      ) {
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
    createProject,
    navigate,
    notify,
    onClose,
    onCreated,
    resourceId,
    setRecentProject,
    t,
    title,
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
                suffix
                ref={resourceIdFieldRef}
                value={resourceId}
                resourceName={t("common.project")}
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
