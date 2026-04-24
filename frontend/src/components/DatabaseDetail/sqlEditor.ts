import type { Router } from "vue-router";
import { SQL_EDITOR_DATABASE_MODULE } from "@/router/sqlEditor";
import { defaultProject, isDefaultProject } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  extractDatabaseResourceName,
  extractInstanceResourceName,
  extractProjectResourceName,
  hasProjectPermissionV2,
  isSQLEditorRoute,
} from "@/utils";

export const openSQLEditorForDatabase = ({
  router,
  database,
  schema,
  table,
  onPermissionDenied,
}: {
  router: Router;
  database: Database;
  schema?: string;
  table?: string;
  onPermissionDenied?: (database: Database) => void;
}) => {
  if (isDefaultProject(database.project)) {
    if (
      !hasProjectPermissionV2(defaultProject(database.project), "bb.sql.select")
    ) {
      onPermissionDenied?.(database);
      return;
    }
  }

  const { instance, databaseName } = extractDatabaseResourceName(database.name);
  const route = router.resolve({
    name: SQL_EDITOR_DATABASE_MODULE,
    params: {
      project: extractProjectResourceName(database.project),
      instance: extractInstanceResourceName(instance),
      database: databaseName,
    },
    query: {
      schema: schema || undefined,
      table: table || undefined,
    },
  });
  if (isSQLEditorRoute(router)) {
    router.push(route);
  } else {
    window.open(route.fullPath);
  }
};
