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
import { create } from "@bufbuild/protobuf";
import { computedAsync } from "@vueuse/core";
import { last } from "lodash-es";
import { CircleAlertIcon } from "lucide-vue-next";
import { NTooltip, NDataTable, type DataTableColumn } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { GetTaskRunLogRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import {
  type TaskRun,
  TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import DetailCell, { detailCellRowSpan } from "./DetailCell";
import DurationCell from "./DurationCell.vue";
import LogTimeCell from "./LogTimeCell.vue";
import StatementCell from "./StatementCell.vue";
import {
  type FlattenLogEntry,
  convertTaskRunLogEntryToFlattenLogEntries,
  displayTaskRunLogEntryType,
} from "./common";

const props = defineProps<{
  taskRun: TaskRun;
  sheet?: Sheet;
}>();

const { t } = useI18n();
const isLoading = ref(false);

const taskRunLog = computedAsync(
  async () => {
    const request = create(GetTaskRunLogRequestSchema, {
      parent: props.taskRun.name,
    });
    const response = await rolloutServiceClientConnect.getTaskRunLog(request);
    return response;
  },
  undefined,
  {
    evaluating: isLoading,
  }
);
const logEntries = computed(() => {
  if (isLoading.value) return [];
  if (!props.sheet) return [];
  return taskRunLog.value?.entries ?? [];
});
const flattenLogEntries = computed(() => {
  const flattenEntries: FlattenLogEntry[] = [];
  logEntries.value.forEach((entry, batch) => {
    flattenEntries.push(
      ...convertTaskRunLogEntryToFlattenLogEntries(entry, batch)
    );
  });
  return flattenEntries;
});
const lastDeployId = computed(() => last(logEntries.value)?.deployId);

const rowKey = (entry: FlattenLogEntry) => {
  return `${entry.batch}-${entry.serial}`;
};
const rowSpan = (entry: FlattenLogEntry) => {
  if (
    entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE &&
    entry.commandExecute
  ) {
    const { commandExecute } = entry;
    if (commandExecute.kind === "commandIndexes") {
      return commandExecute.commandIndexes.length;
    }
    return 1;
  }
  return 1;
};

const colSpan = (entry: FlattenLogEntry) => {
  if (
    entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE &&
    entry.commandExecute
  ) {
    const { commandExecute } = entry;
    if (
      commandExecute.kind === "commandIndexes" &&
      commandExecute.commandIndexes.length > 1
    ) {
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
            {lastDeployId.value !== entry.deployId && (
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
      key: "detail",
      title: () => t("common.detail"),
      width: "40%",
      rowSpan: detailCellRowSpan,
      render: (entry) => {
        return (
          <div class="flex flex-row justify-start items-center">
            <DetailCell entry={entry} sheet={props.sheet} />
          </div>
        );
      },
    },
    {
      key: "statement",
      title: () => t("common.statement"),
      width: "60%",
      render: (entry) => {
        return <StatementCell entry={entry} sheet={props.sheet} />;
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
    {
      key: "log-time",
      title: () => t("common.time"),
      width: 120,
      rowSpan,
      render: (entry) => {
        return <LogTimeCell entry={entry} />;
      },
    },
  ];
  return columns.filter((column) => !column.hide);
});
</script>
