<template>
  <div class="mx-auto w-full max-w-sm flex flex-col gap-8">
    <div class="flex flex-col items-center gap-8">
      <BytebaseLogo />

      <h2 class="text-2xl leading-9 font-medium text-accent">
        {{ $t("setup.workspace-mode.choose") }}
      </h2>
    </div>

    <NRadioGroup v-model:value="state.mode">
      <NSpace vertical size="large">
        <NRadio :value="DatabaseChangeMode.PIPELINE">
          <div class="flex flex-col gap-1">
            <div class="font-medium">
              {{ $t("setup.workspace-mode.issue-mode.self") }}
            </div>
            <div>
              {{ $t("setup.workspace-mode.issue-mode.description") }}
            </div>
          </div>
        </NRadio>
        <NRadio :value="DatabaseChangeMode.EDITOR">
          <div class="flex flex-col gap-1">
            <div class="font-medium">
              {{ $t("setup.workspace-mode.sql-editor-mode.self") }}
            </div>
            <div>
              {{ $t("setup.workspace-mode.sql-editor-mode.description") }}
            </div>
          </div>
        </NRadio>

        <div class="text-control-placeholder text-sm">
          {{ $t("setup.workspace-mode.can-be-changed-later") }}
        </div>
      </NSpace>
    </NRadioGroup>

    <div class="w-full">
      <NButton
        type="primary"
        size="large"
        style="width: 100%"
        :loading="state.isLoading"
        @click="handleFinish"
      >
        {{ $t("common.finish") }}
      </NButton>
    </div>
  </div>

  <AuthFooter />
</template>

<script lang="ts" setup>
import { NButton, NRadio, NRadioGroup, NSpace } from "naive-ui";
import { reactive } from "vue";
import { useRouter } from "vue-router";
import BytebaseLogo from "@/components/BytebaseLogo.vue";
import { useSettingV1Store } from "@/store";
import { DatabaseChangeMode } from "@/types/proto/v1/setting_service";
import AuthFooter from "@/views/auth/AuthFooter.vue";

const router = useRouter();

const settingStore = useSettingV1Store();
const getInitialMode = () => {
  if (
    settingStore.workspaceProfileSetting &&
    [DatabaseChangeMode.PIPELINE, DatabaseChangeMode.EDITOR].includes(
      settingStore.workspaceProfileSetting.databaseChangeMode
    )
  ) {
    return settingStore.workspaceProfileSetting.databaseChangeMode;
  }
  return DatabaseChangeMode.PIPELINE;
};

const state = reactive({
  isLoading: false,
  mode: getInitialMode(),
});

const handleFinish = async () => {
  try {
    state.isLoading = true;
    await settingStore.updateWorkspaceProfile({
      payload: {
        databaseChangeMode: state.mode,
      },
      updateMask: [
        "value.workspace_profile_setting_value.database_change_mode",
      ],
    });
    router.replace("/");
  } finally {
    state.isLoading = false;
  }
};
</script>
