<template>
  <div
    v-if="sections.length > 0"
    class="w-full font-mono text-xs bg-gray-50 border border-gray-200 overflow-hidden rounded"
  >
    <div
      v-for="section in sections"
      :key="section.id"
      class="border-b border-gray-200 last:border-b-0"
    >
      <!-- Section Header -->
      <div
        class="flex items-center gap-x-2 px-3 py-1.5 bg-white hover:bg-gray-50 cursor-pointer select-none"
        @click="toggleSection(section.id)"
      >
        <!-- Expand/Collapse Icon -->
        <component
          :is="expandedSections.has(section.id) ? ChevronDownIcon : ChevronRightIcon"
          class="w-3.5 h-3.5 text-gray-400 shrink-0"
        />
        <!-- Status Icon -->
        <component
          :is="section.statusIcon"
          class="w-3.5 h-3.5 shrink-0"
          :class="[
            section.statusClass,
            { 'animate-spin': section.status === 'running' },
          ]"
        />
        <!-- Section Title -->
        <span class="text-gray-700">{{ section.label }}</span>
        <!-- Entry Count -->
        <span v-if="section.entryCount > 1" class="text-gray-400">
          ({{ section.entryCount }})
        </span>
        <!-- Spacer -->
        <span class="flex-1" />
        <!-- Duration -->
        <span v-if="section.duration" class="text-gray-500 tabular-nums">
          {{ section.duration }}
        </span>
      </div>

      <!-- Section Content with Virtual Scroll -->
      <NVirtualList
        v-if="expandedSections.has(section.id)"
        :items="section.items"
        :item-size="ITEM_HEIGHT"
        item-resizable
        :style="{ maxHeight: `${MAX_VISIBLE_ITEMS * ITEM_HEIGHT}px` }"
        class="bg-gray-50 border-t border-gray-100"
      >
        <template #default="{ item, index }">
          <div
            class="flex items-start gap-x-2 px-3 py-0.5 hover:bg-gray-100"
            :class="{ 'border-t border-gray-100': index > 0 }"
          >
            <!-- Row Number -->
            <span class="text-gray-300 w-6 text-right shrink-0 tabular-nums">
              {{ index + 1 }}
            </span>
            <!-- Timestamp -->
            <span class="text-gray-400 shrink-0 tabular-nums">
              {{ item.time }}
            </span>
            <!-- Relative Time -->
            <span
              v-if="item.relativeTime"
              class="text-gray-300 shrink-0 tabular-nums"
            >
              {{ item.relativeTime }}
            </span>
            <!-- Status Indicator -->
            <span :class="item.levelClass" class="shrink-0">
              {{ item.levelIndicator }}
            </span>
            <!-- Detail -->
            <span :class="item.detailClass" class="break-all">
              {{ item.detail }}
            </span>
            <!-- Affected Rows -->
            <span
              v-if="item.affectedRows !== undefined"
              class="text-gray-400 shrink-0 ml-auto"
            >
              {{ item.affectedRows }} rows
            </span>
          </div>
        </template>
      </NVirtualList>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Timestamp as PbTimestamp } from "@bufbuild/protobuf/wkt";
