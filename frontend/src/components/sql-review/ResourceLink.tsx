import { ShieldAlert } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ProjectLabel } from "@/components/ProjectLabel";
import { RouterLink, type RouterLinkProps } from "@/components/RouterLink";
import { useEnvironment, usePlanFeature } from "@/hooks/useAppState";
import { cn } from "@/lib/utils";
import {
  environmentNamePrefix,
  projectNamePrefix,
} from "@/stores/modules/v1/common";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

type ResourceLinkAnchorProps = Pick<
  RouterLinkProps,
  "className" | "rel" | "target"
>;

export function ResourceLink({
  resource,
  showResourceType = true,
  ...linkProps
}: {
  resource: string;
  showResourceType?: boolean;
} & ResourceLinkAnchorProps) {
  if (resource.startsWith(environmentNamePrefix)) {
    return (
      <EnvironmentResourceLink
        resource={resource}
        showResourceType={showResourceType}
        {...linkProps}
      />
    );
  }
  if (resource.startsWith(projectNamePrefix)) {
    return (
      <ProjectLabel
        projectName={resource}
        link={true}
        showResourceType={showResourceType}
        {...linkProps}
      />
    );
  }
  return <span className={linkProps.className}>{resource}</span>;
}

function EnvironmentResourceLink({
  resource,
  showResourceType,
  className,
  ...linkProps
}: {
  resource: string;
  showResourceType: boolean;
} & ResourceLinkAnchorProps) {
  const { t } = useTranslation();
  const environment = useEnvironment(resource);
  const hasEnvTierFeature = usePlanFeature(
    PlanFeature.FEATURE_ENVIRONMENT_TIERS
  );
  const isProtected =
    hasEnvTierFeature && environment.tags?.protected === "protected";

  return (
    <RouterLink
      {...linkProps}
      to={{ path: `/${resource}` }}
      className={cn("inline-flex items-center gap-x-1 normal-link", className)}
    >
      {showResourceType && (
        <span className="text-control-light text-xs mr-0.5">
          {t("common.environment")}:
        </span>
      )}
      <span>{environment.title || resource}</span>
      {isProtected && <ShieldAlert className="w-3.5 h-3.5 shrink-0" />}
    </RouterLink>
  );
}
