import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  extractDatabaseResourceName,
  getInstanceResource,
} from "@/utils/v1/database";

// Labels a database by its instance title, which tells same-named databases
// apart in a batch.
export function sharePointExportDatabaseLabel(database: Database): string {
  const { instanceName, databaseName } = extractDatabaseResourceName(
    database.name
  );
  const instance = getInstanceResource(database)?.title || instanceName;
  return instance ? `${instance}/${databaseName}` : databaseName;
}
