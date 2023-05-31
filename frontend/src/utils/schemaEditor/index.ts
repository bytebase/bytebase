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
  if (engine === Engine.MYSQL) {
    return [
      "BIT",
      "BOOLEAN",
      "CHAR(1)",
      "DATE",
      "DATETIME",
      "DOUBLE",
      "INT",
      "JSON",
      "VARCHAR(255)",
    ];
  } else if (engine === Engine.POSTGRES) {
    return [
      "BOOLEAN",
      "CHAR(1)",
      "DATE",
      "INTEGER",
      "JSON",
      "REAL",
      "SERIAL",
      "TEXT",
      "VARCHAR(255)",
    ];
  }

  return [];
};
