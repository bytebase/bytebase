<template>
  <div class="w-full space-y-4">
    <div class="textinfolabel">
      {{ $t("settings.sensitive-data.algorithms.upload-label") }}
      <span
        class="normal-link cursor-pointer hover:underline"
        @click="state.showExampleModal = true"
      >
        {{ $t("settings.sensitive-data.view-example") }}
      </span>
    </div>
    <div class="flex items-center justify-end space-x-2">
      <NButton
        :loading="state.processing"
        :disabled="!hasPermission || !hasSensitiveDataFeature"
        @click="onUpload"
      >
        <template #icon>
          <ImportIcon class="h-4 w-4" />
        </template>
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
        <template #icon>
          <PlusIcon class="h-4 w-4" />
        </template>
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
  <DataExampleModal
    v-if="state.showExampleModal"
    :example="JSON.stringify(example, null, 2)"
    @dismiss="state.showExampleModal = false"
  />
</template>

<script lang="ts" setup>
import { PlusIcon, ImportIcon } from "lucide-vue-next";
import { NButton, useDialog } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { featureToRef, useSettingV1Store, pushNotification } from "@/store";
import {
  MaskingAlgorithmSetting_Algorithm as Algorithm,
  MaskingAlgorithmSetting_Algorithm_InnerOuterMask_MaskType,
} from "@/types/proto/v1/setting_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import DataExampleModal from "./components/DataExampleModal.vue";
import MaskingAlgorithmsCreateDrawer from "./components/MaskingAlgorithmsCreateDrawer.vue";
import MaskingAlgorithmsTable from "./components/MaskingAlgorithmsTable.vue";

const uploader = ref<HTMLInputElement | null>(null);
const maxFileSizeInMiB = 10;

interface LocalState {
  showCreateDrawer: boolean;
  pendingEditData: Algorithm;
  processing: boolean;
  showExampleModal: boolean;
}

const state = reactive<LocalState>({
  showCreateDrawer: false,
  pendingEditData: Algorithm.fromPartial({
    id: uuidv4(),
  }),
  processing: false,
  showExampleModal: false,
});

const { t } = useI18n();
const settingStore = useSettingV1Store();
const $dialog = useDialog();

const hasPermission = computed(() => {
  return hasWorkspacePermissionV2("bb.policies.update");
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

const example: Algorithm[] = [
  {
    id: "5d625aee-2628-4515-a4b6-6c494499a829",
    title: "Full mask",
    description: "Use substitution to replace the full data.",
    category: "MASK",
    fullMask: {
      substitution: "****",
    },
  },
  {
    id: "240ea8b6-0dd1-409f-abd3-380a35e4f52b",
    title: "Range mask",
    description: "Use substitution to replace the range data.",
    category: "MASK",
    rangeMask: {
      slices: [
        {
          start: 1,
          end: 2,
          substitution: "****",
        },
      ],
    },
  },
  {
    id: "d530cb90-45d6-4916-99ef-c6dde5b88651",
    title: "MD5 hash",
    description: "Hasing the full data with the salt.",
    category: "HASH",
    md5Mask: {
      salt: "the hash salt",
    },
  },
  {
    id: "d8faf746-7d4a-4291-8e66-dbe547a7f7fc",
    title: "Inner mask",
    description:
      "Masking the interior of its string argument, leaving the ends unmasked. Other arguments specify the sizes of the unmasked ends.",
    category: "MASK",
    innerOuterMask: {
      prefixLen: 1,
      suffixLen: 2,
      substitution: "***",
      type: MaskingAlgorithmSetting_Algorithm_InnerOuterMask_MaskType.INNER,
    },
  },
  {
    id: "112f49cf-1347-4834-ab8a-fe7db5ed1dba",
    title: "Outer mask",
    description:
      "Masking the ends of its string argument, leaving the interior unmasked. Other arguments specify the sizes of the masked ends.",
    category: "MASK",
    innerOuterMask: {
      prefixLen: 1,
      suffixLen: 2,
      substitution: "***",
      type: MaskingAlgorithmSetting_Algorithm_InnerOuterMask_MaskType.OUTER,
    },
  },
];
</script>
