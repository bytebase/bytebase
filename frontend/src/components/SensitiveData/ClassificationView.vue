<template>
  <div class="w-full space-y-4">
    <div>
      <div class="flex items-center space-x-2">
        <NSwitch
          :value="!state.classification.classificationFromConfig"
          :disabled="!allowEdit || !hasSensitiveDataFeature"
          @update:value="onClassificationConfigChange"
        />
        <div class="font-medium leading-7 text-main">
          {{ $t("database.classification.sync-from-comment") }}
        </div>
      </div>
      <i18n-t
        class="textinfolabel"
        tag="div"
        keypath="database.classification.sync-from-comment-tip"
      >
        <template #format>
          <span class="font-semibold">{classification id}-{comment}</span>
        </template>
      </i18n-t>
    </div>

    <NDivider class="my-2" />

    <div class="flex items-center justify-between">
      <div class="textinfolabel">
        {{ $t("settings.sensitive-data.classification.upload-label") }}
        <span
          class="normal-link cursor-pointer hover:underline"
          @click="state.showExampleModal = true"
        >
          {{ $t("settings.sensitive-data.view-example") }}
        </span>
      </div>

      <div class="flex items-center justify-end gap-2">
        <NButton
          :disabled="!allowEdit || !hasSensitiveDataFeature"
          @click="onUpload"
        >
          {{ $t("settings.sensitive-data.classification.upload") }}
        </NButton>
        <NButton
          type="primary"
          :disabled="!allowEdit || !hasSensitiveDataFeature || !allowSave"
          @click="upsertSetting"
        >
          {{ $t("common.save") }}
        </NButton>
        <input
          ref="uploader"
          type="file"
          accept=".json"
          class="sr-only hidden"
          :disabled="!allowEdit || !hasSensitiveDataFeature"
          @input="onFileChange"
        />
      </div>
    </div>

    <div
      v-if="Object.keys(state.classification.classification).length === 0"
      class="flex justify-center border-2 border-gray-300 border-dashed rounded-md relative h-72"
    >
      <SingleFileSelector
        class="space-y-1 text-center flex flex-col justify-center items-center absolute top-0 bottom-0 left-0 right-0"
        :support-file-extensions="['.json']"
        :max-file-size-in-mi-b="maxFileSizeInMiB"
        :disabled="!allowEdit || !hasSensitiveDataFeature"
        @on-select="onFileSelect"
      >
        <template #image>
          <NoDataPlaceholder
            :border="false"
            :img-attrs="{ class: '!max-h-[10vh]' }"
          />
        </template>
      </SingleFileSelector>
    </div>
    <div v-else class="h-full">
      <ClassificationTree :classification-config="state.classification" />
    </div>
  </div>

  <DataExampleModal
    v-if="state.showExampleModal"
    :example="JSON.stringify(example, null, 2)"
    @dismiss="state.showExampleModal = false"
  />
</template>

<script lang="ts" setup>
import { head, isEqual } from "lodash-es";
import { NSwitch, useDialog, NDivider } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  featureToRef,
  useCurrentUserV1,
  useSettingV1Store,
  pushNotification,
} from "@/store";
import { PresetRoleType } from "@/types";
import type {
  DataClassificationSetting_DataClassificationConfig_Level as ClassificationLevel,
  DataClassificationSetting_DataClassificationConfig_DataClassification as DataClassification,
} from "@/types/proto/v1/setting_service";
import { DataClassificationSetting_DataClassificationConfig } from "@/types/proto/v1/setting_service";

const uploader = ref<HTMLInputElement | null>(null);
const maxFileSizeInMiB = 10;

interface UploadClassificationConfig {
  title: string;
  levels: ClassificationLevel[];
  classifications: DataClassification[];
}

interface LocalState {
  classification: DataClassificationSetting_DataClassificationConfig;
  showExampleModal: boolean;
}

const { t } = useI18n();
const $dialog = useDialog();
const settingStore = useSettingV1Store();
const currentUser = useCurrentUserV1();
const state = reactive<LocalState>({
  showExampleModal: false,
  classification:
    DataClassificationSetting_DataClassificationConfig.fromPartial({
      id: uuidv4(),
      ...head(settingStore.classification),
    }),
});

const allowSave = computed(() => {
  return (
    allowEdit.value &&
    hasSensitiveDataFeature.value &&
    !isEqual(head(settingStore.classification), state.classification)
  );
});

const onClassificationConfigChange = (fromComment: boolean) => {
  $dialog.warning({
    title: t("common.warning"),
    content: fromComment
      ? t("database.classification.sync-from-comment-enable-warning")
      : t("database.classification.sync-from-comment-disable-warning"),
    style: "z-index: 100000",
    negativeText: t("common.cancel"),
    positiveText: t("common.confirm"),
    onPositiveClick: async () => {
      state.classification.classificationFromConfig = !fromComment;
      await upsertSetting();
    },
  });
};

const upsertSetting = async () => {
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
    title: t("common.updated"),
  });
};

const allowEdit = computed(() => {
  // Only allow workspace admin to manage user.
  return currentUser.value.roles.includes(PresetRoleType.WORKSPACE_ADMIN);
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
  onFileSelect(file);
};

const onFileSelect = (file: File) => {
  const fr = new FileReader();
  fr.onload = () => {
    if (!fr.result) {
      return;
    }
    const data: UploadClassificationConfig = JSON.parse(fr.result as string);
    if (
      !Array.isArray(data.classifications) ||
      data.classifications.length === 0
    ) {
      return pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Data format error",
        description: "Should has classifications array field",
      });
    }
    if (!Array.isArray(data.levels) || data.levels.length === 0) {
      return pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Data format error",
        description: "Should has levels array field",
      });
    }
    if (
      data.classifications.length !==
      new Set(data.classifications.map((item) => item.id)).size
    ) {
      return pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Data format error",
        description: "Should not contains duplicate classification id",
      });
    }
    Object.assign(state.classification, {
      title: data.title || state.classification.title || "",
      levels: data.levels,
      classification: data.classifications.reduce(
        (obj, item) => {
          obj[item.id] = item;
          return obj;
        },
        {} as { [key: string]: DataClassification }
      ),
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

const example: UploadClassificationConfig = {
  title: "Classification Example",
  levels: [
    {
      id: "1",
      title: "Level 1",
      description: "",
    },
    {
      id: "2",
      title: "Level 2",
      description: "",
    },
  ],
  classifications: [
    {
      id: "1",
      title: "Basic",
      description: "",
    },
    {
      id: "1-1",
      title: "Basic",
      description: "",
      levelId: "1",
    },
    {
      id: "1-2",
      title: "Assert",
      description: "",
      levelId: "1",
    },
    {
      id: "1-3",
      title: "Contact",
      description: "",
      levelId: "2",
    },
    {
      id: "1-4",
      title: "Health",
      description: "",
      levelId: "2",
    },
    {
      id: "2",
      title: "Relationship",
      description: "",
    },
    {
      id: "2-1",
      title: "Social",
      description: "",
      levelId: "1",
    },
    {
      id: "2-2",
      title: "Business",
      description: "",
      levelId: "1",
    },
  ],
};
</script>
