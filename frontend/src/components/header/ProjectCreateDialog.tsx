import { Code, ConnectError } from "@connectrpc/connect";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "@/app/router";
import { PROJECT_V1_ROUTE_DATABASES } from "@/app/router/handles";
import {
  ResourceIdField,
  type ResourceIdFieldRef,
} from "@/components/ResourceIdField";
import { Button } from "@/components/ui/button";
import { FormField, FormFieldGroup, FormTitle } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import {
  isConnectAlreadyExists,
  useCreateProject,
  useNotify,
  useWorkspacePermission,
} from "@/hooks/useAppState";
import {
  CONNECT_DATABASE_PRODUCT_INTRO,
  PRODUCT_INTRO_QUERY_KEY,
} from "@/lib/productIntro";
import { projectNamePrefix } from "@/lib/resourceName";
import { useAppStore } from "@/stores/app";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { extractProjectResourceName } from "@/utils";

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
        void navigate.push({
          name: PROJECT_V1_ROUTE_DATABASES,
          params: {
            projectId: extractProjectResourceName(createdProject.name),
          },
          query: { [PRODUCT_INTRO_QUERY_KEY]: CONNECT_DATABASE_PRODUCT_INTRO },
        });
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
          <FormFieldGroup>
            <FormField>
              <FormTitle id="create-project-title-title">
                {t("project.create-modal.project-name")}
                <span className="ml-0.5 text-error">*</span>
              </FormTitle>
              <Input
                id="create-project-title"
                aria-labelledby="create-project-title-title"
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
            </FormField>
          </FormFieldGroup>

          {isCreating && (
            <div className="absolute inset-0 flex items-center justify-center bg-background/50">
              <div className="size-6 animate-spin rounded-full border-2 border-accent border-t-transparent" />
            </div>
          )}
        </SheetBody>

        <SheetFooter>
          <Button appearance="secondary" onClick={onClose}>
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
