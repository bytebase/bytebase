import { computed, watch } from "vue";
import { useActuatorV1Store, useAppFeature, useSettingByName } from "./store";
import { defaultAppProfile, type WorkspaceMode } from "./types";
import { DatabaseChangeMode } from "./types/proto/v1/setting_service";
import { useCustomTheme } from "./utils/customTheme";

export const overrideAppProfile = () => {
  const actuatorStore = useActuatorV1Store();
  const setting = useSettingByName("bb.workspace.profile");
  const workspaceMode = computed(() => {
    return setting.value?.value?.workspaceProfileSettingValue
      ?.databaseChangeMode === DatabaseChangeMode.EDITOR
      ? "EDITOR"
      : "CONSOLE";
  });
  actuatorStore.appProfile.mode = workspaceMode.value;
  watch(workspaceMode, (mode) => {
    actuatorStore.appProfile.mode = mode;
  });

  const query = new URLSearchParams(window.location.search);
  overrideAppFeatures(workspaceMode.value, query);
  useCustomTheme(useAppFeature("bb.feature.custom-color-scheme"));

  watch(workspaceMode, (mode) => {
    overrideAppFeatures(mode, query);
  });
};

const overrideAppFeatures = (
  workspaceMode: WorkspaceMode,
  query: URLSearchParams
) => {
  const actuatorStore = useActuatorV1Store();

  actuatorStore.appProfile = defaultAppProfile();
  actuatorStore.appProfile.mode = workspaceMode;

  const modeInQuery = query.get("mode");
  if (modeInQuery === "STANDALONE") {
    // The webapp is embedded within iframe
    actuatorStore.appProfile.embedded = true;

    // mode=STANDALONE is not easy to read, but for legacy support we keep it as
    // some customers are using it.
    actuatorStore.overrideAppFeatures({
      "bb.feature.disable-kbar": true,
      "bb.feature.databases.operations": new Set([
        "CHANGE-DATA",
        "EDIT-SCHEMA",
      ]),
      "bb.feature.hide-banner": true,
      "bb.feature.hide-help": true,
      "bb.feature.hide-quick-start": true,
      "bb.feature.hide-release-remind": true,
      "bb.feature.disallow-navigate-to-console": true,
      "bb.feature.console.hide-sidebar": true,
      "bb.feature.console.hide-header": true,
      "bb.feature.console.hide-quick-action": true,
      "bb.feature.project.hide-default": true,
      "bb.feature.issue.disable-schema-editor": true,
      "bb.feature.issue.hide-subscribers": true,
      "bb.feature.sql-check.hide-doc-link": true,
      "bb.feature.databases.hide-unassigned": true,
      "bb.feature.databases.hide-inalterable": true,
      "bb.feature.sql-editor.disallow-share-worksheet": true,
      "bb.feature.sql-editor.disable-setting": true,
      "bb.feature.sql-editor.disallow-request-query": true,
      "bb.feature.sql-editor.disallow-sync-schema": true,
      "bb.feature.sql-editor.disallow-export-query-data": true,
      "bb.feature.sql-editor.hide-bytebase-logo": true,
      "bb.feature.sql-editor.hide-profile": true,
      "bb.feature.sql-editor.hide-readonly-datasource-hint": true,
    });
  }

  const customTheme = query.get("customTheme");
  if (customTheme === "lixiang") {
    actuatorStore.overrideAppFeatures({
      "bb.feature.custom-color-scheme": {
        "--color-accent": "#00665f",
        "--color-accent-hover": "#00554f",
        "--color-accent-disabled": "#b8c3c3",
      },
      "bb.feature.sql-editor.disallow-export-query-data": true,
    });
    if (actuatorStore.appProfile.embedded) {
      actuatorStore.overrideAppFeatures({
        "bb.feature.issue.hide-review-actions": true,
      });
    }
  }

  if (workspaceMode === "EDITOR") {
    actuatorStore.overrideAppFeatures({
      "bb.feature.hide-quick-start": true,
      "bb.feature.hide-help": true,

      "bb.feature.sql-editor.hide-projects": true,
      "bb.feature.sql-editor.hide-environments": true,
      "bb.feature.sql-editor.disallow-batch-query": true,
      "bb.feature.sql-editor.disallow-share-worksheet": true,
      "bb.feature.sql-editor.hide-advance-instance-features": true,
    });
  }
};
