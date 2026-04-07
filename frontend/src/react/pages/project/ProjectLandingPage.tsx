import { useEffect } from "react";
import { router } from "@/router";
import {
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_ISSUES,
} from "@/router/dashboard/projectV1";
import { useProjectV1Store } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { hasProjectPermissionV2 } from "@/utils";

export function ProjectLandingPage({ projectId }: { projectId: string }) {
  const projectName = `${projectNamePrefix}${projectId}`;

  useEffect(() => {
    const projectStore = useProjectV1Store();
    let cancelled = false;
    projectStore.getOrFetchProjectByName(projectName).then((project) => {
      if (cancelled) return;
      if (hasProjectPermissionV2(project, "bb.issues.list")) {
        router.replace({ name: PROJECT_V1_ROUTE_ISSUES });
      } else {
        router.replace({ name: PROJECT_V1_ROUTE_DATABASES });
      }
    });
    return () => {
      cancelled = true;
    };
  }, [projectName]);

  return <div />;
}
