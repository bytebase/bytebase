import { computed } from "vue";
import { useDatabaseCatalog, getColumnCatalog } from "@/store";

export const useColumnMasking = ({
  database,
  schema,
  table,
  column,
}: {
  database: string;
  schema: string;
  table: string;
  column: string;
}) => {
  const databaseCatalog = useDatabaseCatalog(database, false);

  const columnCatalog = computed(() =>
    getColumnCatalog(databaseCatalog.value, schema, table, column)
  );

  const maskingLevel = computed(() => columnCatalog.value.maskingLevel);

  return {
    maskingLevel,
  };
};
