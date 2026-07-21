import { type MouseEventHandler, type ReactNode, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { PROJECT_V1_ROUTE_DETAIL } from "@/app/router/handles";
import { RouterLink, type RouterLinkProps } from "@/components/RouterLink";
import { useProjectByName } from "@/hooks/useProjectByName";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import { isDefaultProject, isValidProjectName } from "@/types/v1/project";
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
  const validProjectName = isValidProjectName(projectName);
  const defaultProject = isDefaultProject(projectName);
  const shouldFetchProject = children === undefined;

  useEffect(() => {
    if (
      shouldFetchProject &&
      validProjectName &&
      !defaultProject &&
      hasWorkspacePermissionV2("bb.projects.get")
    ) {
      void useAppStore.getState().getOrFetchProjectByName(projectName, true);
    }
  }, [defaultProject, projectName, shouldFetchProject, validProjectName]);

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

  if (!link || defaultProject) {
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
