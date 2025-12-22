import { computed } from "vue";
import { locale } from "./plugins/i18n";
import { useActuatorV1Store, useSettingByName } from "./store";
import { defaultAppProfile } from "./types";
import {
  DatabaseChangeMode,
  Setting_SettingName,
} from "./types/proto-es/v1/setting_service_pb";

export const overrideAppProfile = () => {
  const setting = useSettingByName(Setting_SettingName.WORKSPACE_PROFILE);
  const databaseChangeMode = computed(() => {
    if (setting.value?.value?.value?.case === "workspaceProfile") {
      const mode = setting.value.value.value.value.databaseChangeMode;
      if (mode === DatabaseChangeMode.EDITOR) return DatabaseChangeMode.EDITOR;
    }
    return DatabaseChangeMode.PIPELINE;
  });

  overrideAppFeatures(databaseChangeMode.value);

  // Override app language.
  const query = new URLSearchParams(window.location.search);
  const lang = query.get("lang");
  if (lang) {
    locale.value = lang;
  }
};

const overrideAppFeatures = (
  databaseChangeMode: DatabaseChangeMode.PIPELINE | DatabaseChangeMode.EDITOR
) => {
  const actuatorStore = useActuatorV1Store();

  actuatorStore.appProfile = defaultAppProfile();
  actuatorStore.overrideAppFeatures({
    "bb.feature.database-change-mode": databaseChangeMode,
  });

  if (databaseChangeMode === DatabaseChangeMode.EDITOR) {
    actuatorStore.overrideAppFeatures({
      "bb.feature.hide-quick-start": true,
      "bb.feature.hide-help": true,
      "bb.feature.hide-trial": true,
    });
  }
};
