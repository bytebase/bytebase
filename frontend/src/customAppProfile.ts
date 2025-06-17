import { computed } from "vue";
import i18n from "./plugins/i18n";
import { useActuatorV1Store, useSettingByName } from "./store";
import { defaultAppProfile } from "./types";
import {
  DatabaseChangeMode,
  Setting_SettingName,
} from "./types/proto/v1/setting_service";

export const overrideAppProfile = () => {
  const setting = useSettingByName(Setting_SettingName.WORKSPACE_PROFILE);
  const databaseChangeMode = computed(() => {
    const mode =
      setting.value?.value?.workspaceProfileSettingValue?.databaseChangeMode;
    if (mode === DatabaseChangeMode.EDITOR) return DatabaseChangeMode.EDITOR;
    return DatabaseChangeMode.PIPELINE;
  });

  overrideAppFeatures(databaseChangeMode.value);

  // Override app language.
  const query = new URLSearchParams(window.location.search);
  const lang = query.get("lang");
  if (lang) {
    i18n.global.locale.value = lang;
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

  if (databaseChangeMode === "EDITOR") {
    actuatorStore.overrideAppFeatures({
      "bb.feature.hide-quick-start": true,
      "bb.feature.hide-help": true,
      "bb.feature.hide-trial": true,
      "bb.feature.sql-editor.disallow-edit-schema": true,
      "bb.feature.sql-editor.sql-check-style": "PREFLIGHT",
      "bb.feature.sql-editor.disallow-request-query": true,
      "bb.feature.sql-editor.enable-setting": true,
      "bb.feature.databases.operations": new Set([
        "SYNC-SCHEMA",
        "EDIT-LABELS",
        "TRANSFER-OUT",
      ]),
    });
  }
};
