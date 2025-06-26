<template>
  <div class="pb-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center space-x-2">
        <h1 class="text-2xl font-bold">
          {{ title }}
        </h1>
      </div>
    </div>
    <div class="flex-1 mt-4 lg:px-4 lg:mt-0">
      <WorkspaceMode
        v-model:mode="state.databaseChangeMode"
        :disabled="!allowEdit"
      />
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
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { BBModal } from "@/bbkit";
import { router } from "@/router";
import { WORKSPACE_ROUTE_LANDING } from "@/router/dashboard/workspaceRoutes";
import { SQL_EDITOR_HOME_MODULE } from "@/router/sqlEditor";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { DatabaseChangeMode } from "@/types/proto-es/v1/setting_service_pb";
import { isSQLEditorRoute } from "@/utils";
import WorkspaceMode from "@/views/Setup/WorkspaceMode.vue";

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

const state = reactive<LocalState>(getInitialState());

const allowSave = computed((): boolean => {
  return state.databaseChangeMode !== getInitialState().databaseChangeMode;
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
    router.replace({ name: WORKSPACE_ROUTE_LANDING });
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
  revert: () => {
    Object.assign(state, getInitialState());
  },
});
</script>
