<template>
  <div class="w-full space-y-4">
    <div class="flex items-center justify-end">
      <NButton
        type="primary"
        :disabled="!hasPermission || !hasSensitiveDataFeature"
        @click="onUpload"
      >
        {{ $t("settings.sensitive-data.classification.upload") }}
      </NButton>
      <input
        ref="uploader"
        type="file"
        accept=".json"
        class="sr-only hidden"
        :disabled="!hasPermission || !hasSensitiveDataFeature"
        @input="onFileChange"
      />
    </div>
    <div class="textinfolabel">
      {{ $t("settings.sensitive-data.classification.label") }}
      <span
        class="normal-link cursor-pointer hover:underline"
        @click="state.showExampleModal = true"
      >
        {{ $t("settings.sensitive-data.classification.view-example") }}
      </span>
    </div>
    <NoDataPlaceholder v-if="settingStore.classification.length === 0" />
    <div v-else class="h-full">
      <ClassificationTree
        :classification-config="settingStore.classification[0]"
      />
    </div>
  </div>

  <ClassificationExampleModal
    v-if="state.showExampleModal"
    @dismiss="state.showExampleModal = false"
  />

  <BBAlert
    v-if="state.showOverrideModal"
    :style="'WARN'"
    :ok-text="$t('settings.sensitive-data.classification.override-confirm')"
    :title="$t('settings.sensitive-data.classification.override-title')"
    :description="$t('settings.sensitive-data.classification.override-desc')"
    @ok="upsertSetting"
    @cancel="state.showOverrideModal = false"
  />
</template>

<script lang="ts" setup>
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  featureToRef,
  useCurrentUserV1,
  useSettingV1Store,
  pushNotification,
} from "@/store";
import { DataClassificationSetting_DataClassificationConfig } from "@/types/proto/v1/setting_service";
import { hasWorkspacePermissionV1 } from "@/utils";

const uploader = ref<HTMLInputElement | null>(null);

interface LocalState {
  classification?: DataClassificationSetting_DataClassificationConfig;
  showExampleModal: boolean;
  showOverrideModal: boolean;
}

const state = reactive<LocalState>({
  showExampleModal: false,
  showOverrideModal: false,
});
const { t } = useI18n();
const settingStore = useSettingV1Store();
const currentUserV1 = useCurrentUserV1();

watch(
  () => state.classification,
  async () => {
    if (settingStore.classification.length !== 0) {
      state.showOverrideModal = true;
      return;
    }

    await upsertSetting();
  }
);

const upsertSetting = async () => {
  if (!state.classification) {
    return;
  }
  await settingStore.upsertSetting({
    name: "bb.workspace.data-classification",
    value: {
      dataClassificationSettingValue: {
        configs: [state.classification],
      },
    },
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.sensitive-data.classification.upload-succeed"),
  });
  state.showOverrideModal = false;
};

const hasPermission = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sensitive-data",
    currentUserV1.value.userRole
  );
});
const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");

const onUpload = () => {
  uploader.value?.click();
};

const onFileChange = () => {
  const files: File[] = (uploader.value as any).files;
  if (files.length !== 1) {
    return;
  }
  const file = files[0];

  const fr = new FileReader();
  fr.onload = () => {
    if (!fr.result) {
      return;
    }
    state.classification =
      DataClassificationSetting_DataClassificationConfig.fromPartial({
        ...JSON.parse(fr.result as string),
        id: uuidv4(),
      });
  };
  fr.onerror = () => {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Read file error`,
      description: String(fr.error),
    });
    return;
  };
  fr.readAsText(file);
};
</script>
