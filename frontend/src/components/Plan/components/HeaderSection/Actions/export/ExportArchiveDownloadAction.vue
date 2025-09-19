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
import { last } from "lodash-es";
import { DownloadIcon, CircleCheckBigIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { computed, reactive } from "vue";
import { usePlanContext } from "@/components/Plan/logic";
import { useSQLStore } from "@/store";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import type { Plan_ExportDataConfig } from "@/types/proto-es/v1/plan_service_pb";
import { TaskRun_ExportArchiveStatus } from "@/types/proto-es/v1/rollout_service_pb";
import { ExportRequestSchema } from "@/types/proto-es/v1/sql_service_pb";

interface LocalState {
  isExporting: boolean;
}

const { plan, rollout, taskRuns, events } = usePlanContext();
const state = reactive<LocalState>({
  isExporting: false,
});

const taskRun = computed(() => {
  if (!rollout.value || !taskRuns.value.length) return undefined;

  // Find export tasks first
  const exportTasks = rollout.value.stages
    .flatMap((stage) => stage.tasks)
    .filter((task) => {
      // Find tasks that belong to export data specs
      const exportSpec = plan.value.specs.find(
        (spec) =>
          spec.config?.case === "exportDataConfig" && spec.id === task.specId
      );
      return !!exportSpec;
    });

  // Get task runs for export tasks
  const exportTaskRuns = taskRuns.value.filter((taskRun) =>
    exportTasks.some((task) => taskRun.name.startsWith(task.name + "/"))
  );

  return last(exportTaskRuns);
});

const exportDataConfig = computed((): Plan_ExportDataConfig | undefined => {
  const spec = plan.value.specs.find(
    (spec) => spec.config?.case === "exportDataConfig"
  );
  return spec?.config?.value as Plan_ExportDataConfig;
});

const downloadExportArchive = async () => {
  if (!rollout.value) return;

  state.isExporting = true;
  try {
    const content = await useSQLStore().exportData(
      create(ExportRequestSchema, {
        name: rollout.value.name,
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
  } finally {
    state.isExporting = false;
  }
};

const getExportFileType = (exportDataConfig?: Plan_ExportDataConfig) => {
  if (!exportDataConfig) return "application/zip";

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
    default:
      return "application/zip";
  }
};
</script>
