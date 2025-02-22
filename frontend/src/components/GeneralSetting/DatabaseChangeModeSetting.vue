<template>
  <div ref="containerRef" class="pb-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center space-x-2">
        <h1 class="text-2xl font-bold">
          {{ title }}
        </h1>
      </div>
    </div>
    <div class="flex-1 mt-4 lg:px-4 lg:mt-0">
      <p class="mt-0.5 mb-2 font-medium">
        {{ $t("settings.general.workspace.database-change-mode.description") }}
        <LearnMoreLink
          url="https://www.bytebase.com/docs/administration/mode?source=console"
          class="ml-1 text-sm"
        />
      </p>
      <div>
        <NRadioGroup
          v-model:value="state.databaseChangeMode"
          :disabled="!allowEdit"
          size="large"
        >
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
    </div>

    <BBModal
      v-model:show="state.showModal"
      :title="$t('settings.general.workspace.config-updated')"
      container-class="flex flex-col gap-2"
    >
      <div class="py-2">
        {{
          $t(
            "settings.general.workspace.database-change-mode.default-view-changed-to-sql-editor"
          )
        }}
      </div>
      <div class="flex items-center justify-end gap-2">
        <NButton @click="state.showModal = false">
          {{ $t("common.ok") }}
        </NButton>
        <NButton type="primary" @click="goToSQLEditor">
          {{
            $t(
              "settings.general.workspace.database-change-mode.go-to-sql-editor"
            )
          }}
        </NButton>
      </div>
    </BBModal>
  </div>
</template>

<script lang="ts" setup>
import { isEqual } from "lodash-es";
import { NRadioGroup, NSpace, NRadio, NButton } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { BBModal } from "@/bbkit";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { router } from "@/router";
import { WORKSPACE_ROOT_MODULE } from "@/router/dashboard/workspaceRoutes";
import { SQL_EDITOR_HOME_MODULE } from "@/router/sqlEditor";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";
import { isSQLEditorRoute } from "@/utils";

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

const props = defineProps<{
  title: string;
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const containerRef = ref<HTMLDivElement>();

const state = reactive<LocalState>(getInitialState());

const allowSave = computed((): boolean => {
  return !isEqual(state, getInitialState());
});

const onUpdate = async () => {
  await settingV1Store.updateWorkspaceProfile({
    payload: {
      databaseChangeMode: state.databaseChangeMode,
    },
    updateMask: ["value.workspace_profile_setting_value.database_change_mode"],
  });
  if (
    state.databaseChangeMode === DatabaseChangeMode.EDITOR &&
    !isSQLEditorRoute(router)
  ) {
    state.showModal = true;
  }
  if (
    isSQLEditorRoute(router) &&
    state.databaseChangeMode === DatabaseChangeMode.PIPELINE
  ) {
    router.push({ name: WORKSPACE_ROOT_MODULE });
  }
};

const goToSQLEditor = () => {
  router.push({
    name: SQL_EDITOR_HOME_MODULE,
  });
};

defineExpose({
  isDirty: allowSave,
  update: onUpdate,
  title: props.title,
});
</script>
