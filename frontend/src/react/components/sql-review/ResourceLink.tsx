import { ShieldAlert } from "lucide-react";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { RouterLink } from "@/react/components/RouterLink";
import { useEnvironment, usePlanFeature } from "@/react/hooks/useAppState";
import { useProjectByName } from "@/react/hooks/useProjectByName";
import { useAppStore } from "@/react/stores/app";
import {
  environmentNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
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
  const environment = useEnvironment(resource);
  const hasEnvTierFeature = usePlanFeature(
    PlanFeature.FEATURE_ENVIRONMENT_TIERS
  );
  const isProtected =
    hasEnvTierFeature && environment.tags?.protected === "protected";

  return (
    <RouterLink
      to={{ path: `/${resource}` }}
      className="inline-flex items-center gap-x-1 normal-link"
    >
      <span className="text-control-light text-xs mr-0.5">
        {t("common.environment")}:
      </span>
      <span>{environment.title || resource}</span>
      {isProtected && <ShieldAlert className="w-3.5 h-3.5 shrink-0" />}
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
