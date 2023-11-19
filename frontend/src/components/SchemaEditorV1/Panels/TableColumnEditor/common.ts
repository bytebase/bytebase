import { Column } from "@/types/v1/schemaEditor";

export const isDroppedColumn = (column: Column): boolean => {
  return column.status === "dropped";
};
