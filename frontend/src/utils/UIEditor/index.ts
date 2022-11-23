import { Database, EngineType } from "@/types";

// Only allow UI Editor with MySQL.
export const allowUsingUIEditor = (databaseList: Database[]): boolean => {
  return databaseList.every((db) => {
    return db.instance.engine === "MYSQL";
  });
};

export const getDataTypeSuggestionList = (engineType: EngineType = "MYSQL") => {
  if (engineType === "MYSQL") {
    return [
      "bigint",
      "binary",
      "blob",
      "boolean",
      "char",
      "date",
      "enum",
      "int",
      "json",
      "varchar(255)",
    ];
  }

  return [];
};
