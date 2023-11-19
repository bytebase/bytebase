<template>
  <NDropdown
    trigger="hover"
    :options="exportDropdownOptions"
    :disabled="viewMode === 'DRAWER'"
    @select="tryExportViaDropdown"
  >
    <NButton
      :quaternary="size === 'tiny'"
      :size="size"
      :loading="state.isRequesting"
      :disabled="state.isRequesting || disabled"
      @click="handleClickExportButton"
    >
      <template #icon>
        <heroicons-outline:download class="h-5 w-5" />
      </template>
      <span v-if="size !== 'tiny'">
        {{ t("common.export") }}
      </span>
    </NButton>
  </NDropdown>

  <Drawer v-if="viewMode === 'DRAWER'" v-model:show="state.showDrawer">
    <DrawerContent
      :title="$t('export-data.self')"
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
            <NInputNumber
              ref="limitInputRef"
              v-model:value="formData.limit"
              @keydown.enter.prevent
            />
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
</template>

<script lang="ts" setup>
import { asyncComputed } from "@vueuse/core";
import dayjs from "dayjs";
import {
  FormInst,
  FormRules,
  InputNumberInst,
  NButton,
  NDropdown,
  NForm,
  NFormItem,
  NInputNumber,
  NRadio,
  NRadioGroup,
} from "naive-ui";
import { BinaryLike } from "node:crypto";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";
import { ExportFormat, exportFormatToJSON } from "@/types/proto/v1/common";
import { isNullOrUndefined } from "@/utils";
import { ErrorTipsButton } from "./v2";

interface LocalState {
  isRequesting: boolean;
  showDrawer: boolean;
}
interface FormData {
  limit: number;
  format: ExportFormat;
}
const props = withDefaults(
  defineProps<{
    size?: "small" | "tiny" | "medium" | "large";
    disabled?: boolean;
    supportFormats: ExportFormat[];
    allowSpecifyRowCount?: boolean;
  }>(),
  {
    size: "small",
    disabled: false,
    allowSpecifyRowCount: false,
  }
);
const defaultFormData = (): FormData => ({
  limit: 1000,
  format: props.supportFormats[0],
});

const emit = defineEmits<{
  (
    event: "export",
    format: ExportFormat,
    download: (content: BinaryLike | Blob, format: ExportFormat) => void,
    limit: number | undefined // number if allowSpecifyRowCount, undefined otherwise
  ): Promise<void>;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  isRequesting: false,
  showDrawer: false,
});
const formRef = ref<FormInst>();
const limitInputRef = ref<InputNumberInst>();
const formData = ref<FormData>(defaultFormData());

const viewMode = computed(() => {
  return props.allowSpecifyRowCount ? "DRAWER" : "DROPDOWN";
});

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
  doExport(format, undefined);
};
const tryExportViaForm = async (e: MouseEvent) => {
  e.preventDefault();
  formRef.value?.validate((errors) => {
    if (errors) return;
    doExport(formData.value.format, formData.value.limit);
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
  if (viewMode.value === "DROPDOWN") return;

  state.showDrawer = true;
};

const doExport = async (format: ExportFormat, limit: number | undefined) => {
  if (state.isRequesting) {
    return;
  }

  state.isRequesting = true;

  try {
    await emit("export", format, doDownload, limit);
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

const doDownload = (content: BinaryLike | Blob, format: ExportFormat) => {
  const blob = new Blob([content], {
    type: getExportFileType(format),
  });
  const url = window.URL.createObjectURL(blob);

  const fileFormat = exportFormatToJSON(format).toLowerCase();
  const formattedDateString = dayjs(new Date()).format("YYYY-MM-DDTHH-mm-ss");
  const filename = `export-data-${formattedDateString}`;
  const link = document.createElement("a");
  link.download = `${filename}.${fileFormat}`;
  link.href = url;
  link.click();
};

watch(
  () => state.showDrawer,
  (show) => {
    if (show) {
      formData.value = defaultFormData();
    }
  },
  { immediate: true }
);
</script>
