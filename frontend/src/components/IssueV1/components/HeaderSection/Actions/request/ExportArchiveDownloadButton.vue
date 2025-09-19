<template>
  <NTooltip
    v-if="taskRun?.exportArchiveStatus === TaskRun_ExportArchiveStatus.READY"
  >
    <template #trigger>
      <NButton
        type="primary"
        :loading="state.isExporting"
        @click="downloadExportArchive"
      >
        <template #icon>
          <DownloadIcon class="w-5 h-5" />
        </template>
        {{ $t("common.download") }}
      </NButton>
    </template>
    <span class="text-sm">
      {{ $t("issue.data-export.download-tooltip") }}
    </span>
  </NTooltip>
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
import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { head, last } from "lodash-es";
import { DownloadIcon, CircleCheckBigIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { computed, reactive } from "vue";
import { watchEffect } from "vue";
import { useIssueContext } from "@/components/IssueV1";
import { issueServiceClientConnect } from "@/grpcweb";
import { useSQLStore } from "@/store";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import {
  BatchUpdateIssuesStatusRequestSchema,
  IssueStatus as NewIssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_ExportDataConfigSchema,
  Plan_SpecSchema,
  type Plan_ExportDataConfig,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  TaskRun_ExportArchiveStatus,
  Task_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { ExportRequestSchema } from "@/types/proto-es/v1/sql_service_pb";
import { flattenTaskV1List } from "@/utils";

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
  const spec =
    head(issue.value.planEntity?.specs) || create(Plan_SpecSchema, {});
  return (
    (spec.config?.value as Plan_ExportDataConfig) ||
    create(Plan_ExportDataConfigSchema, {})
  );
});

watchEffect(async () => {
  if (issue.value.status === IssueStatus.OPEN) {
    if (
      flattenTaskV1List(issue.value.rolloutEntity).every((task) => {
        return [Task_Status.DONE, Task_Status.SKIPPED].includes(task.status);
      })
    ) {
      const request = create(BatchUpdateIssuesStatusRequestSchema, {
        parent: issue.value.project,
        issues: [issue.value.name],
        status: NewIssueStatus.DONE,
      });
      await issueServiceClientConnect.batchUpdateIssuesStatus(request);
    }
  }
});

const downloadExportArchive = async () => {
  state.isExporting = true;
  const content = await useSQLStore().exportData(
    create(ExportRequestSchema, {
      name: selectedStage.value.name,
    })
  );
  const fileType = getExportFileType(exportDataConfig.value);
  const buffer = content.buffer.slice(
    content.byteOffset,
    content.byteOffset + content.byteLength
  ) as ArrayBuffer;
  const blob = new Blob([buffer], {
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
