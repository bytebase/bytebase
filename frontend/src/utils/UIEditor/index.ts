import { Database, EngineType } from "@/types";

// Only allow using UI Editor with MySQL.
export const allowUsingUIEditor = (databaseList: Database[]): boolean => {
  return databaseList.every((db) => {
    return db.instance.engine === "MYSQL";
  });
};

export const getDataTypeSuggestionList = (engineType: EngineType = "MYSQL") => {
  if (engineType === "MYSQL") {
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
  }

  return [];
};
