<template>
  <NDropdown
    trigger="hover"
    :options="exportDropdownOptions"
    :disabled="viewMode === 'DRAWER' || disabled"
    @select="tryExportViaDropdown"
  >
    <NTooltip :disabled="!tooltip">
      <template #trigger>
        <NButton
          :quaternary="size === 'tiny'"
          :size="size"
          type="primary"
          v-bind="$attrs"
          :loading="state.isRequesting"
          :disabled="state.isRequesting || disabled"
          @click="handleClickExportButton"
        >
          <template #icon>
            <DownloadIcon class="h-4 w-4" />
          </template>
          <span v-if="size !== 'tiny'">
            {{ text }}
          </span>
        </NButton>
      </template>
      <span class="text-sm"> {{ tooltip }} </span>
    </NTooltip>
  </NDropdown>

  <Drawer v-if="viewMode === 'DRAWER'" v-model:show="state.showDrawer">
    <DrawerContent
      :title="$t('custom-approval.risk-rule.risk.namespace.data_export')"
      class="w-200 max-w-[100vw] relative"
    >
      <template #default>
        <NForm
          ref="formRef"
          :model="formData"
          :rules="rules"
          label-placement="left"
        >
          <slot name="form" />
          <NFormItem path="limit" :label="$t('export-data.export-rows')">
            <MaxRowCountSelect
              ref="maxRowCountSelectRef"
              :maximum-export-count="maximumExportCount"
              v-model:value="formData.limit"
            />
          </NFormItem>
          <NFormItem path="format" :label="$t('export-data.export-format')">
            <NRadioGroup v-model:value="formData.format">
              <NRadio
                v-for="format in supportFormats"
                :key="format"
                :value="format"
              >
                {{ ExportFormat[format] }}
              </NRadio>
            </NRadioGroup>
          </NFormItem>
          <NFormItem
            v-if="supportPassword"
            path="password"
            :label="$t('export-data.password-optional')"
          >
            <BBTextField
              v-model:value="formData.password"
              type="password"
              :input-props="{ autocomplete: 'new-password' }"
            />
          </NFormItem>
        </NForm>
      </template>
      <template #footer>
        <div class="flex flex-row items-center justify-end gap-x-3">
          <NButton @click="state.showDrawer = false">
            {{ $t("common.cancel") }}
          </NButton>
          <ErrorTipsButton
            :button-props="{
              type: 'primary',
              loading: state.isRequesting,
              disabled:
                formErrors.length > 0 ||
                state.isRequesting ||
                !validate(formData),
            }"
            :errors="formErrors"
            @click="tryExportViaForm"
          >
            {{ $t("common.confirm") }}
          </ErrorTipsButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>

  <BBModal
    :show="state.showModal"
    :title="$t('export-data.password-optional')"
    class="shadow-inner outline-solid outline-gray-200"
    @close="state.showModal = false"
  >
    <div class="w-80">
      <span class="textinfolabel">{{ $t("export-data.password-info") }}</span>
      <BBTextField
        v-model:value="formData.password"
        class="my-2"
        :focus-on-mount="true"
      />
    </div>
    <div class="w-full flex items-center justify-end mt-2 gap-x-3">
      <NButton @click="state.showModal = false">
        {{ $t("common.cancel") }}
      </NButton>
      <NButton type="primary" @click="exportViaDropdown">
        {{ $t("common.export") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { asyncComputed } from "@vueuse/core";
import dayjs from "dayjs";
import saveAs from "file-saver";
import JSZip from "jszip";
import { DownloadIcon } from "lucide-vue-next";
import type { FormInst, FormRules } from "naive-ui";
import {
  NButton,
  NDropdown,
  NForm,
  NFormItem,
  NRadio,
  NRadioGroup,
  NTooltip,
} from "naive-ui";
import { computed, nextTick, reactive, ref, watch } from "vue";
import { BBModal, BBTextField } from "@/bbkit";
import { t } from "@/plugins/i18n";
import { pushNotification } from "@/store";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import { isNullOrUndefined } from "@/utils";
import MaxRowCountSelect from "./GrantRequestPanel/MaxRowCountSelect.vue";
import { Drawer, DrawerContent, ErrorTipsButton } from "./v2";

interface LocalState {
  isRequesting: boolean;
  showDrawer: boolean;
  showModal: boolean;
}

export type DownloadContent = {
  content: Uint8Array;
  filename: string;
};

export interface ExportOption {
  limit: number;
  format: ExportFormat;
  password: string;
}

const props = withDefaults(
  defineProps<{
    size?: "small" | "tiny" | "medium" | "large";
    disabled?: boolean;
    supportFormats: ExportFormat[];
    supportPassword?: boolean;
    viewMode: "DRAWER" | "DROPDOWN";
    tooltip?: string;
    text?: string;
    validate?: (option: ExportOption) => boolean;
    maximumExportCount?: number;
  }>(),
  {
    size: "small",
    disabled: false,
    tooltip: undefined,
    supportPassword: false,
    text: () => t("common.export"),
    validate: (_: ExportOption) => true,
    maximumExportCount: Number.MAX_VALUE,
  }
);

const maxRowCountSelectRef = ref<InstanceType<typeof MaxRowCountSelect>>();

const defaultFormData = (): ExportOption => ({
  limit: Math.min(
    maxRowCountSelectRef.value?.maximum ?? Number.MAX_VALUE,
    1000
  ),
  format: props.supportFormats[0],
  password: "",
});

const emit = defineEmits<{
  (
    event: "export",
    option: {
      resolve: (content: DownloadContent[]) => void;
      reject: (reason?: unknown) => void;
      options: ExportOption;
    }
  ): Promise<void>;
}>();

const state = reactive<LocalState>({
  isRequesting: false,
  showDrawer: false,
  showModal: false,
});
const formRef = ref<FormInst>();
const formData = ref<ExportOption>(defaultFormData());

const rules: FormRules = {
  limit: [
    {
      required: true,
      validator: (rule, value: number) => {
        if (isNullOrUndefined(value)) {
          return new Error(t("export-data.error.export-rows-required"));
        }
        if (value <= 0) {
          return new Error(t("export-data.error.export-rows-must-gt-zero"));
        }
        return true;
      },
      trigger: ["input", "blur-sm"],
    },
  ],
  format: [
    {
      required: true,
    },
  ],
};

const exportDropdownOptions = computed(() => {
  return props.supportFormats.map((format) => ({
    label: t("sql-editor.download-as-file", {
      file: ExportFormat[format],
    }),
    key: format,
  }));
});

const tryExportViaDropdown = (format: ExportFormat) => {
  formData.value.format = format;
  if (props.supportPassword) {
    state.showModal = true;
  } else {
    doExport();
  }
};

const exportViaDropdown = () => {
  state.showModal = false;
  doExport();
};

const tryExportViaForm = (e: MouseEvent) => {
  e.preventDefault();
  formRef.value?.validate((errors) => {
    if (errors) return;
    doExport();
  });
};

const formErrors = asyncComputed(() => {
  if (!formRef.value) return [];
  try {
    return new Promise<string[]>((resolve) => {
      formRef.value!.validate((errors) => {
        resolve(
          errors?.flatMap((err) => err.map((e) => e.message ?? "")) ?? []
        );
      });
    });
  } catch {
    return [];
  }
}, []);

const handleClickExportButton = (e: MouseEvent) => {
  e.preventDefault();
  if (props.viewMode === "DROPDOWN") return;

  state.showDrawer = true;
};

const doExport = () => {
  if (state.isRequesting) {
    return;
  }

  state.isRequesting = true;
  const options = { ...formData.value };

  new Promise<DownloadContent[]>((resolve, reject) => {
    return emit("export", {
      resolve,
      reject,
      options,
    });
  })
    .then((content: DownloadContent[]) => {
      return doDownload(content, options.format);
    })
    .then(() => {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.succeed"),
        description: t("audit-log.export-finished"),
      });
    })
    .catch((error) => {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: `Failed to export data`,
        description: JSON.stringify(error),
      });
    })
    .finally(() => {
      state.isRequesting = false;
      state.showDrawer = false;
    });
};

