import { computed, unref, type MaybeRef } from "vue";
import { useSettingV1Store } from "@/store";

export const useSemanticType = (semanticTypeId: MaybeRef<string>) => {
  const settingV1Store = useSettingV1Store();

  const semanticTypeList = computed(() => {
    return (
      settingV1Store.getSettingByName("bb.workspace.semantic-types")?.value
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
