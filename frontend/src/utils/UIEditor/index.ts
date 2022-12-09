import { Database, EngineType } from "@/types";
import { isDev } from "..";

// Only allow using UI Editor with MySQL in dev mode.
export const allowUsingUIEditor = (databaseList: Database[]): boolean => {
  return (
    isDev() &&
    databaseList.every((db) => {
      return db.instance.engine === "MYSQL";
    })
  );
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
