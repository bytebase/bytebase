<template>
  <div class="pb-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center gap-x-2">
        <h1 class="text-2xl font-bold">
          {{ title }}
        </h1>
      </div>
    </div>
    <div class="flex-1 mt-4 lg:px-4 lg:mt-0 flex flex-col gap-y-6">
      <WorkspaceMode
        v-model:mode="state.databaseChangeMode"
        :disabled="!allowEdit"
      />

      <div v-if="!isSaaSMode">
        <label class="flex items-center gap-x-2">
          <span class="font-medium">{{
            $t("settings.general.workspace.external-url.self")
          }}</span>
        </label>
        <div class="mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.external-url.description") }}
          <LearnMoreLink
            url="https://docs.bytebase.com/get-started/self-host/external-url?source=console"
          />
        </div>
        <div v-if="externalUrlFromFlag" class="mb-2 text-sm text-accent">
          {{ $t("settings.general.workspace.external-url.managed-by-flag") }}
        </div>
        <NTooltip
          placement="top-start"
          :disabled="allowEdit && !externalUrlFromFlag"
        >
          <template #trigger>
            <NInput
              v-model:value="state.externalUrl"
              class="mb-4 w-full"
              :disabled="!allowEdit || isSaaSMode || externalUrlFromFlag"
            />
          </template>
          <span class="text-sm text-gray-400 -translate-y-2">
            {{
              externalUrlFromFlag
                ? $t("settings.general.workspace.external-url.cannot-edit-flag")
                : $t("settings.general.workspace.only-admin-can-edit")
            }}
          </span>
        </NTooltip>
      </div>
    </div>

    <BBModal
      v-model:show="showModal"
      :title="$t('settings.general.workspace.config-updated')"
      container-class="flex flex-col gap-2"
    >
      <div class="py-2">
        {{
          $t(
            "settings.general.workspace.default-landing-page.default-view-changed-to-sql-editor"
          )
        }}
      </div>
      <div class="flex items-center justify-end gap-2">
        <NButton @click="showModal = false">
          {{ $t("common.ok") }}
        </NButton>
        <NButton type="primary" @click="goToSQLEditor">
          {{
            $t(
              "settings.general.workspace.default-landing-page.go-to-sql-editor"
            )
          }}
        </NButton>
      </div>
    </BBModal>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import { NButton, NInput, NTooltip } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, reactive, ref } from "vue";
import { BBModal } from "@/bbkit";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { router } from "@/router";
import { SQL_EDITOR_HOME_MODULE } from "@/router/sqlEditor";
import { useActuatorV1Store, useSettingV1Store } from "@/store";
import { DatabaseChangeMode } from "@/types/proto-es/v1/setting_service_pb";
import WorkspaceMode from "@/views/Setup/WorkspaceMode.vue";

interface LocalState {
  databaseChangeMode: DatabaseChangeMode;
  externalUrl: string;
}

const getInitialState = (): LocalState => {
  const defaultState: LocalState = {
    databaseChangeMode: DatabaseChangeMode.PIPELINE,
    externalUrl: settingV1Store.workspaceProfileSetting?.externalUrl ?? "",
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
const actuatorV1Store = useActuatorV1Store();
const { isSaaSMode } = storeToRefs(actuatorV1Store);

const state = reactive<LocalState>(getInitialState());
const showModal = ref(false);

const externalUrlFromFlag = computed(() => {
  return actuatorV1Store.serverInfo?.externalUrlFromFlag ?? false;
});

const allowSave = computed((): boolean => {
  return !isEqual(state, getInitialState());
});

const onUpdate = async () => {
  const initState = getInitialState();
  if (state.externalUrl !== initState.externalUrl) {
    await settingV1Store.updateWorkspaceProfile({
      payload: {
        externalUrl: state.externalUrl,
      },
      updateMask: create(FieldMaskSchema, {
        paths: ["value.workspace_profile.external_url"],
      }),
    });
  }
  if (state.databaseChangeMode !== initState.databaseChangeMode) {
    await settingV1Store.updateWorkspaceProfile({
      payload: {
        databaseChangeMode: state.databaseChangeMode,
      },
      updateMask: create(FieldMaskSchema, {
        paths: ["value.workspace_profile.database_change_mode"],
      }),
    });
    if (state.databaseChangeMode === DatabaseChangeMode.EDITOR) {
      showModal.value = true;
    }
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
