import { useCallback, useEffect } from "react";
import { router } from "@/app/router";
import {
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_INSTANCES,
} from "@/app/router/handles";
import { InstanceDetailView } from "@/components/instance/InstanceDetailView";
import { useAppStore } from "@/stores/app";

interface ProjectInstanceDetailPageProps {
  projectId: string;
  instanceId: string;
}

export function ProjectInstanceDetailPage({
  projectId,
  instanceId,
}: ProjectInstanceDetailPageProps) {
  const parent = `projects/${projectId}`;
  const project = useAppStore((state) => state.projectsByName[parent]);
  const defaultProject = useAppStore(
    (state) => state.serverInfo?.defaultProject ?? ""
  );
  const isDefault = parent === defaultProject;

  useEffect(() => {
    if (!isDefault) return;
    router.replace({
      name: PROJECT_V1_ROUTE_DATABASES,
      params: { projectId },
    });
  }, [isDefault, projectId]);

  const onLeave = useCallback(() => {
    router.replace({
      name: PROJECT_V1_ROUTE_INSTANCES,
      params: { projectId },
    });
  }, [projectId]);

  if (isDefault || !project) return null;

  return (
    <InstanceDetailView
      instanceName={`${parent}/instances/${instanceId}`}
      project={project}
      onLeave={onLeave}
    />
  );
}
