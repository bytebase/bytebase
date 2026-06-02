import { create as createProto } from "@bufbuild/protobuf";
import { useMemo } from "react";
import { useVueState } from "@/react/hooks/useVueState";
import { useAppStore } from "@/react/stores/app";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { State, WebhookType } from "@/types/proto-es/v1/common_pb";
import {
  Activity_Type,
  WebhookSchema,
} from "@/types/proto-es/v1/project_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { ProjectWebhookForm } from "./ProjectWebhookForm";

export function ProjectWebhookCreatePage({ projectId }: { projectId: string }) {
  const projectsByName = useAppStore((s) => s.projectsByName);
  const projectName = `${projectNamePrefix}${projectId}`;
  // subscribe to re-render on project cache change
  void projectsByName;
  const project = useVueState(() =>
    useAppStore.getState().getProjectByName(projectName)
  );

  const allowEdit = useMemo(() => {
    if (!project) return false;
    if (project.state === State.DELETED) return false;
    return hasProjectPermissionV2(project, "bb.projects.update");
  }, [project]);

  const defaultWebhook = useMemo(
    () =>
      createProto(WebhookSchema, {
        type: WebhookType.SLACK,
        notificationTypes: [Activity_Type.ISSUE_CREATED],
      }),
    []
  );

  if (!project) return null;

  return (
    <ProjectWebhookForm
      allowEdit={allowEdit}
      create={true}
      project={project}
      webhook={defaultWebhook}
    />
  );
}
