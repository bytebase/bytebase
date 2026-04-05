import { create as createProto } from "@bufbuild/protobuf";
import { useMemo } from "react";
import { useVueState } from "@/react/hooks/useVueState";
import { useProjectV1Store, useProjectWebhookV1Store } from "@/store";
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
  const projectStore = useProjectV1Store();
  const projectWebhookV1Store = useProjectWebhookV1Store();
  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  const webhook = useVueState(() => {
    if (!project) return undefined;
    return projectWebhookV1Store.getProjectWebhookFromProjectById(
      project,
      webhookResourceId
    );
  });

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
