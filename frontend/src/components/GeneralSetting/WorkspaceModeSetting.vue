<template>
  <div ref="containerRef" class="py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center space-x-2">
        <h1 class="text-2xl font-bold">
          {{ $t("settings.general.workspace.workspace-mode.self") }}
        </h1>
      </div>
    </div>
    <div class="flex-1 lg:px-4">
      <p class="mb-2 textinfolabel">
        {{ $t("settings.general.workspace.workspace-mode.description.title") }}
      </p>
      <div>
        <NRadioGroup v-model:value="state.workspaceMode" size="large">
          <NSpace vertical>
            <NRadio key="CONSOLE" :value="WorkspaceMode.WORKSPACE_MODE_CONSOLE">
              <div>
                {{
                  $t(
                    "settings.general.workspace.workspace-mode.console-mode.self"
                  )
                }}
              </div>
              <div>
                {{
                  $t(
                    "settings.general.workspace.workspace-mode.console-mode.description"
                  )
                }}
              </div>
            </NRadio>
            <NRadio key="EDITOR" :value="WorkspaceMode.WORKSPACE_MODE_EDITOR">
              <div>
                {{
                  $t(
                    "settings.general.workspace.workspace-mode.editor-mode.self"
                  )
                }}
              </div>
              <div>
                {{
                  $t(
                    "settings.general.workspace.workspace-mode.editor-mode.description"
                  )
                }}
              </div>
            </NRadio>
          </NSpace>
        </NRadioGroup>
      </div>
      <div class="mb-7 mt-4 lg:mt-0">
        <div class="flex justify-end">
          <NButton
            type="primary"
            :disabled="!allowEdit || !allowSave"
            @click.prevent="save"
          >
            {{ $t("common.update") }}
          </NButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { isEqual } from "lodash-es";
import { NRadioGroup, NSpace, NRadio, NButton } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { WorkspaceMode } from "@/types/proto/v1/setting_service";

const getInitialState = (): LocalState => {
  const defaultState: LocalState = {
    workspaceMode: WorkspaceMode.WORKSPACE_MODE_CONSOLE,
  };
  if (
    settingV1Store.workspaceProfileSetting &&
    [
      WorkspaceMode.WORKSPACE_MODE_CONSOLE,
      WorkspaceMode.WORKSPACE_MODE_EDITOR,
    ].includes(settingV1Store.workspaceProfileSetting.workspaceMode)
  ) {
    defaultState.workspaceMode =
      settingV1Store.workspaceProfileSetting.workspaceMode;
  }
  return defaultState;
};

interface LocalState {
  workspaceMode: WorkspaceMode;
}

defineProps<{
  allowEdit: boolean;
}>();

const { t } = useI18n();
const settingV1Store = useSettingV1Store();
const containerRef = ref<HTMLDivElement>();

const state = reactive<LocalState>(getInitialState());

const allowSave = computed((): boolean => {
  return !isEqual(state, getInitialState());
});

const save = async () => {
  await settingV1Store.updateWorkspaceProfile({
    payload: {
      workspaceMode: state.workspaceMode,
    },
    updateMask: ["value.workspace_profile_setting_value.app_mode"],
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.config-updated"),
  });
};
</script>
