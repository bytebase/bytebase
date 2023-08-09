<template>
  <NDropdown
    trigger="hover"
    :options="exportDropdownOptions"
    @select="handleExportData"
  >
    <NButton
      quaternary
      size="tiny"
      :loading="state.isRequesting"
      :disabled="state.isRequesting"
    >
      <heroicons-outline:document-download class="h-5 w-5" />
    </NButton>
  </NDropdown>
</template>

<script lang="ts" setup>
import { NButton, NDropdown } from "naive-ui";
import { ExportRecord } from "./types";
import { computed, reactive } from "vue";
import {
  getExportFileType,
  pushNotification,
  useProjectIamPolicyStore,
  useSQLStore,
} from "@/store";
import dayjs from "dayjs";
import { useI18n } from "vue-i18n";
import { ExportRequest_Format } from "@/types/proto/v1/sql_service";

interface LocalState {
  showConfirmModal: boolean;
  isRequesting: boolean;
}

const props = defineProps<{
  exportRecord: ExportRecord;
}>();

const { t } = useI18n();
const sqlStore = useSQLStore();
const projectIamPolicyStore = useProjectIamPolicyStore();
const state = reactive<LocalState>({
  showConfirmModal: false,
  isRequesting: false,
});

const exportDropdownOptions = computed(() => [
  {
    label: t("sql-editor.download-as-file", { file: "CSV" }),
    key: ExportRequest_Format.CSV,
  },
  {
    label: t("sql-editor.download-as-file", { file: "JSON" }),
    key: ExportRequest_Format.JSON,
  },
  {
    label: t("sql-editor.download-as-file", { file: "SQL" }),
    key: ExportRequest_Format.SQL,
  },
  {
    label: t("sql-editor.download-as-file", { file: "XLSX" }),
    key: ExportRequest_Format.XLSX,
  },
]);

const handleExportData = async (format: ExportRequest_Format) => {
  if (state.isRequesting) {
    return;
  }

  state.isRequesting = true;

  const exportRecord = props.exportRecord;
  const database = exportRecord.database;

  try {
    const { content } = await sqlStore.exportData({
      name: database.instance,
      connectionDatabase: database.databaseName,
      statement: exportRecord.statement,
      limit: exportRecord.maxRowCount,
      format: format,
      admin: false,
    });

    const blob = new Blob([content], {
      type: getExportFileType(exportRecord.exportFormat),
    });
    const url = window.URL.createObjectURL(blob);

    const fileFormat = exportRecord.exportFormat.toLowerCase();
    const formattedDateString = dayjs(new Date()).format("YYYY-MM-DDTHH-mm-ss");
    const filename = `export-data-${formattedDateString}`;
    const link = document.createElement("a");
    link.download = `${filename}.${fileFormat}`;
    link.href = url;
    link.click();
    // Fetch the latest iam policy.
    await projectIamPolicyStore.fetchProjectIamPolicy(database.project, true);
    state.isRequesting = false;
    state.showConfirmModal = false;
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Failed to export data`,
      description: JSON.stringify(error),
    });
  }
};
</script>
