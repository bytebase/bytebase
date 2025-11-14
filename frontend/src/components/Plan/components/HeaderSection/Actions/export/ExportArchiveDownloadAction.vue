<template>
  <NTooltip>
    <template #trigger>
      <NButton
        type="primary"
        :loading="state.isExporting"
        @click="downloadExportArchive"
        v-if="!isExported"
      >
        <template #icon>
          <DownloadIcon class="w-5 h-5" />
        </template>
        {{ $t("common.download") }}
      </NButton>
      <div
        v-else
        class="flex flex-row items-center gap-2 text-sm textlabel leading-8!"
      >
        <CalendarXIcon class="w-5 h-auto" />
        {{ $t("issue.data-export.file-expired") }}
      </div>
    </template>
    <span class="text-sm">
      {{ $t("issue.data-export.download-tooltip") }}
    </span>
  </NTooltip>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { first, orderBy } from "lodash-es";
import { CalendarXIcon, DownloadIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { computed, reactive } from "vue";
import { usePlanContext } from "@/components/Plan/logic";
import { useSQLStore } from "@/store";
import {
  type TaskRun,
  TaskRun_ExportArchiveStatus,
} from "@/types/proto-es/v1/rollout_service_pb";
import { ExportRequestSchema } from "@/types/proto-es/v1/sql_service_pb";
import { extractTaskRunUID, extractTaskUID } from "@/utils";

interface LocalState {
  isExporting: boolean;
}

const { rollout, taskRuns, events } = usePlanContext();
const state = reactive<LocalState>({
  isExporting: false,
});

const isExported = computed(() => {
  const exportTaskRuns = rollout.value?.stages
    .flatMap((stage) => stage.tasks)
    .map((task) => {
      const taskRunsForTask = taskRuns.value.filter(
        (taskRun) => extractTaskUID(taskRun.name) === extractTaskUID(task.name)
      );
      return first(
        orderBy(
          taskRunsForTask,
          (taskRun) => Number(extractTaskRunUID(taskRun.name)),
          "desc"
        )
      );
    })
    .filter(Boolean) as TaskRun[];
  return exportTaskRuns.every(
    (taskRun) =>
      taskRun.exportArchiveStatus === TaskRun_ExportArchiveStatus.EXPORTED
  );
});

const downloadExportArchive = async () => {
  if (!rollout.value) return;

  state.isExporting = true;
  try {
    const content = await useSQLStore().exportData(
      create(ExportRequestSchema, {
        name: `${rollout.value.name}/stages/-`,
      })
    );
    const buffer = content.buffer.slice(
      content.byteOffset,
      content.byteOffset + content.byteLength
    ) as ArrayBuffer;
    const blob = new Blob([buffer], {
      type: "application/zip", // the download file is always zip file.
    });
    const url = window.URL.createObjectURL(blob);

    const formattedDateString = dayjs(new Date()).format("YYYY-MM-DDTHH-mm-ss");
    const filename = `export-data-${formattedDateString}.zip`;
    const link = document.createElement("a");
    link.download = filename;
    link.href = url;
    link.click();
    events.emit("status-changed", { eager: true });
  } finally {
    state.isExporting = false;
  }
};
</script>
