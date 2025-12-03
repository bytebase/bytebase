import { computed, type MaybeRef, unref } from "vue";
import { useSettingV1Store } from "@/store";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";

export const useSemanticType = (semanticTypeId: MaybeRef<string>) => {
  const settingV1Store = useSettingV1Store();

  const semanticTypeList = computed(() => {
    const setting = settingV1Store.getSettingByName(
      Setting_SettingName.SEMANTIC_TYPES
    );
    if (setting?.value?.value?.case === "semanticType") {
      return setting.value.value.value.types ?? [];
    }
    return [];
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
