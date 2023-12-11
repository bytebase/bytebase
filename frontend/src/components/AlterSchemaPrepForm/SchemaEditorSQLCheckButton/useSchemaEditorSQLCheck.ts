import { Ref, computed } from "vue";
import { ComposedDatabase } from "@/types";

export const useSchemaEditorSQLCheck = (params: {
  databaseList: Ref<ComposedDatabase[]>;
}) => {
  const { databaseList } = params;

  const show = computed(() => {
    // SQL Check is highly related to the databases' environments.
    // By now we cannot handle mixed environments correctly.
    // so we just support SQL Check when editing single database's schema.
    return databaseList.value.length === 1;
  });

  const database = computed(() => {
    return databaseList.value[0];
  });

  return { show, database };
};
