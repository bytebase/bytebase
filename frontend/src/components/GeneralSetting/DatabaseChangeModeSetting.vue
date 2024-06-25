<template>
  <div ref="containerRef" class="py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center space-x-2">
        <h1 class="text-2xl font-bold">
          {{ $t("settings.general.workspace.database-change-mode.self") }}
        </h1>
      </div>
    </div>
    <div class="flex-1 lg:px-4">
      <p class="mb-2 textinfolabel">
        {{ $t("settings.general.workspace.database-change-mode.description") }}
      </p>
      <div>
        <NRadioGroup v-model:value="state.databaseChangeMode" size="large">
          <NSpace vertical>
            <NRadio
              key="PIPELINE"
              :value="DatabaseChangeMode.PIPELINE"
              :label="
                $t(
                  'settings.general.workspace.database-change-mode.issue-mode.self'
                )
              "
            />
            <NRadio
              key="EDITOR"
              :value="DatabaseChangeMode.EDITOR"
              :label="
                $t(
                  'settings.general.workspace.database-change-mode.sql-editor-mode.self'
                )
              "
            />
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
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";

const getInitialState = (): LocalState => {
  const defaultState: LocalState = {
    databaseChangeMode: DatabaseChangeMode.PIPELINE,
  };
  if (
    settingV1Store.workspaceProfileSetting &&
    [DatabaseChangeMode.PIPELINE, DatabaseChangeMode.EDITOR].includes(
      settingV1Store.workspaceProfileSetting.databaseChangeMode
    )
  ) {
    defaultState.databaseChangeMode =
      settingV1Store.workspaceProfileSetting.databaseChangeMode;
  }
  return defaultState;
};

interface LocalState {
  databaseChangeMode: DatabaseChangeMode;
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
      databaseChangeMode: state.databaseChangeMode,
    },
    updateMask: ["value.workspace_profile_setting_value.database_change_mode"],
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.config-updated"),
  });
};
</script>
