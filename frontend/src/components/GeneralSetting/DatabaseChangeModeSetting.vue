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
      <p class="mt-0.5 mb-2 font-medium">
        {{ $t("settings.general.workspace.database-change-mode.description") }}
      </p>
      <div>
        <NRadioGroup v-model:value="state.databaseChangeMode" size="large">
          <NSpace vertical>
            <NRadio key="PIPELINE" :value="DatabaseChangeMode.PIPELINE">
              <div class="flex flex-col gap-1">
                <div class="text-medium">
                  {{
                    $t(
                      "settings.general.workspace.database-change-mode.issue-mode.self"
                    )
                  }}
                </div>
                <div class="textinfolabel">
                  {{
                    $t(
                      "settings.general.workspace.database-change-mode.issue-mode.description"
                    )
                  }}
                </div>
              </div>
            </NRadio>
            <NRadio key="EDITOR" :value="DatabaseChangeMode.EDITOR">
              <div class="flex flex-col gap-1">
                <div class="text-medium">
                  {{
                    $t(
                      "settings.general.workspace.database-change-mode.sql-editor-mode.self"
                    )
                  }}
                </div>
                <div class="textinfolabel">
                  {{
                    $t(
                      "settings.general.workspace.database-change-mode.sql-editor-mode.description"
                    )
                  }}
                </div>
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

    <BBModal
      v-model:show="state.showModal"
      :title="$t('settings.general.workspace.config-updated')"
      container-class="flex flex-col gap-2"
    >
      <div class="py-2">
        The workspace's default view has been changed to SQL Editor.
      </div>
      <div class="flex items-center justify-end gap-2">
        <NButton @click="state.showModal = false">OK</NButton>
        <NButton type="primary" @click="goToSQLEditor">
          Go to SQL Editor
        </NButton>
      </div>
    </BBModal>
  </div>
</template>

<script lang="ts" setup>
import { isEqual } from "lodash-es";
import { NRadioGroup, NSpace, NRadio, NButton } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBModal } from "@/bbkit";
import { router } from "@/router";
import { SQL_EDITOR_HOME_MODULE } from "@/router/sqlEditor";
import { pushNotification } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";

interface LocalState {
  databaseChangeMode: DatabaseChangeMode;
  showModal: boolean;
}

const getInitialState = (): LocalState => {
  const defaultState: LocalState = {
    databaseChangeMode: DatabaseChangeMode.PIPELINE,
    showModal: false,
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
  if (state.databaseChangeMode === DatabaseChangeMode.EDITOR) {
    state.showModal = true;
  }
};

const goToSQLEditor = () => {
  router.push({
    name: SQL_EDITOR_HOME_MODULE,
  });
};
</script>
