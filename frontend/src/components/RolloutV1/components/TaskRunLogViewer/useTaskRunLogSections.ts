import {
  CheckCircle2Icon,
  CircleIcon,
  LoaderCircleIcon,
  XCircleIcon,
} from "lucide-vue-next";
import type { Component, ComputedRef, MaybeRefOrGetter, Ref } from "vue";
import { computed, ref, toValue, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { TaskRunLogEntry } from "@/types/proto-es/v1/rollout_service_pb";
import {
  TaskRunLogEntry_TaskRunStatusUpdate_Status,
  TaskRunLogEntry_TransactionControl_Type,
  TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import { addToSet, deleteFromSet } from "../utils/reactivity";
import type { DeployGroup, DisplayItem, Section, SectionStatus } from "./types";
import {
  formatDuration,
  formatRelativeTime,
  formatTime,
  getTimestampMs,
  getUniqueDeployIds,
  groupEntriesByDeploy,
  groupEntriesByType,
  hasError,
  isComplete,
} from "./utils";

const STATUS_CONFIG: Record<SectionStatus, { icon: Component; class: string }> =
  {
    success: { icon: CheckCircle2Icon, class: "text-green-600" },
    error: { icon: XCircleIcon, class: "text-red-600" },
    running: { icon: LoaderCircleIcon, class: "text-blue-600" },
    pending: { icon: CircleIcon, class: "text-gray-400" },
  };

export interface UseTaskRunLogSectionsReturn {
  sections: ComputedRef<Section[]>;
  hasMultipleDeploys: ComputedRef<boolean>;
  deployGroups: ComputedRef<DeployGroup[]>;
  expandedSections: Ref<Set<string>>;
  expandedDeploys: Ref<Set<string>>;
  toggleSection: (sectionId: string) => void;
  toggleDeploy: (deployId: string) => void;
  isSectionExpanded: (sectionId: string) => boolean;
  isDeployExpanded: (deployId: string) => boolean;
  expandAll: () => void;
  collapseAll: () => void;
  areAllExpanded: ComputedRef<boolean>;
  totalSections: ComputedRef<number>;
  totalEntries: ComputedRef<number>;
}

export const useTaskRunLogSections = (
  entries: MaybeRefOrGetter<TaskRunLogEntry[]>,
  sheet: MaybeRefOrGetter<Sheet | undefined>
): UseTaskRunLogSectionsReturn => {
  const { t } = useI18n();

  const expandedSections = ref<Set<string>>(new Set());
  const userCollapsedSections = ref<Set<string>>(new Set());
  const expandedDeploys = ref<Set<string>>(new Set());
  const userCollapsedDeploys = ref<Set<string>>(new Set());

  const getSectionLabel = (type: TaskRunLogEntry_Type): string => {
    const labelMap: Partial<Record<TaskRunLogEntry_Type, string>> = {
      [TaskRunLogEntry_Type.SCHEMA_DUMP]: t("task-run.log-type.schema-dump"),
      [TaskRunLogEntry_Type.COMMAND_EXECUTE]: t(
        "task-run.log-type.command-execute"
      ),
      [TaskRunLogEntry_Type.DATABASE_SYNC]: t(
        "task-run.log-type.database-sync"
      ),
      [TaskRunLogEntry_Type.TASK_RUN_STATUS_UPDATE]: t(
        "task-run.log-type.status-update"
      ),
      [TaskRunLogEntry_Type.TRANSACTION_CONTROL]: t(
        "task-run.log-type.transaction"
      ),
      [TaskRunLogEntry_Type.PRIOR_BACKUP]: t("task-run.log-type.prior-backup"),
      [TaskRunLogEntry_Type.RETRY_INFO]: t("task-run.log-type.retry"),
      [TaskRunLogEntry_Type.COMPUTE_DIFF]: t("task-run.log-type.compute-diff"),
    };
    return labelMap[type] ?? "Unknown";
  };

  const getEntryDetail = (entry: TaskRunLogEntry): string => {
    switch (entry.type) {
      case TaskRunLogEntry_Type.COMMAND_EXECUTE: {
        const cmd = entry.commandExecute;
        if (!cmd) return "";
        if (cmd.response?.error) return cmd.response.error;

        let statement: string | undefined;
        if (cmd.statement) {
          statement = cmd.statement;
        } else if (cmd.range) {
          const sheetValue = toValue(sheet);
          if (sheetValue) {
            const subarray = sheetValue.content.subarray(
              cmd.range.start,
              cmd.range.end
            );
            statement = new TextDecoder().decode(subarray);
          }
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

  const buildDisplayItems = (
    groupEntries: TaskRunLogEntry[],
    groupIdx: number,
    startTime: number
  ): DisplayItem[] => {
    return groupEntries.map((entry, idx) => {
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
  };

  const calculateSectionStatus = (
    groupEntries: TaskRunLogEntry[]
  ): SectionStatus => {
    const hasAnyError = groupEntries.some(hasError);
    const allComplete = groupEntries.every(isComplete);

    if (hasAnyError) return "error";
    if (allComplete) return "success";
    return "running";
  };

  const buildSectionsFromEntries = (
    entriesList: TaskRunLogEntry[],
    idPrefix = "",
    forceError = false
  ): Section[] => {
    if (!entriesList.length) return [];

    const groups = groupEntriesByType(entriesList);

    return groups.map((group, groupIdx) => {
      const { type, entries: groupEntries } = group;

      // For interrupted deployments, treat "running" as "error"
      let status = calculateSectionStatus(groupEntries);
      if (forceError && status === "running") {
        status = "error";
      }

      const timestamps = groupEntries
        .map((e) => getTimestampMs(e.logTime))
        .filter((t) => t > 0);
      const startTime = timestamps.length ? Math.min(...timestamps) : 0;
      const endTime = timestamps.length ? Math.max(...timestamps) : 0;
      const durationMs = endTime - startTime;

      const items = buildDisplayItems(groupEntries, groupIdx, startTime);
      const statusCfg = STATUS_CONFIG[status];

      return {
        id: idPrefix
          ? `${idPrefix}-section-${groupIdx}`
          : `section-${groupIdx}`,
        type,
        label: getSectionLabel(type),
        status,
        statusIcon: statusCfg.icon,
        statusClass: statusCfg.class,
        duration: durationMs > 0 ? formatDuration(durationMs) : "",
        entryCount: groupEntries.length,
        items,
      };
    });
  };

  const sections = computed((): Section[] => {
    const entriesValue = toValue(entries);
    return buildSectionsFromEntries(entriesValue);
  });

  const hasMultipleDeploys = computed((): boolean => {
    const entriesValue = toValue(entries);
    const deployIds = getUniqueDeployIds(entriesValue);
    return deployIds.length > 1;
  });

  const deployGroups = computed((): DeployGroup[] => {
    const entriesValue = toValue(entries);
    if (!entriesValue.length) return [];

    const deployIds = getUniqueDeployIds(entriesValue);
    if (deployIds.length <= 1) return [];

    const entriesByDeploy = groupEntriesByDeploy(entriesValue);

    return deployIds.map((deployId, idx) => {
      const deployEntries = entriesByDeploy.get(deployId) || [];
      const isLatestDeploy = idx === deployIds.length - 1;
      return {
        deployId,
        sections: buildSectionsFromEntries(
          deployEntries,
          deployId,
          !isLatestDeploy // forceError: true for non-latest deploys
        ),
      };
    });
  });

  const toggleSection = (sectionId: string) => {
    if (expandedSections.value.has(sectionId)) {
      deleteFromSet(expandedSections, sectionId);
      addToSet(userCollapsedSections, sectionId);
    } else {
      addToSet(expandedSections, sectionId);
      deleteFromSet(userCollapsedSections, sectionId);
    }
  };

  const toggleDeploy = (deployId: string) => {
    if (expandedDeploys.value.has(deployId)) {
      deleteFromSet(expandedDeploys, deployId);
      addToSet(userCollapsedDeploys, deployId);
    } else {
      addToSet(expandedDeploys, deployId);
      deleteFromSet(userCollapsedDeploys, deployId);
    }
  };

  const isSectionExpanded = (sectionId: string): boolean => {
    return expandedSections.value.has(sectionId);
  };

  const isDeployExpanded = (deployId: string): boolean => {
    return expandedDeploys.value.has(deployId);
  };

  // Get all section IDs (both from flat sections and deploy groups)
  const getAllSectionIds = (): string[] => {
    if (hasMultipleDeploys.value) {
      return deployGroups.value.flatMap((group) =>
        group.sections.map((section) => section.id)
      );
    }
    return sections.value.map((section) => section.id);
  };

  // Get all deploy IDs
  const getAllDeployIds = (): string[] => {
    return deployGroups.value.map((group) => group.deployId);
  };

  const expandAll = () => {
    // Expand all sections
    for (const sectionId of getAllSectionIds()) {
      addToSet(expandedSections, sectionId);
      deleteFromSet(userCollapsedSections, sectionId);
    }
    // Expand all deploy groups (for multi-deploy view)
    if (hasMultipleDeploys.value) {
      for (const deployId of getAllDeployIds()) {
        addToSet(expandedDeploys, deployId);
        deleteFromSet(userCollapsedDeploys, deployId);
      }
    }
  };

  const collapseAll = () => {
    // Collapse all sections
    for (const sectionId of getAllSectionIds()) {
      deleteFromSet(expandedSections, sectionId);
      addToSet(userCollapsedSections, sectionId);
    }
    // Collapse all deploy groups (for multi-deploy view)
    if (hasMultipleDeploys.value) {
      for (const deployId of getAllDeployIds()) {
        deleteFromSet(expandedDeploys, deployId);
        addToSet(userCollapsedDeploys, deployId);
      }
    }
  };

  const areAllExpanded = computed((): boolean => {
    const allSectionIds = getAllSectionIds();
    if (allSectionIds.length === 0) return false;

    const allSectionsExpanded = allSectionIds.every((id) =>
      expandedSections.value.has(id)
    );

    // For multi-deploy view, also check deploy groups
    if (hasMultipleDeploys.value) {
      const allDeployIds = getAllDeployIds();
      const allDeploysExpanded = allDeployIds.every((id) =>
        expandedDeploys.value.has(id)
      );
      return allSectionsExpanded && allDeploysExpanded;
    }

    return allSectionsExpanded;
  });

  const totalSections = computed((): number => {
    if (hasMultipleDeploys.value) {
      return deployGroups.value.reduce(
        (sum, group) => sum + group.sections.length,
        0
      );
    }
    return sections.value.length;
  });

  const totalEntries = computed((): number => {
    if (hasMultipleDeploys.value) {
      return deployGroups.value.reduce(
        (sum, group) =>
          sum +
          group.sections.reduce(
            (sSum, section) => sSum + section.entryCount,
            0
          ),
        0
      );
    }
    return sections.value.reduce((sum, section) => sum + section.entryCount, 0);
  });

  // Auto-expand error sections (respecting user's manual collapse)
  watch(
    sections,
    (newSections) => {
      for (const section of newSections) {
        if (
          section.status === "error" &&
          !userCollapsedSections.value.has(section.id)
        ) {
          addToSet(expandedSections, section.id);
        }
      }
    },
    { immediate: true }
  );

  // Auto-expand all deploys and error sections in deploy groups
  watch(
    deployGroups,
    (newDeployGroups) => {
      for (const deployGroup of newDeployGroups) {
        // Auto-expand all deploy groups by default
        if (!userCollapsedDeploys.value.has(deployGroup.deployId)) {
          addToSet(expandedDeploys, deployGroup.deployId);
        }
        // Auto-expand error sections within deploy groups
        for (const section of deployGroup.sections) {
          if (
            section.status === "error" &&
            !userCollapsedSections.value.has(section.id)
          ) {
            addToSet(expandedSections, section.id);
          }
        }
      }
    },
    { immediate: true }
  );

  return {
    sections,
    hasMultipleDeploys,
    deployGroups,
    expandedSections,
    expandedDeploys,
    toggleSection,
    toggleDeploy,
    isSectionExpanded,
    isDeployExpanded,
    expandAll,
    collapseAll,
    areAllExpanded,
    totalSections,
    totalEntries,
  };
};
