import { create as createProto } from "@bufbuild/protobuf";
import { useMemo } from "react";
import { useProjectByName } from "@/react/hooks/useProjectByName";
import { useAppStore } from "@/react/stores/app";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { State } from "@/types/proto-es/v1/common_pb";
import { WebhookSchema } from "@/types/proto-es/v1/project_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { ProjectWebhookForm } from "./ProjectWebhookForm";

export function ProjectWebhookDetailPage({
  projectId,
  webhookResourceId,
}: {
  projectId: string;
  webhookResourceId: string;
}) {
  const getProjectWebhookFromProjectById = useAppStore(
    (state) => state.getProjectWebhookFromProjectById
  );
  const projectsByName = useAppStore((s) => s.projectsByName);
  const projectName = `${projectNamePrefix}${projectId}`;
  // subscribe to re-render on project cache change
  void projectsByName;
  const project = useProjectByName(projectName);

  const webhook = useMemo(() => {
    if (!project) return undefined;
    return getProjectWebhookFromProjectById(project, webhookResourceId);
  }, [project, getProjectWebhookFromProjectById, webhookResourceId]);

  const allowEdit = useMemo(() => {
    if (!project) return false;
    if (project.state === State.DELETED) return false;
    return hasProjectPermissionV2(project, "bb.projects.update");
  }, [project]);

  const fallback = useMemo(() => createProto(WebhookSchema, {}), []);

  if (!project) return null;

  return (
    <ProjectWebhookForm
      create={false}
      allowEdit={allowEdit}
      project={project}
      webhook={webhook ?? fallback}
    />
  );
}
