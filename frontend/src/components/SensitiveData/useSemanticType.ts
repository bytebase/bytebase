import { computed } from "vue";
import { useSettingV1Store, useDatabaseCatalog, getColumnCatalog } from "@/store";

export const useSemanticType = ({
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
  const settingV1Store = useSettingV1Store();

  const semanticTypeList = computed(() => {
    return (
      settingV1Store.getSettingByName("bb.workspace.semantic-types")?.value
        ?.semanticTypeSettingValue?.types ?? []
    );
  });

  const databaseCatalog = useDatabaseCatalog(database, false);

  const semanticType = computed(() => {
    const columnCatalog = getColumnCatalog(databaseCatalog.value, schema, table, column)
    if (!columnCatalog.semanticType) {
      return;
    }
    return semanticTypeList.value.find(
      (data) => data.id === columnCatalog.semanticType
    );
  });

  return {
    semanticTypeList,
    semanticType,
  };
};
