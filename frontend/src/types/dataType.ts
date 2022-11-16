import { EngineType } from "@/types";

export const getDataTypeList = (engineType: EngineType = "MYSQL") => {
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
