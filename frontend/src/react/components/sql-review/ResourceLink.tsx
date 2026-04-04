import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { useProjectV1Store } from "@/store";
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
    <a
      href={`/${resource}`}
      className="inline-flex items-center gap-x-1"
      onClick={(e) => {
        e.preventDefault();
        router.push({ path: `/${resource}` });
      }}
    >
      <span className="text-control-light text-xs mr-0.5">
        {t("common.environment")}:
      </span>
      <EnvironmentLabel environmentName={resource} />
    </a>
  );
}

function ProjectResourceLink({ resource }: { resource: string }) {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();

  useEffect(() => {
    if (hasWorkspacePermissionV2("bb.projects.get")) {
      projectStore.getOrFetchProjectByName(resource, true);
    }
  }, [projectStore, resource]);

  const project = useVueState(() => projectStore.getProjectByName(resource));

  return (
    <a
      href={`/${resource}`}
      className="inline-flex items-center gap-x-1 normal-link"
      onClick={(e) => {
        e.preventDefault();
        router.push({ path: `/${resource}` });
      }}
    >
      <span className="text-control-light text-xs mr-0.5">
        {t("common.project")}:
      </span>
      <span>{project?.title || resource}</span>
    </a>
  );
}
