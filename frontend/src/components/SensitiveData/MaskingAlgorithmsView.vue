<template>
  <div class="w-full space-y-4">
    <div class="flex items-center justify-end space-x-2">
      <NButton
        :loading="state.processing"
        :disabled="!hasPermission || !hasSensitiveDataFeature"
        @click="onUpload"
      >
        {{ $t("settings.sensitive-data.algorithms.upload") }}
        <input
          ref="uploader"
          type="file"
          accept=".json"
          class="sr-only hidden"
          :disabled="!hasPermission || !hasSensitiveDataFeature"
          @input="onFileChange"
        />
      </NButton>
      <NButton
        type="primary"
        :disabled="!hasPermission || !hasSensitiveDataFeature"
        @click="onCreate"
      >
        {{ $t("common.add") }}
      </NButton>
    </div>
    <div class="space-y-5 divide-y-2 pb-10 divide-gray-100">
      <MaskingAlgorithmsTable
        :readonly="!hasPermission || !hasSensitiveDataFeature"
        :row-clickable="false"
        @edit="onEdit"
      />
    </div>
  </div>
  <MaskingAlgorithmsCreateDrawer
    :show="state.showCreateDrawer"
    :algorithm="state.pendingEditData"
    :readonly="!hasPermission || !hasSensitiveDataFeature"
    @dismiss="onDrawerDismiss"
  />
</template>

<script lang="ts" setup>
import { NButton, useDialog } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  featureToRef,
  useCurrentUserV1,
  useSettingV1Store,
  pushNotification,
} from "@/store";
import { MaskingAlgorithmSetting_Algorithm as Algorithm } from "@/types/proto/v1/setting_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import MaskingAlgorithmsCreateDrawer from "./components/MaskingAlgorithmsCreateDrawer.vue";
import MaskingAlgorithmsTable from "./components/MaskingAlgorithmsTable.vue";

const uploader = ref<HTMLInputElement | null>(null);
const maxFileSizeInMiB = 10;

interface LocalState {
  showCreateDrawer: boolean;
  pendingEditData: Algorithm;
  processing: boolean;
}

const state = reactive<LocalState>({
  showCreateDrawer: false,
  pendingEditData: Algorithm.fromPartial({
    id: uuidv4(),
  }),
  processing: false,
});

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const settingStore = useSettingV1Store();
const $dialog = useDialog();

const hasPermission = computed(() => {
  return hasWorkspacePermissionV2(currentUser.value, "bb.policies.update");
});
const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");

const rawAlgorithmList = computed((): Algorithm[] => {
  return (
    settingStore.getSettingByName("bb.workspace.masking-algorithm")?.value
      ?.maskingAlgorithmSettingValue?.algorithms ?? []
  );
});

const onCreate = () => {
  state.pendingEditData = Algorithm.fromPartial({
    id: uuidv4(),
  });
  state.showCreateDrawer = true;
};

const onDrawerDismiss = () => {
  state.showCreateDrawer = false;
};

const onEdit = (data: Algorithm) => {
  state.pendingEditData = data;
  state.showCreateDrawer = true;
};

const onUpload = () => {
  uploader.value?.click();
};

const onFileChange = () => {
  const files: File[] = (uploader.value as any).files;
  if (files.length !== 1) {
    return;
  }
  const file = files[0];
  if (file.size > maxFileSizeInMiB * 1024 * 1024) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.file-selector.size-limit", {
        size: maxFileSizeInMiB,
      }),
    });
    return;
  }

  if (rawAlgorithmList.value.length > 0) {
    $dialog.warning({
      title: t("settings.sensitive-data.algorithms.override-title"),
      content: t("settings.sensitive-data.algorithms.override-desc"),
      style: "z-index: 100000",
      negativeText: t("common.cancel"),
      positiveText: t("settings.sensitive-data.algorithms.override-confirm"),
      onPositiveClick: () => {
        onFileSelect(file);
      },
    });
  } else {
    onFileSelect(file);
  }
};

const onFileSelect = async (file: File) => {
  const fr = new FileReader();
  fr.onload = () => {
    if (!fr.result) {
      return;
    }
    const data: Algorithm[] = JSON.parse(fr.result as string);

    settingStore
      .upsertSetting({
        name: "bb.workspace.masking-algorithm",
        value: {
          maskingAlgorithmSettingValue: {
            algorithms: data.map((algorithm) =>
              Algorithm.fromPartial({
                ...algorithm,
                id: algorithm.id || uuidv4(),
              })
            ),
          },
        },
      })
      .then(() => {
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("settings.sensitive-data.classification.upload-succeed"),
        });
      })
      .finally(() => {
        state.processing = false;
      });
  };

  fr.onerror = () => {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Read file error",
      description: String(fr.error),
    });
    return;
  };
  fr.readAsText(file);
};
</script>
