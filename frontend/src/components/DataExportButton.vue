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
          :loading="state.isRequesting"
          :disabled="state.isRequesting || disabled"
          @click="handleClickExportButton"
        >
          <template #icon>
            <DownloadIcon class="h-4 w-4" />
          </template>
          <span v-if="size !== 'tiny'">
            {{ t("common.export") }}
          </span>
        </NButton>
      </template>
      <span class="text-sm"> {{ tooltip }} </span>
    </NTooltip>
  </NDropdown>

  <Drawer v-if="viewMode === 'DRAWER'" v-model:show="state.showDrawer">
    <DrawerContent
      :title="$t('custom-approval.risk-rule.risk.namespace.data_export')"
      class="w-[30rem] max-w-[100vw] relative"
    >
      <template #default>
        <NForm
          ref="formRef"
          :model="formData"
          :rules="rules"
          label-placement="left"
        >
          <NFormItem path="limit" :label="$t('export-data.export-rows')">
            <MaxRowCountSelect v-model:value="formData.limit" />
          </NFormItem>
          <NFormItem path="format" :label="$t('export-data.export-format')">
            <NRadioGroup v-model:value="formData.format">
              <NRadio
                v-for="format in supportFormats"
                :key="format"
                :value="format"
              >
                {{ exportFormatToJSON(format) }}
              </NRadio>
            </NRadioGroup>
          </NFormItem>
          <NFormItem
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
              disabled: formErrors.length > 0 || state.isRequesting,
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
    class="shadow-inner outline outline-gray-200"
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
    <div class="w-full flex items-center justify-end mt-2 space-x-3">
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
import type { BinaryLike } from "node:crypto";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBModal, BBTextField } from "@/bbkit";
import { pushNotification } from "@/store";
import { ExportFormat, exportFormatToJSON } from "@/types/proto/v1/common";
import { isNullOrUndefined } from "@/utils";
import MaxRowCountSelect from "./GrantRequestPanel/MaxRowCountSelect.vue";
import { Drawer, DrawerContent, ErrorTipsButton } from "./v2";

interface LocalState {
  isRequesting: boolean;
  showDrawer: boolean;
  showModal: boolean;
}

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
    viewMode: "DRAWER" | "DROPDOWN";
    fileType: "zip" | "raw";
    tooltip?: string;
  }>(),
  {
    size: "small",
    disabled: false,
    tooltip: undefined,
  }
);

const defaultFormData = (): ExportOption => ({
  limit: 1000,
  format: props.supportFormats[0],
  password: "",
});

const emit = defineEmits<{
  (
    event: "export",
    options: ExportOption,
    download: (content: BinaryLike | Blob, filename: string) => void
  ): Promise<void>;
}>();

const { t } = useI18n();
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
      trigger: ["input", "blur"],
    },
  ],
};

const exportDropdownOptions = computed(() => {
  return props.supportFormats.map((format) => ({
    label: t("sql-editor.download-as-file", {
      file: exportFormatToJSON(format),
    }),
    key: format,
  }));
});

const tryExportViaDropdown = async (format: ExportFormat) => {
  formData.value.format = format;
  if (props.fileType === "zip") {
    state.showModal = true;
  } else {
    await doExport();
  }
};

const exportViaDropdown = async () => {
  state.showModal = false;
  await doExport();
};

const tryExportViaForm = async (e: MouseEvent) => {
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

const doExport = async () => {
  if (state.isRequesting) {
    return;
  }

  state.isRequesting = true;
  const options = { ...formData.value };
  try {
    await emit(
      "export",
      options,
      (content: BinaryLike | Blob, filename: string) => {
        doDownload(content, options, filename);
      }
    );
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Failed to export data`,
      description: JSON.stringify(error),
    });
  } finally {
    state.isRequesting = false;
    state.showDrawer = false;
  }
};

const downloadFileAsZip = (options: ExportOption) => {
  return props.fileType === "zip" && !!options.password;
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

const doDownload = async (
  content: BinaryLike | Blob,
  options: ExportOption,
  filename: string
) => {
  const isZip = downloadFileAsZip(options);
  const fileType = isZip
    ? "application/zip"
    : getExportFileType(options.format);

  let blob = new Blob([content], {
    type: fileType,
  });
  if (options.format === ExportFormat.JSON) {
    const text = await blob.text();
    const formatted = JSON.stringify(JSON.parse(text), null, 2);
    blob = new Blob([formatted], {
      type: fileType,
    });
  }
  const url = window.URL.createObjectURL(blob);

  const fileFormat = exportFormatToJSON(options.format).toLowerCase();
  const link = document.createElement("a");
  link.download = `${filename}.${isZip ? "zip" : fileFormat}`;
  link.href = url;
  link.click();
};

watch(
  () => [state.showDrawer, state.showModal],
  ([showDrawer, showModal]) => {
    if (showDrawer) {
      formData.value = defaultFormData();
    } else if (showModal) {
      formData.value.password = "";
    }
  },
  { immediate: true }
);
</script>
