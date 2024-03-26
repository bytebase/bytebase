<template>
  <DataExportButton
    size="tiny"
    :file-type="'zip'"
    :support-formats="[
      ExportFormat.CSV,
      ExportFormat.JSON,
      ExportFormat.SQL,
      ExportFormat.XLSX,
    ]"
    @export="handleExportData"
  />
</template>

<script lang="ts" setup>
import type { BinaryLike } from "node:crypto";
import type { ExportOption } from "@/components/DataExportButton.vue";
import { useProjectIamPolicyStore } from "@/store";
import { useExportData } from "@/store/modules/export";
import { ExportFormat } from "@/types/proto/v1/common";
import type { ExportRecord } from "./types";

const props = defineProps<{
  exportRecord: ExportRecord;
}>();

const projectIamPolicyStore = useProjectIamPolicyStore();
const { exportData } = useExportData();

const handleExportData = async (
  options: ExportOption,
  callback: (content: BinaryLike | Blob, options: ExportOption) => void
) => {
  const exportRecord = props.exportRecord;
  const database = exportRecord.database;

  const content = await exportData({
    database: database.name,
    instance: database.instance,
    format: options.format,
    statement: exportRecord.statement,
    limit: exportRecord.maxRowCount,
    admin: false,
    password: options.password,
  });

  callback(content, options);

  // Fetch the latest iam policy.
  await projectIamPolicyStore.fetchProjectIamPolicy(database.project, true);
};
</script>
