import { type MouseEventHandler, type ReactNode, useEffect } from "react";
import { useTranslation } from "react-i18next";
import {
  RouterLink,
  type RouterLinkProps,
} from "@/react/components/RouterLink";
import { useProjectByName } from "@/react/hooks/useProjectByName";
import { cn } from "@/react/lib/utils";
import { PROJECT_V1_ROUTE_DETAIL } from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { extractProjectResourceName, hasWorkspacePermissionV2 } from "@/utils";

export function ProjectLabel({
  children,
  projectName,
  showResourceType = false,
  className,
  link,
  onClick,
  ...linkProps
}: {
  children?: ReactNode;
  projectName: string;
  showResourceType?: boolean;
  link?: boolean;
  onClick?: MouseEventHandler<HTMLAnchorElement>;
} & Pick<RouterLinkProps, "className" | "rel" | "target">) {
  const { t } = useTranslation();
  const project = useProjectByName(projectName);
  const projectId = extractProjectResourceName(projectName);
  const validProjectName =
    projectName.startsWith(projectNamePrefix) && projectId.length > 0;
  const shouldFetchProject = children === undefined;

  useEffect(() => {
    if (
      shouldFetchProject &&
      validProjectName &&
      hasWorkspacePermissionV2("bb.projects.get")
    ) {
      void useAppStore.getState().getOrFetchProjectByName(projectName, true);
    }
  }, [projectName, shouldFetchProject, validProjectName]);

  if (!validProjectName) {
    return (
      <span className={className}>{children ?? (projectName || "-")}</span>
    );
  }

  const label =
    project.name === projectName && project.title ? project.title : projectId;
  const content: ReactNode = children ?? (
    <>
      {showResourceType && (
        <span className="mr-1 text-xs text-control-light">
          {t("common.project")}:
        </span>
      )}
      <span className="truncate">{label}</span>
    </>
  );

  if (!link) {
    if (className) {
      return <span className={className}>{content}</span>;
    }
    return content;
  }

  return (
    <RouterLink
      {...linkProps}
      to={{
        name: PROJECT_V1_ROUTE_DETAIL,
        params: { projectId },
      }}
      className={cn(
        "inline-flex min-w-0 max-w-full normal-link hover:underline",
        className
      )}
      onClick={(event) => {
        event.stopPropagation();
        onClick?.(event);
      }}
    >
      {content}
    </RouterLink>
  );
}
