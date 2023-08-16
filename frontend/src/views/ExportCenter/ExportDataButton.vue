<template>
  <DataExportButton
    size="tiny"
    :support-formats="['CSV', 'JSON', 'SQL', 'XLSX']"
    @export="handleExportData"
  />
</template>

<script lang="ts" setup>
import { BinaryLike } from "node:crypto";
import { ExportFormat } from "@/components/DataExportButton.vue";
import { useProjectIamPolicyStore } from "@/store";
import { useExportData } from "@/store/modules/export";
import { ExportRecord } from "./types";

const props = defineProps<{
  exportRecord: ExportRecord;
}>();

const projectIamPolicyStore = useProjectIamPolicyStore();
const { exportData } = useExportData();

const handleExportData = async (
  format: ExportFormat,
  callback: (content: BinaryLike | Blob, format: ExportFormat) => void
) => {
  const exportRecord = props.exportRecord;
  const database = exportRecord.database;

  const content = await exportData({
    database: database.name,
    instance: database.instance,
    format,
    statement: exportRecord.statement,
    limit: exportRecord.maxRowCount,
    admin: false,
  });

  callback(content, format);

  // Fetch the latest iam policy.
  await projectIamPolicyStore.fetchProjectIamPolicy(database.project, true);
};
</script>
