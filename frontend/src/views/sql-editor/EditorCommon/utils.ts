import { head } from "lodash-es";
import type { ComposedDatabase, SQLEditorConnection } from "@/types";
import { DataSourceType } from "@/types/proto-es/v1/instance_service_pb";

const KEY_WITH_POSITION_DELIMITER = "###";

export const keyWithPosition = (key: string, position: number) => {
  return `${key}${KEY_WITH_POSITION_DELIMITER}${position}`;
};

export const extractKeyWithPosition = (key: string) => {
  const [maybeKey, maybePosition] = key.split(KEY_WITH_POSITION_DELIMITER);
  const position = parseInt(maybePosition, 10);
  return [maybeKey, Number.isNaN(position) ? -1 : position];
};

export const setDefaultDataSourceForConn = (
  conn: SQLEditorConnection,
  database: ComposedDatabase
) => {
  if (conn.dataSourceId) {
    return;
  }

  // Default connect to the first read-only data source if available.
  // Skip checking env/project policy for now.
  const defaultDataSource =
    head(
      database.instanceResource.dataSources.filter(
        (d) => d.type === DataSourceType.READ_ONLY
      )
    ) || head(database.instanceResource.dataSources);
  if (defaultDataSource) {
    conn.dataSourceId = defaultDataSource.id;
  }
};
