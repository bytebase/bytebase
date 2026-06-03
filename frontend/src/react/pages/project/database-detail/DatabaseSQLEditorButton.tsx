import { SquareTerminal } from "lucide-react";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/react/router";
import { SQL_EDITOR_DATABASE_MODULE } from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { defaultProject, isDefaultProject } from "@/types/v1/project";

const extractDatabaseParts = (resource: string) => {
  const matches = resource.match(
    /(?:^|\/)instances\/(?<instanceName>[^/]+)\/databases\/(?<databaseName>[^/]+)(?:$|\/)/
  );
  return {
    instanceName: matches?.groups?.instanceName ?? "",
    databaseName: matches?.groups?.databaseName ?? "",
  };
};

const extractProjectId = (resource: string) => {
  const matches = resource.match(/(?:^|\/)projects\/([^/]+)(?:$|\/)/);
  return matches?.[1] ?? "";
};

export function DatabaseSQLEditorButton({
  database,
  disabled = false,
  onFailed,
}: {
  database: Database;
  disabled?: boolean;
  onFailed?: (database: Database) => void;
}) {
  const { t } = useTranslation();
  const hasProjectPermissionFn = useAppStore(
    (state) => state.hasProjectPermission
  );
  const hasWorkspacePermissionFn = useAppStore(
    (state) => state.hasWorkspacePermission
  );

  const handleClick = useCallback(() => {
    if (disabled) {
      return;
    }

    if (isDefaultProject(database.project)) {
      const canQuery =
        hasWorkspacePermissionFn("bb.sql.select") ||
        hasProjectPermissionFn(
          defaultProject(database.project),
          "bb.sql.select"
        );
      if (!canQuery) {
        onFailed?.(database);
        return;
      }
    }

    const { instanceName, databaseName } = extractDatabaseParts(database.name);
    const route = router.resolve({
      name: SQL_EDITOR_DATABASE_MODULE,
      params: {
        project: extractProjectId(database.project),
        instance: instanceName,
        database: databaseName,
      },
    });

    if (router.currentRoute.value.name?.toString().startsWith("sql-editor")) {
      void router.push(route);
      return;
    }

    window.open(route.fullPath, "_blank");
  }, [
    database,
    disabled,
    hasProjectPermissionFn,
    hasWorkspacePermissionFn,
    onFailed,
  ]);

  return (
    <dd
      className="flex cursor-pointer items-center text-sm textlabel hover:text-accent md:mr-4"
      onClick={handleClick}
    >
      <SquareTerminal className="mr-1 size-4" />
      {t("sql-editor.self")}
    </dd>
  );
}
