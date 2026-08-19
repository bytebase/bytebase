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
  };

export function SQLEditorButton({
  database,
  project,
  label,
  openInNewTab = false,
  disabled = false,
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
    if (database) {
      return autoSQLEditorDatabaseRoute(database);
    }
    if (project) {
      return {
        name: SQL_EDITOR_PROJECT_MODULE,
        params: {
          project: extractProjectResourceName(project.name),
        },
      };
    }

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
      return {
        name: SQL_EDITOR_DATABASE_MODULE,
        params: {
          project: projectId,
          instance: instanceId,
          database: databaseName,
        },
      };
    }
    if (projectId) {
      return {
        name: SQL_EDITOR_PROJECT_MODULE,
        params: {
          project: projectId,
        },
      };
    }
    return { name: SQL_EDITOR_HOME_MODULE };
  }, [database, project, route.params]);

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
