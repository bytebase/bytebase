import { useTranslation } from "react-i18next";
import { EnvironmentLabel } from "@/components/EnvironmentLabel";
import { ProjectLabel } from "@/components/ProjectLabel";
import { RouterLink, type RouterLinkProps } from "@/components/RouterLink";
import { cn } from "@/lib/utils";
import {
  environmentNamePrefix,
  projectNamePrefix,
} from "@/stores/modules/v1/common";

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
  const { t } = useTranslation();

  if (resource.startsWith(environmentNamePrefix)) {
    const environmentLink = (
      <RouterLink
        {...linkProps}
        to={{ path: `/${resource}` }}
        className={cn(
          "inline-flex items-center gap-x-1 normal-link",
          linkProps.className
        )}
      >
        <EnvironmentLabel environmentName={resource} />
      </RouterLink>
    );

    if (showResourceType) {
      return (
        <span className="inline-flex min-w-0 max-w-full items-center gap-x-1">
          <span className="text-control-light text-xs">
            {t("common.environment")}:
          </span>
          {environmentLink}
        </span>
      );
    }
    return environmentLink;
  }

  if (resource.startsWith(projectNamePrefix)) {
    if (showResourceType) {
      return (
        <span className="inline-flex min-w-0 max-w-full items-center gap-x-1">
          <span className="text-control-light text-xs">
            {t("common.project")}:
          </span>
          <ProjectLabel
            projectName={resource}
            link={true}
            showResourceType={false}
            {...linkProps}
          />
        </span>
      );
    }
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
