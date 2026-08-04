import { useCallback, useEffect } from "react";
import { router } from "@/app/router";
import {
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_INSTANCE_DETAIL,
  PROJECT_V1_ROUTE_INSTANCES,
} from "@/app/router/handles";
import { CreateInstanceView } from "@/components/instance/CreateInstanceView";
import { useAppStore } from "@/stores/app";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import { extractInstanceResourceName } from "@/utils";

export function ProjectCreateInstancePage({
  projectId,
}: {
  projectId: string;
}) {
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

  const onDismiss = useCallback(() => {
    router.push({
      name: PROJECT_V1_ROUTE_INSTANCES,
      params: { projectId },
    });
  }, [projectId]);

  const onCreated = useCallback(
    (instance: Instance) => {
      const instanceId = extractInstanceResourceName(instance.name);
      router.push({
        name: PROJECT_V1_ROUTE_INSTANCE_DETAIL,
        params: { projectId, instanceId },
        query: { syncingInstance: instanceId },
        hash: "databases",
      });
    },
    [projectId]
  );

  if (isDefault || !project) return null;

  return (
    <CreateInstanceView
      parent={parent}
      project={project}
      onDismiss={onDismiss}
      onCreated={onCreated}
    />
  );
}
