import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

const BACKUP_AVAILABLE_ENGINES = [
  Engine.MYSQL,
  Engine.MARIADB,
  Engine.TIDB,
  Engine.MSSQL,
  Engine.ORACLE,
  Engine.POSTGRES,
];

export const getPlanOptionVisibility = ({
  databases,
  isSheetBasedDatabaseChange,
  isChangeDatabaseConfig,
}: {
  databases: Database[];
  isChangeDatabaseConfig: boolean;
  isSheetBasedDatabaseChange: boolean;
}) => {
  const hasDatabases = databases.length > 0;
  const showTransactionMode =
    hasDatabases &&
    isSheetBasedDatabaseChange &&
    databases.every((database) =>
      instanceSupportsTransactionMode(database.instanceResource?.engine)
    );
  const showInstanceRole =
    hasDatabases &&
    isSheetBasedDatabaseChange &&
    databases.every(
      (database) => database.instanceResource?.engine === Engine.POSTGRES
    );
  const showIsolationLevel =
    hasDatabases &&
    isSheetBasedDatabaseChange &&
    databases.every((database) =>
      [Engine.MYSQL, Engine.MARIADB, Engine.TIDB].includes(
        database.instanceResource?.engine ?? Engine.ENGINE_UNSPECIFIED
      )
    );
  const showPreBackup =
    hasDatabases &&
    isChangeDatabaseConfig &&
    databases.every((database) =>
      BACKUP_AVAILABLE_ENGINES.includes(
        database.instanceResource?.engine ?? Engine.ENGINE_UNSPECIFIED
      )
    );
  const showGhost =
    hasDatabases &&
    isSheetBasedDatabaseChange &&
    databases.every((database) =>
      allowGhostForDatabase(
        database.instanceResource?.engine ?? Engine.ENGINE_UNSPECIFIED,
        database.instanceResource?.engineVersion ?? ""
      )
    );

  return {
    showGhost,
    showInstanceRole,
    showIsolationLevel,
    showPreBackup,
    showTransactionMode,
    shouldShow:
      showTransactionMode ||
      showInstanceRole ||
      showIsolationLevel ||
      showPreBackup ||
      showGhost,
  };
};

export function allowGhostForDatabase(engine: Engine, engineVersion: string) {
  if (engine === Engine.MYSQL) return engineVersion >= "5.6.0";
  if (engine === Engine.MARIADB) return engineVersion >= "10.6.0";
  return false;
}

const instanceSupportsTransactionMode = (engine?: Engine): boolean => {
  if (engine === undefined) return false;
  return ![
    Engine.MONGODB,
    Engine.REDIS,
    Engine.DYNAMODB,
    Engine.COSMOSDB,
    Engine.CASSANDRA,
    Engine.ELASTICSEARCH,
  ].includes(engine);
};
