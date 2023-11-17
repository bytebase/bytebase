import { Table } from "@/types/v1/schemaEditor";

export const disableChangeTable = (table: Table): boolean => {
  return table.status === "dropped";
};

export const isDroppedTable = (table: Table) => {
  return table.status === "dropped";
};
