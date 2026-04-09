import { Terminal } from "lucide-react";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { router } from "@/router";
import { SQL_EDITOR_DATABASE_MODULE } from "@/router/sqlEditor";
import { usePermissionStore } from "@/store";
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
  const permissionStore = usePermissionStore();

  const handleClick = useCallback(() => {
    if (disabled) {
      return;
    }

    if (isDefaultProject(database.project)) {
      if (
        !permissionStore.currentPermissions.has("bb.sql.select") &&
        !permissionStore
          .currentPermissionsInProjectV1(defaultProject(database.project))
          .has("bb.sql.select")
      ) {
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
  }, [database, disabled, onFailed, permissionStore]);

  return (
    <Button variant="ghost" size="sm" disabled={disabled} onClick={handleClick}>
      <Terminal className="h-4 w-4" />
      {t("sql-editor.self")}
    </Button>
  );
}
