import { computed } from "vue";
import i18n from "./plugins/i18n";
import { useActuatorV1Store, useSettingByName } from "./store";
import { defaultAppProfile } from "./types";
import {
  DatabaseChangeMode as NewDatabaseChangeMode,
  Setting_SettingName,
} from "./types/proto-es/v1/setting_service_pb";

export const overrideAppProfile = () => {
  const setting = useSettingByName(Setting_SettingName.WORKSPACE_PROFILE);
  const databaseChangeMode = computed(() => {
    if (setting.value?.value?.value?.case === "workspaceProfileSettingValue") {
      const mode = setting.value.value.value.value.databaseChangeMode;
      if (mode === NewDatabaseChangeMode.EDITOR) return NewDatabaseChangeMode.EDITOR;
    }
    return NewDatabaseChangeMode.PIPELINE;
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
  databaseChangeMode: NewDatabaseChangeMode.PIPELINE | NewDatabaseChangeMode.EDITOR
) => {
  const actuatorStore = useActuatorV1Store();

  actuatorStore.appProfile = defaultAppProfile();
  actuatorStore.overrideAppFeatures({
    "bb.feature.database-change-mode": databaseChangeMode,
  });

  if (databaseChangeMode === NewDatabaseChangeMode.EDITOR) {
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
