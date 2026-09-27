import { useCallback, useEffect } from "react";
import { router } from "@/app/router";
import {
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_INSTANCE_CREATE,
} from "@/app/router/handles";
import { markListScrollRestorationEntry } from "@/app/router/NavigationScrollRestoration";
import { InstanceDashboard } from "@/components/instance/InstanceDashboard";
import { ProjectPageLayout } from "@/components/ProjectPageLayout";
import { useAppStore } from "@/stores/app";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";

export function ProjectInstancesPage({ projectId }: { projectId: string }) {
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

  const handleOpen = useCallback(
    (instance: Instance, event: React.MouseEvent) => {
      const url = `/${instance.name}`;
      if (event.ctrlKey || event.metaKey) {
        window.open(url, "_blank");
        return;
      }
      markListScrollRestorationEntry();
      router.push(url);
    },
    []
  );

  if (isDefault || !project) return null;

  return (
    <ProjectPageLayout>
      <InstanceDashboard
        parent={parent}
        project={project}
        layout="project"
        onCreate={() =>
          router.push({
            name: PROJECT_V1_ROUTE_INSTANCE_CREATE,
            params: { projectId },
          })
        }
        onOpen={handleOpen}
      />
    </ProjectPageLayout>
  );
}