const getExportFileType = (format: ExportFormat) => {
  switch (format) {
    case ExportFormat.CSV:
      return "text/csv";
    case ExportFormat.JSON:
      return "application/json";
    case ExportFormat.SQL:
      return "application/sql";
    case ExportFormat.XLSX:
      return "application/vnd.ms-excel";
  }
};

const convertSingleFile = async (
  { content, filename }: DownloadContent,
  format: ExportFormat
) => {
  const isZip = filename.endsWith(".zip");
  const fileType = isZip ? "application/zip" : getExportFileType(format);

  // Create Blob from Uint8Array
  const buffer = content.buffer.slice(
    content.byteOffset,
    content.byteOffset + content.byteLength
  ) as ArrayBuffer; // TypeScript 5.9.2 requires explicit ArrayBuffer type
  return new Blob([buffer], {
    type: fileType,
  });
};

const doDownloadSingleFile = async (
  content: DownloadContent,
  format: ExportFormat
) => {
  const blob = await convertSingleFile(content, format);
  const url = window.URL.createObjectURL(blob);

  const link = document.createElement("a");
  link.download = content.filename;
  link.href = url;
  link.click();
};

const doDownload = async (content: DownloadContent[], format: ExportFormat) => {
  if (content.length === 1) {
    return doDownloadSingleFile(content[0], format);
  }

  const zip = new JSZip();

  await Promise.all(
    content.map(async (c) => {
      const blob = await convertSingleFile(c, format);
      zip.file(c.filename, blob);
    })
  );

  const zipFile = await zip.generateAsync({ type: "blob" });
  const fileName = `download_${dayjs().format("YYYY-MM-DDTHH-mm-ss")}.zip`;
  saveAs(zipFile, fileName);
};

watch(
  () => [state.showDrawer, state.showModal],
  ([showDrawer, showModal]) => {
    if (showDrawer) {
      formData.value = defaultFormData();
      nextTick(() => {
        formData.value.limit = Math.min(
          maxRowCountSelectRef.value?.maximum ?? Number.MAX_VALUE,
          formData.value.limit
        );
      });
    } else if (showModal) {
      formData.value.password = "";
    }
  },
  { immediate: true }
);
</script>
