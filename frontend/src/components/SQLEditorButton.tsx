import { SquareTerminal } from "lucide-react";
import { type ReactNode, useMemo } from "react";
import { useTranslation } from "react-i18next";
import {
  SQL_EDITOR_DATABASE_MODULE,
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_PROJECT_MODULE,
  useCurrentRoute,
} from "@/app/router";
import { RouterLink, type RouterLinkProps } from "@/components/RouterLink";
import { type ButtonProps, buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { autoSQLEditorDatabaseRoute } from "@/utils/auto-route";
import { extractProjectResourceName } from "@/utils/v1/project";

export type SQLEditorButtonProps = Omit<
  RouterLinkProps,
  "children" | "className" | "onClick" | "rel" | "target" | "to"
> &
  Pick<ButtonProps, "appearance" | "size" | "variant" | "className"> & {
    database?: Pick<Database, "name" | "project">;
    project?: Pick<Project, "name">;
    label?: ReactNode;
    openInNewTab?: boolean;
    disabled?: boolean;
    query?: Record<string, string>;
  };

export function SQLEditorButton({
  database,
  project,
  label,
  openInNewTab = false,
  disabled = false,
  query,
  appearance,
  size,
  variant,
  className,
  tabIndex,
  "aria-label": ariaLabel,
  ...props
}: SQLEditorButtonProps) {
  const { t } = useTranslation();
  const route = useCurrentRoute();
  const defaultLabel = t("sql-editor.self");
  const accessibleLabel =
    ariaLabel ?? (typeof label === "string" ? label : defaultLabel);
  const to = useMemo(() => {
    let target;
    if (database) {
      target = autoSQLEditorDatabaseRoute(database);
    } else if (project) {
      target = {
        name: SQL_EDITOR_PROJECT_MODULE,
        params: {
          project: extractProjectResourceName(project.name),
        },
      };
    } else {
      const projectId = getRouteParam(
        route.params.projectId ?? route.params.project
      );
      const instanceId = getRouteParam(
        route.params.instanceId ?? route.params.instance
      );
      const databaseName = getRouteParam(
        route.params.databaseName ?? route.params.database
      );
      if (projectId && instanceId && databaseName) {
        target = {
          name: SQL_EDITOR_DATABASE_MODULE,
          params: {
            project: projectId,
            instance: instanceId,
            database: databaseName,
          },
        };
      } else if (projectId) {
        target = {
          name: SQL_EDITOR_PROJECT_MODULE,
          params: {
            project: projectId,
          },
        };
      } else {
        target = { name: SQL_EDITOR_HOME_MODULE };
      }
    }
    return query ? { ...target, query } : target;
  }, [database, project, query, route.params]);

  return (
    <RouterLink
      {...props}
      to={to}
      target={openInNewTab ? "_blank" : undefined}
      rel={openInNewTab ? "noopener noreferrer" : undefined}
      aria-label={accessibleLabel}
      aria-disabled={disabled || undefined}
      tabIndex={disabled ? -1 : tabIndex}
      className={buttonVariants({
        appearance,
        size,
        variant,
        className: cn(className, disabled && "cursor-not-allowed opacity-50"),
      })}
      onClickCapture={
        disabled
          ? (event) => {
              event.preventDefault();
            }
          : undefined
      }
    >
      <SquareTerminal className="size-4" />
      {label === undefined ? defaultLabel : label}
    </RouterLink>
  );
}

function getRouteParam(value: string | string[] | undefined) {
  return typeof value === "string" ? value : undefined;
}
