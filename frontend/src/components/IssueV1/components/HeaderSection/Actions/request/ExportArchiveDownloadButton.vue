<template>
  <NButton
    v-if="taskRun?.exportArchiveStatus === TaskRun_ExportArchiveStatus.READY"
    type="primary"
    :loading="state.isExporting"
    @click="downloadExportArchive"
  >
    <template #icon>
      <DownloadIcon class="w-5 h-5" />
    </template>
    {{ $t("common.download") }}
  </NButton>
  <div
    v-else-if="
      taskRun?.exportArchiveStatus === TaskRun_ExportArchiveStatus.EXPORTED
    "
    class="flex flex-row items-center gap-2 text-sm textlabel !leading-8"
  >
    <CircleCheckBigIcon class="w-5 h-auto" />
    {{ $t("issue.data-export.file-downloaded") }}
  </div>
</template>

<script setup lang="ts">
import dayjs from "dayjs";
import { head, last } from "lodash-es";
import { DownloadIcon, CircleCheckBigIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { watchEffect } from "vue";
import { useIssueContext } from "@/components/IssueV1";
import { issueServiceClient } from "@/grpcweb";
import { useSQLStore } from "@/store";
import { ExportFormat } from "@/types/proto/v1/common";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import {
  Plan_ExportDataConfig,
  Plan_Spec,
} from "@/types/proto/v1/plan_service";
import { TaskRun_ExportArchiveStatus } from "@/types/proto/v1/rollout_service";
import { ExportRequest } from "@/types/proto/v1/sql_service";

interface LocalState {
  isExporting: boolean;
}

const { issue, events, selectedStage } = useIssueContext();
const state = reactive<LocalState>({
  isExporting: false,
});

const taskRun = computed(() => {
  return last(issue.value.rolloutTaskRunList);
});

const exportDataConfig = computed(() => {
  return (
    (head(issue.value.planEntity?.specs) || Plan_Spec.fromPartial({}))
      .exportDataConfig || Plan_ExportDataConfig.fromPartial({})
  );
});

watchEffect(async () => {
  if (issue.value.status === IssueStatus.OPEN) {
    if (
      issue.value.rolloutTaskRunList.every(
        (taskRun) =>
          taskRun.exportArchiveStatus === TaskRun_ExportArchiveStatus.EXPORTED
      )
    ) {
      await issueServiceClient.batchUpdateIssuesStatus({
        parent: issue.value.project,
        issues: [issue.value.name],
        status: IssueStatus.DONE,
      });
    }
  }
});

const downloadExportArchive = async () => {
  state.isExporting = true;
  const content = await useSQLStore().exportData(
    ExportRequest.fromPartial({
      name: selectedStage.value.name,
    })
  );
  const fileType = getExportFileType(exportDataConfig.value);
  const blob = new Blob([content], {
    type: fileType,
  });
  const url = window.URL.createObjectURL(blob);

  const formattedDateString = dayjs(new Date()).format("YYYY-MM-DDTHH-mm-ss");
  const filename = `export-data-${formattedDateString}`;
  const link = document.createElement("a");
  link.download = `${filename}.zip`;
  link.href = url;
  link.click();
  events.emit("status-changed", { eager: true });
  state.isExporting = false;
};

const getExportFileType = (exportDataConfig: Plan_ExportDataConfig) => {
  if (exportDataConfig.password) {
    return "application/zip";
  }
  switch (exportDataConfig.format) {
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
</script>
