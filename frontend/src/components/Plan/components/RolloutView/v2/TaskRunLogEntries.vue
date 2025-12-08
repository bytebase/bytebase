<template>
  <NVirtualList
    v-if="displayItems.length > 0"
    ref="virtualListRef"
    :items="displayItems"
    :item-size="20"
    item-resizable
    item-key="key"
    class="w-full font-mono text-xs bg-gray-50 border border-gray-200 p-2 max-h-48"
    :class="isTruncated ? 'rounded-t rounded-b-none' : 'rounded'"
    @scroll="handleScroll"
  >
    <template #default="{ item }">
      <div v-if="item.isSpacer" class="h-3" />
      <div v-else class="flex items-start gap-x-2 py-0.5">
        <span class="text-gray-400 shrink-0">{{ item.time }}</span>
        <span class="shrink-0" :class="item.levelClass"
          >[{{ item.level }}]</span
        >
        <span class="text-blue-600 shrink-0">{{ item.typeLabel }}</span>
        <span :class="item.detailClass">{{ item.detail }}</span>
      </div>
    </template>
  </NVirtualList>
</template>

<script lang="ts" setup>
import type { Timestamp as PbTimestamp } from "@bufbuild/protobuf/wkt";
import type { VirtualListInst } from "naive-ui";
import { NVirtualList } from "naive-ui";
import { computed, nextTick, onMounted, ref, watch } from "vue";
import { getDateForPbTimestampProtoEs } from "@/types";
import type { TaskRunLogEntry } from "@/types/proto-es/v1/rollout_service_pb";
import {
  TaskRunLogEntry_TaskRunStatusUpdate_Status,
  TaskRunLogEntry_TransactionControl_Type,
  TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import { extractSheetCommandByIndex } from "@/utils";

// Types
type LogLevel = "ERR" | "WRN" | "INF";

interface DisplayItem {
  key: string; // Stable key based on timestamp
  time: string;
  level: LogLevel;
  levelClass: string;
  typeLabel: string;
  detail: string;
  detailClass: string;
  isSpacer?: boolean;
}

// Constants
const LOG_LEVEL_CONFIG: Record<
  LogLevel,
  { levelClass: string; detailClass: string }
> = {
  ERR: { levelClass: "text-red-600", detailClass: "text-red-700" },
  WRN: { levelClass: "text-yellow-600", detailClass: "text-yellow-700" },
  INF: { levelClass: "text-green-600", detailClass: "text-gray-600" },
};

const ENTRY_TYPE_LABELS: Partial<Record<TaskRunLogEntry_Type, string>> = {
  [TaskRunLogEntry_Type.COMMAND_EXECUTE]: "EXEC",
  [TaskRunLogEntry_Type.SCHEMA_DUMP]: "DUMP",
  [TaskRunLogEntry_Type.TASK_RUN_STATUS_UPDATE]: "STATUS",
  [TaskRunLogEntry_Type.TRANSACTION_CONTROL]: "TXN",
  [TaskRunLogEntry_Type.DATABASE_SYNC]: "SYNC",
  [TaskRunLogEntry_Type.PRIOR_BACKUP]: "BACKUP",
  [TaskRunLogEntry_Type.COMPUTE_DIFF]: "DIFF",
  [TaskRunLogEntry_Type.RETRY_INFO]: "RETRY",
};

const TXN_TYPE_LABELS: Partial<
  Record<TaskRunLogEntry_TransactionControl_Type, string>
> = {
  [TaskRunLogEntry_TransactionControl_Type.BEGIN]: "BEGIN",
  [TaskRunLogEntry_TransactionControl_Type.COMMIT]: "COMMIT",
  [TaskRunLogEntry_TransactionControl_Type.ROLLBACK]: "ROLLBACK",
};

// Auto-scroll threshold (px) for detecting if user is at bottom
const SCROLL_THRESHOLD = 50;

// Spacer item to ensure last log entry is fully visible when scrolled to bottom
const SPACER_ITEM: DisplayItem = {
  key: "spacer",
  time: "",
  level: "INF",
  levelClass: "",
  typeLabel: "",
  detail: "",
  detailClass: "",
  isSpacer: true,
};

// Props
const props = defineProps<{
  entries: TaskRunLogEntry[];
  sheet?: Sheet;
  isTruncated?: boolean;
}>();

// Refs for auto-scroll
const virtualListRef = ref<VirtualListInst>();
const isUserAtBottom = ref(true);

// Compare two protobuf timestamps for sorting (ascending)
const compareTimestamps = (
  a: PbTimestamp | undefined,
  b: PbTimestamp | undefined
): number => {
  if (!a && !b) return 0;
  if (!a) return -1;
  if (!b) return 1;
  const secondsDiff = Number(a.seconds) - Number(b.seconds);
  if (secondsDiff !== 0) return secondsDiff;
  return a.nanos - b.nanos;
};

// Sort entries by logTime ascending (oldest first)
const sortedEntries = computed(() => {
  return [...props.entries].sort((a, b) =>
    compareTimestamps(a.logTime, b.logTime)
  );
});

// Log display utilities
const formatLogTime = (timestamp: PbTimestamp | undefined): string => {
  if (!timestamp) return "--:--:--.---";
  const date = getDateForPbTimestampProtoEs(timestamp);
  if (!date) return "--:--:--.---";
  return (
    [
      date.getHours().toString().padStart(2, "0"),
      date.getMinutes().toString().padStart(2, "0"),
      date.getSeconds().toString().padStart(2, "0"),
    ].join(":") +
    "." +
    date.getMilliseconds().toString().padStart(3, "0")
  );
};

const getLogLevel = (entry: TaskRunLogEntry): LogLevel => {
  switch (entry.type) {
    case TaskRunLogEntry_Type.COMMAND_EXECUTE:
      if (entry.commandExecute?.response?.error) return "ERR";
      break;
    case TaskRunLogEntry_Type.TRANSACTION_CONTROL:
      if (entry.transactionControl?.error) return "ERR";
      if (
        entry.transactionControl?.type ===
        TaskRunLogEntry_TransactionControl_Type.ROLLBACK
      ) {
        return "WRN";
      }
      break;
    case TaskRunLogEntry_Type.SCHEMA_DUMP:
      if (entry.schemaDump?.error) return "ERR";
      break;
    case TaskRunLogEntry_Type.DATABASE_SYNC:
      if (entry.databaseSync?.error) return "ERR";
      break;
    case TaskRunLogEntry_Type.PRIOR_BACKUP:
      if (entry.priorBackup?.error) return "ERR";
      break;
    case TaskRunLogEntry_Type.RETRY_INFO:
      return "WRN";
    case TaskRunLogEntry_Type.COMPUTE_DIFF:
      if (entry.computeDiff?.error) return "ERR";
      break;
  }
  return "INF";
};

const getEntryTypeLabel = (entry: TaskRunLogEntry): string => {
  return ENTRY_TYPE_LABELS[entry.type] ?? "LOG";
};

// Entry detail handlers
const truncate = (str: string, maxLen: number): string => {
  return str.length <= maxLen ? str : str.substring(0, maxLen) + "...";
};

const getCommandExecuteDetail = (entry: TaskRunLogEntry): string => {
  const cmd = entry.commandExecute;
  if (!cmd) return "";

  // If error, show error message only
  if (cmd.response?.error) {
    return cmd.response.error;
  }

  // Try to get the actual statement
  let statement: string | undefined;
  if (cmd.statement) {
    statement = cmd.statement;
  } else if (cmd.commandIndexes.length > 0 && props.sheet) {
    statement = extractSheetCommandByIndex(props.sheet, cmd.commandIndexes[0]);
  }

  if (statement) {
    const stmt = statement.trim().replace(/\s+/g, " ");
    return truncate(stmt, 60);
  }

  return "-";
};

const getTransactionControlDetail = (entry: TaskRunLogEntry): string => {
  const txn = entry.transactionControl;
  if (!txn) return "";
  const typeStr = TXN_TYPE_LABELS[txn.type] ?? "";
  return txn.error ? `${typeStr} error: ${txn.error}` : typeStr;
};

const getProgressDetail = (
  data: { error?: string; startTime?: unknown; endTime?: unknown } | undefined,
  progressText: string
): string => {
  if (!data) return "";
  if (data.error) return data.error;
  if (data.startTime && data.endTime) return "completed";
  return progressText;
};

const getStatusUpdateDetail = (entry: TaskRunLogEntry): string => {
  const status = entry.taskRunStatusUpdate;
  if (!status) return "";
  switch (status.status) {
    case TaskRunLogEntry_TaskRunStatusUpdate_Status.RUNNING_WAITING:
      return "waiting to execute";
    case TaskRunLogEntry_TaskRunStatusUpdate_Status.RUNNING_RUNNING:
      return "executing";
    default:
      return "status changed";
  }
};

const getPriorBackupDetail = (entry: TaskRunLogEntry): string => {
  const backup = entry.priorBackup;
  if (!backup) return "";
  if (backup.error) return backup.error;
  const items = backup.priorBackupDetail?.items;
  if (items?.length) {
    return `completed (${items.length} tables backed up)`;
  }
  if (backup.startTime && backup.endTime) return "completed";
  return "backing up...";
};

const getRetryInfoDetail = (entry: TaskRunLogEntry): string => {
  const retry = entry.retryInfo;
  if (!retry) return "";
  const attempt = `attempt ${retry.retryCount}/${retry.maximumRetries}`;
  return retry.error ? `${attempt} - ${retry.error}` : attempt;
};

const getEntryDetail = (entry: TaskRunLogEntry): string => {
  switch (entry.type) {
    case TaskRunLogEntry_Type.COMMAND_EXECUTE:
      return getCommandExecuteDetail(entry);
    case TaskRunLogEntry_Type.TRANSACTION_CONTROL:
      return getTransactionControlDetail(entry);
    case TaskRunLogEntry_Type.SCHEMA_DUMP:
      return getProgressDetail(entry.schemaDump, "dumping...");
    case TaskRunLogEntry_Type.DATABASE_SYNC:
      return getProgressDetail(entry.databaseSync, "syncing...");
    case TaskRunLogEntry_Type.TASK_RUN_STATUS_UPDATE:
      return getStatusUpdateDetail(entry);
    case TaskRunLogEntry_Type.PRIOR_BACKUP:
      return getPriorBackupDetail(entry);
    case TaskRunLogEntry_Type.RETRY_INFO:
      return getRetryInfoDetail(entry);
    case TaskRunLogEntry_Type.COMPUTE_DIFF:
      return getProgressDetail(entry.computeDiff, "computing...");
    default:
      return "";
  }
};

// Generate stable key from timestamp
const getEntryKey = (entry: TaskRunLogEntry, index: number): string => {
  const ts = entry.logTime;
  if (ts) {
    // Use seconds + nanos for uniqueness
    return `${ts.seconds}-${ts.nanos}`;
  }
  // Fallback to index if no timestamp
  return `idx-${index}`;
};

// Pre-compute display data to avoid multiple function calls per render
const displayItems = computed((): DisplayItem[] => {
  const items: DisplayItem[] = sortedEntries.value.map((entry, index) => {
    const level = getLogLevel(entry);
    const config = LOG_LEVEL_CONFIG[level];
    return {
      key: getEntryKey(entry, index),
      time: formatLogTime(entry.logTime),
      level,
      levelClass: config.levelClass,
      typeLabel: getEntryTypeLabel(entry),
      detail: getEntryDetail(entry),
      detailClass: config.detailClass,
    };
  });
  if (items.length > 0) {
    items.push(SPACER_ITEM);
  }
  return items;
});

// Scroll to bottom helper
const scrollToBottom = () => {
  nextTick(() => {
    virtualListRef.value?.scrollTo({ position: "bottom" });
  });
};

// Auto-scroll: detect if user is at bottom
const handleScroll = (event: Event) => {
  const el = event.target as HTMLElement;
  const distanceFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight;
  isUserAtBottom.value = distanceFromBottom < SCROLL_THRESHOLD;
};

// Scroll to bottom on initial mount
onMounted(scrollToBottom);

// Auto-scroll to bottom when new entries arrive (if user is at bottom)
watch(
  () => props.entries.length,
  () => {
    if (isUserAtBottom.value) {
      scrollToBottom();
    }
  }
);
</script>

