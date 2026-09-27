import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  escapeMongoDBCollectionName,
  generateSchemaAndTableNameInSQL,
} from "@/utils/v1/sql";

export interface DataExplorerQueryTarget {
  engine: Engine;
  schema: string;
  table: string;
}

export const getDataExplorerQueryPrefix = ({
  engine,
  schema,
  table,
}: DataExplorerQueryTarget): string | undefined => {
  if (!table && engine !== Engine.COSMOSDB) return undefined;
  switch (engine) {
    case Engine.ENGINE_UNSPECIFIED:
    case Engine.REDIS:
      return undefined;
    case Engine.COSMOSDB:
      return "SELECT * FROM c";
    case Engine.MONGODB:
      return `db["${escapeMongoDBCollectionName(table)}"].find(`;
    case Engine.ELASTICSEARCH:
      return `GET ${table}/_search?q=`;
    default:
      return `SELECT * FROM ${generateSchemaAndTableNameInSQL(engine, schema, table)}`;
  }
};

export const getDataExplorerStatement = (
  target: DataExplorerQueryTarget,
  filter: string,
  limit: number
): string | undefined => {
  const prefix = getDataExplorerQueryPrefix(target);
  if (!prefix) return undefined;
  const suffix = filter.trim().replace(/;$/, "");

  switch (target.engine) {
    case Engine.COSMOSDB:
      return suffix ? `${prefix} ${suffix}` : prefix;
    case Engine.MONGODB:
      return `${prefix}${suffix});`;
    case Engine.ELASTICSEARCH:
      return `${prefix}${suffix || "*"}&size=${limit}`;
    default:
      return suffix ? `${prefix} ${suffix};` : `${prefix};`;
  }
};
