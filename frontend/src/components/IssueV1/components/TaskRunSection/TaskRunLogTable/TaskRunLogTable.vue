<template>
  <NDataTable
    :columns="columns"
    :loading="isLoading"
    :data="flattenLogEntries"
    :row-key="rowKey"
    size="small"
  />
</template>

<script setup lang="tsx">
import { computedAsync } from "@vueuse/core";
import { head } from "lodash-es";
import { CircleAlertIcon } from "lucide-vue-next";
import { NTooltip, NDataTable, type DataTableColumn } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useIssueContext } from "@/components/IssueV1";
import { rolloutServiceClient } from "@/grpcweb";
import { useSheetV1Store } from "@/store";
import {
  type TaskRun,
  TaskRunLogEntry_Type,
} from "@/types/proto/v1/rollout_service";
import { sheetNameOfTaskV1 } from "@/utils";
import DetailCell, { detailCellRowSpan } from "./DetailCell";
import DurationCell from "./DurationCell.vue";
import LogTimeCell from "./LogTimeCell.vue";
import StatementCell from "./StatementCell.vue";
import { type FlattenLogEntry, displayTaskRunLogEntryType } from "./common";

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
const flattenLogEntries = computed(() => {
  const flattenEntries: FlattenLogEntry[] = [];
  logEntries.value.forEach((entry, batch) => {
    const {
      type,
      taskRunStatusUpdate,
      schemaDump,
      commandExecute,
      transactionControl,
      databaseSync,
      deployId,
    } = entry;
    if (
      type === TaskRunLogEntry_Type.TASK_RUN_STATUS_UPDATE &&
      taskRunStatusUpdate
    ) {
      flattenEntries.push({
        batch,
        deployId,
        serial: 0,
        type: TaskRunLogEntry_Type.TASK_RUN_STATUS_UPDATE,
        startTime: entry.logTime,
        endTime: undefined,
        taskRunStatusUpdate,
      });
    }
    if (type === TaskRunLogEntry_Type.DATABASE_SYNC && databaseSync) {
      flattenEntries.push({
        batch,
        deployId,
        serial: 0,
        type: TaskRunLogEntry_Type.DATABASE_SYNC,
        startTime: databaseSync.startTime,
        endTime: databaseSync.endTime,
        databaseSync,
      });
    }
    if (
      type === TaskRunLogEntry_Type.TRANSACTION_CONTROL &&
      transactionControl
    ) {
      flattenEntries.push({
        batch,
        deployId,
        serial: 0,
        type: TaskRunLogEntry_Type.TRANSACTION_CONTROL,
        startTime: entry.logTime,
        endTime: undefined,
        transactionControl,
      });
    }
    if (type === TaskRunLogEntry_Type.SCHEMA_DUMP && schemaDump) {
      flattenEntries.push({
        batch,
        deployId,
        serial: 0,
        type: TaskRunLogEntry_Type.SCHEMA_DUMP,
        startTime: schemaDump.startTime,
        endTime: schemaDump.endTime,
        schemaDump,
      });
    }
    if (type === TaskRunLogEntry_Type.COMMAND_EXECUTE && commandExecute) {
      const { response, logTime: startTime } = commandExecute;
      commandExecute.commandIndexes.forEach((commandIndex, serial) => {
        let affectedRows = response?.affectedRows;
        if (
          commandExecute.commandIndexes.length ===
          response?.allAffectedRows.length
        ) {
          affectedRows = response?.allAffectedRows[serial] ?? affectedRows;
        }
        const endTime = response?.logTime;
        flattenEntries.push({
          batch,
          deployId,
          serial,
          type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
          startTime,
          endTime,
          commandExecute: {
            raw: commandExecute,
            commandIndex,
            affectedRows,
          },
        });
      });
    }
  });
  return flattenEntries;
});
const headDeployId = computed(() => head(logEntries.value)?.deployId);

const rowKey = (entry: FlattenLogEntry) => {
  return `${entry.batch}-${entry.serial}`;
};
const rowSpan = (entry: FlattenLogEntry) => {
  if (
    entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE &&
    entry.commandExecute
  ) {
    const { commandExecute } = entry;
    return commandExecute.raw.commandIndexes.length;
  }
  return 1;
};
const colSpan = (entry: FlattenLogEntry) => {
  if (
    entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE &&
    entry.commandExecute
  ) {
    const { commandExecute } = entry;
    if (commandExecute.raw.commandIndexes.length > 1) {
      return 1;
    }
  }
  return 2;
};

const columns = computed(() => {
  const splitBatchAndSerialCol = flattenLogEntries.value.some(
    (entry) => colSpan(entry) === 1
  );
  const columns: (DataTableColumn<FlattenLogEntry> & { hide?: boolean })[] = [
    {
      key: "batch",
      title: () => "",
      width: 50,
      className: "whitespace-nowrap",
      titleColSpan: splitBatchAndSerialCol ? 2 : 1,
      rowSpan,
      colSpan: (entry) => {
        if (splitBatchAndSerialCol) return colSpan(entry);
        return 1;
      },
      render: (entry) => {
        return (
          <div class="flex flex-row items-center gap-1">
            <span>{String(entry.batch + 1)}</span>
            {headDeployId.value !== entry.deployId && (
              <NTooltip>
                {{
                  trigger: () => (
                    <CircleAlertIcon class="w-4 h-auto text-red-600" />
                  ),
                  default: () => (
                    <div class="max-w-[20rem]">
                      Deploy ID mismatch. Please check if there is another
                      deployment running.
                    </div>
                  ),
                }}
              </NTooltip>
            )}
          </div>
        );
      },
    },
    {
      key: "serial",
      title: () => "",
      width: 50,
      className: "whitespace-nowrap",
      hide: !splitBatchAndSerialCol,
      render: (entry) => {
        return String(entry.serial + 1);
      },
    },
    {
      key: "type",
      title: () => t("common.type"),
      width: 120,
      className: "whitespace-nowrap",
      render: (entry) => {
        const text = displayTaskRunLogEntryType(entry.type);
        if (text) {
          return <span class="text-sm">{text}</span>;
        }
        return <span class="text-sm text-control-placeholder">-</span>;
      },
    },
    {
      key: "statement",
      title: () => t("common.statement"),
      width: "60%",
      render: (entry) => {
        return <StatementCell entry={entry} sheet={sheet.value} />;
      },
    },
    {
      key: "detail",
      title: () => t("common.detail"),
      width: "40%",
      rowSpan: detailCellRowSpan,
      render: (entry) => {
        return <DetailCell entry={entry} sheet={sheet.value} />;
      },
    },
    {
      key: "log-time",
      title: () => t("common.time"),
      width: 120,
      rowSpan,
      render: (entry) => {
        return <LogTimeCell entry={entry} />;
      },
    },
    {
      key: "duration",
      title: () => t("common.duration"),
      width: 120,
      rowSpan,
      render: (entry) => {
        return <DurationCell entry={entry} />;
      },
    },
  ];
  return columns.filter((column) => !column.hide);
});
</script>
