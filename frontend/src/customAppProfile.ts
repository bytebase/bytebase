import { computed } from "vue";
import { locale } from "./plugins/i18n";
import { useActuatorV1Store, useSettingV1Store } from "./store";
import { defaultAppProfile } from "./types";
import { DatabaseChangeMode } from "./types/proto-es/v1/setting_service_pb";

export const overrideAppProfile = () => {
  overrideAppFeatures();

  // Override app language.
  const query = new URLSearchParams(window.location.search);
  const lang = query.get("lang");
  if (lang) {
    locale.value = lang;
  }
};

const overrideAppFeatures = () => {
  const actuatorStore = useActuatorV1Store();
  const settingV1Store = useSettingV1Store();

  const databaseChangeMode = computed(() => {
    const mode = settingV1Store.workspaceProfile.databaseChangeMode;
    if (mode === DatabaseChangeMode.EDITOR) return DatabaseChangeMode.EDITOR;
    return DatabaseChangeMode.PIPELINE;
  });

  actuatorStore.appProfile = defaultAppProfile();
  actuatorStore.overrideAppFeatures({
    "bb.feature.database-change-mode": databaseChangeMode.value,
  });

  if (databaseChangeMode.value === DatabaseChangeMode.EDITOR) {
    actuatorStore.overrideAppFeatures({
      "bb.feature.hide-quick-start": true,
      "bb.feature.hide-help": true,
      "bb.feature.hide-trial": true,
    });
  }
};
