import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  escapeMongoDBCollectionName,
  generateSchemaAndTableNameInSQL,
} from "@/utils/v1/sql";

const DATA_EXPLORER_LIMIT = 50;

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
    case Engine.MSSQL:
      return `SELECT TOP ${DATA_EXPLORER_LIMIT} * FROM ${generateSchemaAndTableNameInSQL(engine, schema, table)}`;
    default:
      return `SELECT * FROM ${generateSchemaAndTableNameInSQL(engine, schema, table)}`;
  }
};

export const getDataExplorerStatement = (
  target: DataExplorerQueryTarget,
  filter: string
): string | undefined => {
  const prefix = getDataExplorerQueryPrefix(target);
  if (!prefix) return undefined;
  const suffix = filter.trim().replace(/;$/, "");

  switch (target.engine) {
    case Engine.COSMOSDB:
      return suffix ? `${prefix} ${suffix}` : prefix;
    case Engine.MONGODB:
      return `${prefix}${suffix}).limit(${DATA_EXPLORER_LIMIT});`;
    case Engine.ELASTICSEARCH:
      return `${prefix}${suffix || "*"}&size=${DATA_EXPLORER_LIMIT}`;
    case Engine.MSSQL:
      return `${prefix}${suffix ? ` ${suffix}` : ""};`;
    case Engine.ORACLE:
      if (!suffix) return `${prefix} WHERE ROWNUM <= ${DATA_EXPLORER_LIMIT};`;
      return `SELECT * FROM (${prefix} ${suffix}) WHERE ROWNUM <= ${DATA_EXPLORER_LIMIT};`;
    default:
      return `${prefix}${suffix ? ` ${suffix}` : ""} LIMIT ${DATA_EXPLORER_LIMIT};`;
  }
};

export const getDataExplorerFilterPlaceholderKey = (engine: Engine) => {
  switch (engine) {
    case Engine.COSMOSDB:
      return "sql-editor.data-explorer-filter-placeholder";
    case Engine.MONGODB:
      return "sql-editor.data-explorer-mongodb-filter-placeholder";
    case Engine.ELASTICSEARCH:
      return "sql-editor.data-explorer-elasticsearch-filter-placeholder";
    default:
      return "sql-editor.data-explorer-sql-filter-placeholder";
  }
};
