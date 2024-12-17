import { computed } from "vue";
import { useDBSchemaV1Store, useSettingV1Store } from "@/store";
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
  const dbSchemaV1Store = useDBSchemaV1Store();

  const rawAlgorithmList = computed((): Algorithm[] => {
    return (
      settingV1Store.getSettingByName("bb.workspace.masking-algorithm")?.value
        ?.maskingAlgorithmSettingValue?.algorithms ?? []
    );
  });

  const columnConfig = computed(() =>
    dbSchemaV1Store.getColumnConfig({
      database,
      schema,
      table,
      column,
    })
  );

  const maskingLevel = computed(() => columnConfig.value.maskingLevel);
  const fullMaskingAlgorithm = computed(() => {
    const id = columnConfig.value.fullMaskingAlgorithmId;
    return rawAlgorithmList.value.find((a) => a.id === id);
  });
  const partialMaskingAlgorithm = computed(() => {
    const id = columnConfig.value.partialMaskingAlgorithmId;
    return rawAlgorithmList.value.find((a) => a.id === id);
  });

  return {
    maskingLevel,
    fullMaskingAlgorithm,
    partialMaskingAlgorithm,
  };
};
