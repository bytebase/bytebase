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
      "bit",
      "boolean",
      "char(1)",
      "date",
      "datetime",
      "double",
      "int",
      "json",
      "varchar(255)",
    ];
  }

  return [];
};
