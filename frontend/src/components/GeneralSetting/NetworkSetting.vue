<template>
  <div class="lg:flex">
    <div class="text-left lg:w-1/4">
      <h1 class="text-2xl font-bold">
        {{ $t("settings.general.workspace.network") }}
      </h1>
      <span v-if="!allowEdit" class="text-sm text-gray-400">
        {{ $t("settings.general.workspace.only-admin-can-edit") }}
      </span>
    </div>
    <div class="flex-1 lg:px-4">
      <div class="mb-7 mt-4 lg:mt-0">
        <label
          class="flex items-center gap-x-2"
          :class="[allowEdit ? 'cursor-pointer' : 'cursor-not-allowed']"
        >
          <span class="font-medium">{{
            $t("settings.general.workspace.external-url.self")
          }}</span>
        </label>
        <div class="mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.external-url.description") }}
          <LearnMoreLink
            url="https://www.bytebase.com/docs/get-started/install/external-url?source=console"
          />
        </div>
        <NTooltip placement="top-start" :disabled="allowEdit">
          <template #trigger>
            <NInput
              v-model:value="state.externalUrl"
              class="mb-4 w-full"
              :disabled="!allowEdit"
            />
          </template>
          <span class="text-sm text-gray-400 -translate-y-2">
            {{ $t("settings.general.workspace.only-admin-can-edit") }}
          </span>
        </NTooltip>

        <label
          class="flex items-center gap-x-2"
          :class="[allowEdit ? 'cursor-pointer' : 'cursor-not-allowed']"
        >
          <span class="font-medium">
            {{ $t("settings.general.workspace.gitops-webhook-url.self") }}
          </span>
        </label>
        <div class="mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.gitops-webhook-url.description") }}
          <LearnMoreLink
            url="https://www.bytebase.com/docs/get-started/install/external-url#gitops-webhook-url?source=console"
          />
        </div>
        <NTooltip placement="top-start" :disabled="allowEdit">
          <template #trigger>
            <NInput
              v-model:value="state.gitopsWebhookUrl"
              class="mb-4 w-full"
              :placeholder="
                t(
                  'settings.general.workspace.gitops-webhook-url.default-to-external-url'
                )
              "
              :disabled="!allowEdit"
            />
          </template>
          <span class="text-sm text-gray-400 -translate-y-2">
            {{ $t("settings.general.workspace.only-admin-can-edit") }}
          </span>
        </NTooltip>

        <div class="flex justify-end">
          <NButton
            type="primary"
            :disabled="!allowSave"
            @click.prevent="updateNetworkSetting"
          >
            {{ $t("common.update") }}
          </NButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useCurrentUserV1 } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { hasWorkspacePermissionV1 } from "@/utils";

interface LocalState {
  externalUrl: string;
  gitopsWebhookUrl: string;
}

const { t } = useI18n();
const settingV1Store = useSettingV1Store();
const currentUserV1 = useCurrentUserV1();

const state = reactive<LocalState>({
  externalUrl: "",
  gitopsWebhookUrl: "",
});

watchEffect(() => {
  state.externalUrl = settingV1Store.workspaceProfileSetting?.externalUrl ?? "";
  state.gitopsWebhookUrl =
    settingV1Store.workspaceProfileSetting?.gitopsWebhookUrl ?? "";
});

const allowEdit = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-general",
    currentUserV1.value.userRole
  );
});

const allowSave = computed((): boolean => {
  if (!allowEdit.value) {
    return false;
  }

  const externalUrlChanged =
    state.externalUrl !== "" &&
    state.externalUrl !== settingV1Store.workspaceProfileSetting?.externalUrl;
  const gitopsWebhookUrlChanged =
    state.gitopsWebhookUrl !== "" &&
    state.gitopsWebhookUrl !==
      settingV1Store.workspaceProfileSetting?.gitopsWebhookUrl;
  return externalUrlChanged || gitopsWebhookUrlChanged;
});

const updateNetworkSetting = async () => {
  if (!allowSave.value) {
    return;
  }
  await settingV1Store.updateWorkspaceProfile({
    externalUrl: state.externalUrl,
    gitopsWebhookUrl: state.gitopsWebhookUrl,
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.config-updated"),
  });

  state.externalUrl = settingV1Store.workspaceProfileSetting?.externalUrl ?? "";
  state.gitopsWebhookUrl =
    settingV1Store.workspaceProfileSetting?.gitopsWebhookUrl ?? "";
};
</script>
