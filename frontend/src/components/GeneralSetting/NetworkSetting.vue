<template>
  <div class="px-4 py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <h1 class="text-2xl font-bold">
        {{ $t("settings.general.workspace.network") }}
      </h1>
      <span v-if="!allowEdit" class="text-sm text-gray-400">
        {{ $t("settings.general.workspace.only-owner-can-edit") }}
      </span>
    </div>
    <div class="flex-1 lg:px-5">
      <div class="mb-7 mt-5 lg:mt-0">
        <label
          class="flex items-center gap-x-2 tooltip-wrapper"
          :class="[allowEdit ? 'cursor-pointer' : 'cursor-not-allowed']"
        >
          <span class="font-medium">{{
            $t("settings.general.workspace.external-url.self")
          }}</span>

          <span
            v-if="!allowEdit"
            class="text-sm text-gray-400 -translate-y-2 tooltip"
          >
            {{ $t("settings.general.workspace.only-owner-can-edit") }}
          </span>
        </label>
        <div class="mb-3 text-sm text-gray-400">
          <i18n-t keypath="settings.general.workspace.external-url.description">
            <template #viewDoc>
              <a
                href="https://www.bytebase.com/docs/get-started/install/external-url"
                class="normal-link"
                target="_blank"
                >{{ $t("common.view-doc") }}</a
              >
            </template>
          </i18n-t>
        </div>
        <BBTextField
          class="mb-5 w-full"
          :disabled="!allowEdit"
          :value="state.externalUrl"
          @input="handleExternalUrlChange"
        />

        <label
          class="flex items-center gap-x-2 tooltip-wrapper"
          :class="[allowEdit ? 'cursor-pointer' : 'cursor-not-allowed']"
        >
          <span class="font-medium">{{
            $t("settings.general.workspace.gitops-webhook-url.self")
          }}</span>

          <span
            v-if="!allowEdit"
            class="text-sm text-gray-400 -translate-y-2 tooltip"
          >
            {{ $t("settings.general.workspace.only-owner-can-edit") }}
          </span>
        </label>
        <div class="mb-3 text-sm text-gray-400">
          <i18n-t
            keypath="settings.general.workspace.gitops-webhook-url.description"
          >
          </i18n-t>
        </div>
        <BBTextField
          class="mb-5 w-full"
          :disabled="!allowEdit"
          :value="state.gitopsWebhookUrl"
          @input="handleGitOpsWebhookUrlChange"
        />

        <div class="flex">
          <button
            type="button"
            class="btn-primary ml-auto"
            :disabled="!allowSave"
            @click.prevent="updateNetworkSetting"
          >
            {{ $t("common.update") }}
          </button>
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

const handleExternalUrlChange = (event: InputEvent) => {
  state.externalUrl = (event.target as HTMLInputElement).value;
};

const handleGitOpsWebhookUrlChange = (event: InputEvent) => {
  state.gitopsWebhookUrl = (event.target as HTMLInputElement).value;
};

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
