import { computed } from "vue";
import { useDBSchemaV1Store, useSettingV1Store } from "@/store";

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
  const dbSchemaV1Store = useDBSchemaV1Store();

  const semanticTypeList = computed(() => {
    return (
      settingV1Store.getSettingByName("bb.workspace.semantic-types")?.value
        ?.semanticTypeSettingValue?.types ?? []
    );
  });

  const semanticType = computed(() => {
    const config = dbSchemaV1Store.getColumnConfig({
      database,
      schema,
      table,
      column,
    });
    if (!config.semanticTypeId) {
      return;
    }
    return semanticTypeList.value.find(
      (data) => data.id === config.semanticTypeId
    );
  });

  return {
    semanticTypeList,
    semanticType,
  };
};
