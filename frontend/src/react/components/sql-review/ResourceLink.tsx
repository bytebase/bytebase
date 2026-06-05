import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { RouterLink } from "@/react/components/RouterLink";
import { useProjectByName } from "@/react/hooks/useProjectByName";
import { useAppStore } from "@/react/stores/app";
import {
  environmentNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { hasWorkspacePermissionV2 } from "@/utils";

export function ResourceLink({ resource }: { resource: string }) {
  if (resource.startsWith(environmentNamePrefix)) {
    return <EnvironmentResourceLink resource={resource} />;
  }
  if (resource.startsWith(projectNamePrefix)) {
    return <ProjectResourceLink resource={resource} />;
  }
  return <span>{resource}</span>;
}

function EnvironmentResourceLink({ resource }: { resource: string }) {
  const { t } = useTranslation();
  return (
    <RouterLink
      to={{ path: `/${resource}` }}
      className="inline-flex items-center gap-x-1"
    >
      <span className="text-control-light text-xs mr-0.5">
        {t("common.environment")}:
      </span>
      <EnvironmentLabel environmentName={resource} />
    </RouterLink>
  );
}

function ProjectResourceLink({ resource }: { resource: string }) {
  const { t } = useTranslation();
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);

  useEffect(() => {
    if (hasWorkspacePermissionV2("bb.projects.get")) {
      useAppStore.getState().getOrFetchProjectByName(resource, true);
    }
  }, [resource]);

  const project = useProjectByName(resource);
  void projectsByName;

  return (
    <RouterLink
      to={{ path: `/${resource}` }}
      className="inline-flex items-center gap-x-1 normal-link"
    >
      <span className="text-control-light text-xs mr-0.5">
        {t("common.project")}:
      </span>
      <span>{project?.title || resource}</span>
    </RouterLink>
  );
}
