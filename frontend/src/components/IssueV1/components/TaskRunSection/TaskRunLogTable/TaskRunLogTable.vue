<template>
  <NDataTable
    :columns="columns"
    :loading="isLoading"
    :data="logEntries"
    size="small"
  />
</template>

<script setup lang="tsx">
import { computedAsync } from "@vueuse/core";
import { NDataTable, type DataTableColumn } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useIssueContext } from "@/components/IssueV1";
import { rolloutServiceClient } from "@/grpcweb";
import { useSheetV1Store } from "@/store";
import {
  TaskRunLogEntry,
  type TaskRun,
} from "@/types/proto/v1/rollout_service";
import { sheetNameOfTaskV1 } from "@/utils";
import AffectedRowsCell from "./AffectedRowsCell.vue";
import DurationCell from "./DurationCell.vue";
import StatementCell from "./StatementCell.vue";

const props = defineProps<{
  taskRun: TaskRun;
}>();

const { t } = useI18n();
const { selectedTask } = useIssueContext();
const isLoadingTaskRunLog = ref(false);
const isLoadingSheet = ref(false);
const isLoading = computed(() => {
  return isLoadingTaskRunLog.value || isLoadingSheet.value;
});
const taskRunLog = computedAsync(
  () => {
    console.log("evaluate taskRunLog", props.taskRun.name);
    return rolloutServiceClient.getTaskRunLog({
      parent: props.taskRun.name,
    });
  },
  undefined,
  {
    evaluating: isLoadingTaskRunLog,
  }
);
const sheetName = computed(() => {
  return sheetNameOfTaskV1(selectedTask.value);
});
const sheet = computedAsync(
  async () => {
    const name = sheetName.value;

    console.log("evaluate sheet", name);
    return useSheetV1Store().getOrFetchSheetByName(name, "FULL");
  },
  undefined,
  {
    evaluating: isLoadingSheet,
  }
);
const logEntries = computed(() => {
  if (isLoading.value) return [];
  if (!sheet.value) return [];
  return taskRunLog.value?.entries ?? [];
});

const columns: DataTableColumn<TaskRunLogEntry>[] = [
  {
    key: "serial",
    title: () => t("issue.task-run.task-run-log.batch"),
    width: 50,
    className: "whitespace-nowrap",
    render: (row, index) => {
      return String(index + 1);
    },
  },
  {
    key: "type",
    title: () => t("common.type"),
    width: 120,
    className: "whitespace-nowrap",
    render: (entry) => {
      return <span class="text-sm">{entry.type}</span>;
    },
  },
  {
    key: "statement",
    title: () => t("common.statement"),
    render: (entry) => {
      return <StatementCell entry={entry} sheet={sheet.value} />;
    },
  },
  {
    key: "affected-rows",
    title: () => t("issue.task-run.task-run-log.affected-rows"),
    width: 120,
    render: (entry) => {
      return <AffectedRowsCell entry={entry} sheet={sheet.value} />;
    },
  },
  {
    key: "duration",
    title: () => t("common.duration"),
    width: 120,
    render: (entry) => {
      return <DurationCell entry={entry} />;
    },
  },
];
</script>
