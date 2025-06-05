import { computed, unref, type MaybeRef } from "vue";
import { useSettingV1Store } from "@/store";
import { Setting_SettingName } from "@/types/proto/v1/setting_service";

export const useSemanticType = (semanticTypeId: MaybeRef<string>) => {
  const settingV1Store = useSettingV1Store();

  const semanticTypeList = computed(() => {
    return (
      settingV1Store.getSettingByName(Setting_SettingName.SEMANTIC_TYPES)?.value
        ?.semanticTypeSettingValue?.types ?? []
    );
  });

  const semanticType = computed(() => {
    const id = unref(semanticTypeId);
    if (!id) {
      return;
    }
    return semanticTypeList.value.find((data) => data.id === id);
  });

  return {
    semanticTypeList,
    semanticType,
  };
};