import {
  CheckCircle2Icon,
  ChevronDownIcon,
  ChevronRightIcon,
  CircleIcon,
  LoaderCircleIcon,
  XCircleIcon,
} from "lucide-vue-next";
import { NVirtualList } from "naive-ui";
import type { Component } from "vue";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { getDateForPbTimestampProtoEs } from "@/types";
import type { TaskRunLogEntry } from "@/types/proto-es/v1/rollout_service_pb";
import {
  TaskRunLogEntry_TaskRunStatusUpdate_Status,
  TaskRunLogEntry_TransactionControl_Type,
  TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import { extractSheetCommandByIndex } from "@/utils";

// Constants for virtual scroll
const ITEM_HEIGHT = 20; // px per row
const MAX_VISIBLE_ITEMS = 10; // Show max 10 items before scrolling

// Types
type SectionStatus = "success" | "error" | "running" | "pending";

interface DisplayItem {
  key: string;
  time: string;
  relativeTime: string;
  levelIndicator: string;
  levelClass: string;
  detail: string;
  detailClass: string;
  affectedRows?: number;
}

interface Section {
  id: string; // Unique identifier for this section
  type: TaskRunLogEntry_Type;
  label: string;
  status: SectionStatus;
  statusIcon: Component;
  statusClass: string;
  duration: string;
  entryCount: number;
  items: DisplayItem[];
}

// Props
const props = defineProps<{
  entries: TaskRunLogEntry[];
  sheet?: Sheet;
}>();

const { t } = useI18n();

// Section labels
const getSectionLabel = (type: TaskRunLogEntry_Type): string => {
  switch (type) {
    case TaskRunLogEntry_Type.SCHEMA_DUMP:
      return t("task-run.log-type.schema-dump");
    case TaskRunLogEntry_Type.COMMAND_EXECUTE:
      return t("task-run.log-type.command-execute");
    case TaskRunLogEntry_Type.DATABASE_SYNC:
      return t("task-run.log-type.database-sync");
    case TaskRunLogEntry_Type.TASK_RUN_STATUS_UPDATE:
      return t("task-run.log-type.status-update");
    case TaskRunLogEntry_Type.TRANSACTION_CONTROL:
      return t("task-run.log-type.transaction");
    case TaskRunLogEntry_Type.PRIOR_BACKUP:
      return t("task-run.log-type.prior-backup");
    case TaskRunLogEntry_Type.RETRY_INFO:
      return t("task-run.log-type.retry");
    case TaskRunLogEntry_Type.COMPUTE_DIFF:
      return t("task-run.log-type.compute-diff");
    default:
      return "Unknown";
  }
};

// Status configuration
const STATUS_CONFIG: Record<SectionStatus, { icon: Component; class: string }> =
  {
    success: { icon: CheckCircle2Icon, class: "text-green-600" },
    error: { icon: XCircleIcon, class: "text-red-600" },
    running: { icon: LoaderCircleIcon, class: "text-blue-600" },
    pending: { icon: CircleIcon, class: "text-gray-400" },
  };

// Expanded sections state (keyed by section id)
const expandedSections = ref<Set<string>>(new Set());

const toggleSection = (sectionId: string) => {
  if (expandedSections.value.has(sectionId)) {
    expandedSections.value.delete(sectionId);
  } else {
    expandedSections.value.add(sectionId);
  }
  // Trigger reactivity
  expandedSections.value = new Set(expandedSections.value);
};

// Timestamp utilities
const getTimestampMs = (ts: PbTimestamp | undefined): number => {
  if (!ts) return 0;
  return Number(ts.seconds) * 1000 + ts.nanos / 1000000;
};

const formatTime = (ts: PbTimestamp | undefined): string => {
  if (!ts) return "--:--:--.---";
  const date = getDateForPbTimestampProtoEs(ts);
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

const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  const mins = Math.floor(ms / 60000);
  const secs = ((ms % 60000) / 1000).toFixed(0);
  return `${mins}m ${secs}s`;
};

const formatRelativeTime = (ms: number): string => {
  if (ms < 1000) return `+${ms.toFixed(0)}ms`;
  return `+${(ms / 1000).toFixed(2)}s`;
};

// Entry analysis
const hasError = (entry: TaskRunLogEntry): boolean => {
  switch (entry.type) {
    case TaskRunLogEntry_Type.COMMAND_EXECUTE:
      return !!entry.commandExecute?.response?.error;
    case TaskRunLogEntry_Type.TRANSACTION_CONTROL:
      return !!entry.transactionControl?.error;
    case TaskRunLogEntry_Type.SCHEMA_DUMP:
      return !!entry.schemaDump?.error;
    case TaskRunLogEntry_Type.DATABASE_SYNC:
      return !!entry.databaseSync?.error;
    case TaskRunLogEntry_Type.PRIOR_BACKUP:
      return !!entry.priorBackup?.error;
    case TaskRunLogEntry_Type.COMPUTE_DIFF:
      return !!entry.computeDiff?.error;
    default:
      return false;
  }
};

const isComplete = (entry: TaskRunLogEntry): boolean => {
  switch (entry.type) {
    case TaskRunLogEntry_Type.COMMAND_EXECUTE:
      return !!entry.commandExecute?.response;
    case TaskRunLogEntry_Type.SCHEMA_DUMP:
      return !!(entry.schemaDump?.startTime && entry.schemaDump?.endTime);
    case TaskRunLogEntry_Type.DATABASE_SYNC:
      return !!(entry.databaseSync?.startTime && entry.databaseSync?.endTime);
    case TaskRunLogEntry_Type.PRIOR_BACKUP:
      return !!(entry.priorBackup?.startTime && entry.priorBackup?.endTime);
    case TaskRunLogEntry_Type.COMPUTE_DIFF:
      return !!(entry.computeDiff?.startTime && entry.computeDiff?.endTime);
    case TaskRunLogEntry_Type.TRANSACTION_CONTROL:
      return true;
    case TaskRunLogEntry_Type.TASK_RUN_STATUS_UPDATE:
      return true;
    case TaskRunLogEntry_Type.RETRY_INFO:
      return true;
    default:
      return true;
  }
};

// Get entry detail text
const getEntryDetail = (entry: TaskRunLogEntry): string => {
  switch (entry.type) {
    case TaskRunLogEntry_Type.COMMAND_EXECUTE: {
      const cmd = entry.commandExecute;
      if (!cmd) return "";
      if (cmd.response?.error) return cmd.response.error;

      let statement: string | undefined;
      if (cmd.statement) {
        statement = cmd.statement;
      } else if (cmd.commandIndexes.length > 0 && props.sheet) {
        statement = extractSheetCommandByIndex(
          props.sheet,
          cmd.commandIndexes[0]
        );
      }
      if (statement) {
        const stmt = statement.trim().replace(/\s+/g, " ");
        return stmt.length > 80 ? stmt.substring(0, 80) + "..." : stmt;
      }
      return "-";
    }
    case TaskRunLogEntry_Type.TRANSACTION_CONTROL: {
      const txn = entry.transactionControl;
      if (!txn) return "";
      const typeLabels: Record<number, string> = {
        [TaskRunLogEntry_TransactionControl_Type.BEGIN]: "BEGIN",
        [TaskRunLogEntry_TransactionControl_Type.COMMIT]: "COMMIT",
        [TaskRunLogEntry_TransactionControl_Type.ROLLBACK]: "ROLLBACK",
      };
      const typeStr = typeLabels[txn.type] ?? "";
      return txn.error ? `${typeStr} error: ${txn.error}` : typeStr;
    }
    case TaskRunLogEntry_Type.SCHEMA_DUMP: {
      const dump = entry.schemaDump;
      if (!dump) return "";
      if (dump.error) return dump.error;
      if (dump.startTime && dump.endTime) return "completed";
      return t("task-run.log-detail.dumping");
    }
    case TaskRunLogEntry_Type.DATABASE_SYNC: {
      const sync = entry.databaseSync;
      if (!sync) return "";
      if (sync.error) return sync.error;
      if (sync.startTime && sync.endTime) return "completed";
      return t("task-run.log-detail.syncing");
    }
    case TaskRunLogEntry_Type.TASK_RUN_STATUS_UPDATE: {
      const status = entry.taskRunStatusUpdate;
      if (!status) return "";
      switch (status.status) {
        case TaskRunLogEntry_TaskRunStatusUpdate_Status.RUNNING_WAITING:
          return t("task-run.log-detail.waiting-to-execute");
        case TaskRunLogEntry_TaskRunStatusUpdate_Status.RUNNING_RUNNING:
          return t("task-run.log-detail.executing");
        default:
          return t("task-run.log-detail.status-changed");
      }
    }
    case TaskRunLogEntry_Type.PRIOR_BACKUP: {
      const backup = entry.priorBackup;
      if (!backup) return "";
      if (backup.error) return backup.error;
      const items = backup.priorBackupDetail?.items;
      if (items?.length) {
        return t("task-run.log-detail.backup-completed", {
          count: items.length,
        });
      }
      if (backup.startTime && backup.endTime) return "completed";
      return t("task-run.log-detail.backing-up");
    }
    case TaskRunLogEntry_Type.RETRY_INFO: {
      const retry = entry.retryInfo;
      if (!retry) return "";
      const attempt = t("task-run.log-detail.retry-attempt", {
        current: retry.retryCount,
        max: retry.maximumRetries,
      });
      return retry.error ? `${attempt} - ${retry.error}` : attempt;
    }
    case TaskRunLogEntry_Type.COMPUTE_DIFF: {
      const diff = entry.computeDiff;
      if (!diff) return "";
      if (diff.error) return diff.error;
      if (diff.startTime && diff.endTime) return "completed";
      return t("task-run.log-detail.computing");
    }
    default:
      return "";
  }
};

// Group consecutive entries of the same type into sections (ordered by time)
const sections = computed((): Section[] => {
  if (!props.entries.length) return [];

  // Sort entries by time first
  const sortedEntries = [...props.entries].sort((a, b) => {
    return getTimestampMs(a.logTime) - getTimestampMs(b.logTime);
  });

  // Group consecutive entries of the same type
  const groups: { type: TaskRunLogEntry_Type; entries: TaskRunLogEntry[] }[] =
    [];
  let currentGroup: {
    type: TaskRunLogEntry_Type;
    entries: TaskRunLogEntry[];
  } | null = null;

  for (const entry of sortedEntries) {
    if (currentGroup && currentGroup.type === entry.type) {
      // Same type as current group - add to it
      currentGroup.entries.push(entry);
    } else {
      // Different type - start a new group
      currentGroup = { type: entry.type, entries: [entry] };
      groups.push(currentGroup);
    }
  }

  // Build sections from groups
  const result: Section[] = groups.map((group, groupIdx) => {
    const { type, entries } = group;

    // Calculate section status
    const hasAnyError = entries.some(hasError);
    const allComplete = entries.every(isComplete);
    let status: SectionStatus = "pending";
    if (hasAnyError) {
      status = "error";
    } else if (allComplete) {
      status = "success";
    } else {
      status = "running";
    }

    // Calculate duration
    const timestamps = entries
      .map((e) => getTimestampMs(e.logTime))
      .filter((t) => t > 0);
    const startTime = timestamps.length ? Math.min(...timestamps) : 0;
    const endTime = timestamps.length ? Math.max(...timestamps) : 0;
    const durationMs = endTime - startTime;

    // Build display items
    const items: DisplayItem[] = entries.map((entry, idx) => {
      const entryTime = getTimestampMs(entry.logTime);
      const relativeMs = startTime > 0 ? entryTime - startTime : 0;
      const entryHasError = hasError(entry);

      return {
        key: `${groupIdx}-${idx}`,
        time: formatTime(entry.logTime),
        relativeTime: relativeMs > 0 ? formatRelativeTime(relativeMs) : "",
        levelIndicator: entryHasError ? "\u2717" : "\u2713",
        levelClass: entryHasError ? "text-red-600" : "text-green-600",
        detail: getEntryDetail(entry),
        detailClass: entryHasError ? "text-red-700" : "text-gray-600",
        affectedRows:
          entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE
            ? Number(entry.commandExecute?.response?.affectedRows ?? 0) ||
              undefined
            : undefined,
      };
    });

    const statusCfg = STATUS_CONFIG[status];
    return {
      id: `section-${groupIdx}`,
      type,
      label: getSectionLabel(type),
      status,
      statusIcon: statusCfg.icon,
      statusClass: statusCfg.class,
      duration: durationMs > 0 ? formatDuration(durationMs) : "",
      entryCount: entries.length,
      items,
    };
  });

  return result;
});

// Auto-expand error sections when sections change
watch(
  sections,
  (newSections) => {
    for (const section of newSections) {
      if (section.status === "error") {
        expandedSections.value.add(section.id);
      }
    }
  },
  { immediate: true }
);
</script>
