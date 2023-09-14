<template>
  <NDropdown
    trigger="hover"
    :options="exportDropdownOptions"
    @select="doExport"
  >
    <NButton
      :quaternary="size === 'tiny'"
      :size="size"
      :loading="state.isRequesting"
      :disabled="state.isRequesting || disabled"
    >
      <template #icon>
        <heroicons-outline:download class="h-5 w-5" />
      </template>
      <span v-if="size !== 'tiny'">
        {{ t("common.export") }}
      </span>
    </NButton>
  </NDropdown>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { NButton, NDropdown } from "naive-ui";
import { BinaryLike } from "node:crypto";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";
import { ExportFormat, exportFormatToJSON } from "@/types/proto/v1/common";

interface LocalState {
  isRequesting: boolean;
}

const props = withDefaults(
  defineProps<{
    size?: "small" | "tiny" | "medium" | "large";
    disabled?: boolean;
    supportFormats: ExportFormat[];
  }>(),
  {
    size: "small",
    disabled: false,
  }
);

const emit = defineEmits<{
  (
    event: "export",
    format: ExportFormat,
    download: (content: string, format: ExportFormat) => void
  ): Promise<void>;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  isRequesting: false,
});

const exportDropdownOptions = computed(() => {
  return props.supportFormats.map((format) => ({
    label: t("sql-editor.download-as-file", {
      file: exportFormatToJSON(format),
    }),
    key: format,
  }));
});

const doExport = async (format: ExportFormat) => {
  if (state.isRequesting) {
    return;
  }

  state.isRequesting = true;

  try {
    await emit("export", format, doDownload);
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Failed to export data`,
      description: JSON.stringify(error),
    });
  } finally {
    state.isRequesting = false;
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
</script>
