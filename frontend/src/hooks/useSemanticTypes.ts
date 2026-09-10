import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useAppStore } from "@/stores/app";
import type { SemanticTypeSetting_SemanticType } from "@/types/proto-es/v1/setting_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { getSemanticTypeListWithBuiltins } from "@/types/semanticTypes";

export const useSemanticTypes = (): {
  configuredSemanticTypes: SemanticTypeSetting_SemanticType[];
  semanticTypes: SemanticTypeSetting_SemanticType[];
} => {
  const { t } = useTranslation();
  const setting = useAppStore((state) =>
    state.getSettingByName(Setting_SettingName.SEMANTIC_TYPES)
  );

  return useMemo(() => {
    const configuredSemanticTypes =
      setting?.value?.value.case === "semanticType"
        ? setting.value.value.value.types
        : [];
    return {
      configuredSemanticTypes,
      semanticTypes: getSemanticTypeListWithBuiltins(
        configuredSemanticTypes,
        t
      ),
    };
  }, [setting, t]);
};
