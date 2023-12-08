import { ComposedDatabase, Database } from "@/types";
import { Engine } from "@/types/proto/v1/common";

// Only allow using Schema Editor with MySQL.
export const allowUsingSchemaEditor = (databaseList: Database[]): boolean => {
  return databaseList.every((db) => {
    return db.instance.engine === "MYSQL" || db.instance.engine === "POSTGRES";
  });
};

export const allowUsingSchemaEditorV1 = (
  databaseList: ComposedDatabase[]
): boolean => {
  const supported = new Set([Engine.MYSQL, Engine.POSTGRES]);
  return databaseList.every((db) => {
    return supported.has(db.instanceEntity.engine);
  });
};

export const getDataTypeSuggestionList = (engine: Engine = Engine.MYSQL) => {
  if (engine === Engine.MYSQL || engine === Engine.TIDB) {
    return [
      "bigint",
      "binary",
      "bit",
      "blob",
      "boolean",
      // char is equivalent to char(1).
      "char",
      "char(255)",
      "date",
      "datetime",
      "decimal",
      "double",
      "enum",
      "float",
      "geometry",
      "geometrycollection",
      "int",
      "json",
      "linestring",
      "longblob",
      "longtext",
      "mediumblob",
      "mediumint",
      "mediumtext",
      "multilinestring",
      "multipoint",
      "multipolygon",
      "point",
      "polygon",
      "smallint",
      "text",
      "time",
      "timestamp",
      "tinublob",
      "tinyint",
      "tinytext",
      // Unlike Postgres, MySQL does not support varchar with no length specified.
      "varchar(255)",
      "year",
    ];
  } else if (engine === Engine.POSTGRES) {
    return [
      "bigserial",
      "bit",
      "bool",
      "box",
      "bytea",
      "char",
      "char(255)",
      "cidr",
      "circle",
      "date",
      "decimal",
      "float4",
      "float8",
      "inet",
      "int2",
      "int4",
      "int8",
      "interval",
      "json",
      "jsonb",
      "line",
      "lseg",
      "macaddr",
      "money",
      "numeric",
      "path",
      "point",
      "polygon",
      "serial",
      "serial2",
      "serial4",
      "serial8",
      "smallserial",
      "text",
      "time",
      "timestamp",
      "timestamptz",
      "timetz",
      "tsquery",
      "tsvector",
      "txid_snaps",
      "uuid",
      "varbit",
      "varchar",
      "varchar(255)",
      "xml",
    ];
  }

  return [];
};
