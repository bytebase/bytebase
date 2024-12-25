import { computed } from "vue";
import { useDatabaseCatalog, useSettingV1Store, getColumnCatalog } from "@/store";
import { MaskingAlgorithmSetting_Algorithm as Algorithm } from "@/types/proto/v1/setting_service";

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
  const settingV1Store = useSettingV1Store();

  const rawAlgorithmList = computed((): Algorithm[] => {
    return (
      settingV1Store.getSettingByName("bb.workspace.masking-algorithm")?.value
        ?.maskingAlgorithmSettingValue?.algorithms ?? []
    );
  });

  const databaseCatalog = useDatabaseCatalog(database, false);

  const columnCatalog = computed(() =>
    getColumnCatalog(databaseCatalog.value, schema, table, column)
  );

  const maskingLevel = computed(() => columnCatalog.value.maskingLevel);
  const fullMaskingAlgorithm = computed(() => {
    const id = columnCatalog.value.fullMaskingAlgorithmId;
    return rawAlgorithmList.value.find((a) => a.id === id);
  });
  const partialMaskingAlgorithm = computed(() => {
    const id = columnCatalog.value.partialMaskingAlgorithmId;
    return rawAlgorithmList.value.find((a) => a.id === id);
  });

  return {
    maskingLevel,
    fullMaskingAlgorithm,
    partialMaskingAlgorithm,
  };
};
