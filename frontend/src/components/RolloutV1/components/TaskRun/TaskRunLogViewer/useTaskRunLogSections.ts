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
  TaskRunLogEntry_TransactionControl_Type,
  TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import { addToSet, deleteFromSet } from "../../shared/reactivity";
import type {
  DisplayItem,
  ReleaseFileGroup,
  ReplicaGroup,
  Section,
  SectionStatus,
} from "./types";
import {
  formatDuration,
  formatRelativeTime,
  formatTime,
  getTimestampMs,
  getUniqueReplicaIds,
  groupEntriesByReleaseFile,
  groupEntriesByReplica,
  groupEntriesByType,
  hasError,
  hasReleaseFileMarkers,
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
  hasMultipleReplicas: ComputedRef<boolean>;
  hasReleaseFiles: ComputedRef<boolean>;
  releaseFileGroups: ComputedRef<ReleaseFileGroup[]>;
  replicaGroups: ComputedRef<ReplicaGroup[]>;
  expandedSections: Ref<Set<string>>;
  expandedReplicas: Ref<Set<string>>;
  expandedReleaseFiles: Ref<Set<string>>;
  toggleSection: (sectionId: string) => void;
  toggleReplica: (replicaId: string) => void;
  toggleReleaseFile: (fileId: string) => void;
  isSectionExpanded: (sectionId: string) => boolean;
  isReplicaExpanded: (replicaId: string) => boolean;
  isReleaseFileExpanded: (fileId: string) => boolean;
  expandAll: () => void;
  collapseAll: () => void;
  areAllExpanded: ComputedRef<boolean>;
  totalSections: ComputedRef<number>;
  totalEntries: ComputedRef<number>;
  totalDuration: ComputedRef<string>;
}

export const useTaskRunLogSections = (
  entries: MaybeRefOrGetter<TaskRunLogEntry[]>,
  sheet: MaybeRefOrGetter<Sheet | undefined>,
  sheetsMap: MaybeRefOrGetter<Map<string, Sheet> | undefined> = () => undefined
): UseTaskRunLogSectionsReturn => {
  const { t } = useI18n();

  const expandedSections = ref<Set<string>>(new Set());
  const userCollapsedSections = ref<Set<string>>(new Set());
  const expandedReplicas = ref<Set<string>>(new Set());
  const userCollapsedReplicas = ref<Set<string>>(new Set());
  const expandedReleaseFiles = ref<Set<string>>(new Set());
  const userCollapsedReleaseFiles = ref<Set<string>>(new Set());

  const getSectionLabel = (type: TaskRunLogEntry_Type): string => {
    const labelMap: Partial<Record<TaskRunLogEntry_Type, string>> = {
      [TaskRunLogEntry_Type.SCHEMA_DUMP]: t("task-run.log-type.schema-dump"),
      [TaskRunLogEntry_Type.COMMAND_EXECUTE]: t(
        "task-run.log-type.command-execute"
      ),
      [TaskRunLogEntry_Type.DATABASE_SYNC]: t(
        "task-run.log-type.database-sync"
      ),
      [TaskRunLogEntry_Type.TRANSACTION_CONTROL]: t(
        "task-run.log-type.transaction"
      ),
      [TaskRunLogEntry_Type.PRIOR_BACKUP]: t("task-run.log-type.prior-backup"),
      [TaskRunLogEntry_Type.RETRY_INFO]: t("task-run.log-type.retry"),
      [TaskRunLogEntry_Type.COMPUTE_DIFF]: t("task-run.log-type.compute-diff"),
      [TaskRunLogEntry_Type.RELEASE_FILE_EXECUTE]: t(
        "task-run.log-type.release-file-execute"
      ),
    };
    return labelMap[type] ?? "Unknown";
  };

  // Extract statement from sheet content using byte range
  const extractStatementFromRange = (
    range: { start: number; end: number },
    sheetContent: Uint8Array | undefined,
    sheetsMapValue: Map<string, Sheet> | undefined,
    fileVersion?: string
  ): string | undefined => {
    // For release tasks, look up sheet by version from sheetsMap
    // For non-release tasks, use the single sheet
    let content = sheetContent;
    if (fileVersion && sheetsMapValue) {
      const fileSheet = sheetsMapValue.get(fileVersion);
      if (fileSheet?.content) {
        content = fileSheet.content;
      }
    }
    if (!content) return undefined;

    const subarray = content.subarray(range.start, range.end);
    return new TextDecoder().decode(subarray);
  };

  const getEntryDetail = (
    entry: TaskRunLogEntry,
    sheetContent: Uint8Array | undefined,
    sheetsMapValue: Map<string, Sheet> | undefined,
    fileVersion?: string
  ): string => {
    switch (entry.type) {
      case TaskRunLogEntry_Type.COMMAND_EXECUTE: {
        const cmd = entry.commandExecute;
        if (!cmd) return "";
        if (cmd.response?.error) return cmd.response.error;

        let statement: string | undefined;
        if (cmd.statement) {
          statement = cmd.statement;
        } else if (cmd.range) {
          statement = extractStatementFromRange(
            cmd.range,
            sheetContent,
            sheetsMapValue,
            fileVersion
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
      case TaskRunLogEntry_Type.RELEASE_FILE_EXECUTE: {
        const rfe = entry.releaseFileExecute;
        if (!rfe) return "";
        if (rfe.filePath) {
          return `${rfe.version}: ${rfe.filePath}`;
        }
        return rfe.version;
      }
      default:
        return "";
    }
  };

  const getCommandDuration = (entry: TaskRunLogEntry): string | undefined => {
    if (entry.type !== TaskRunLogEntry_Type.COMMAND_EXECUTE) return undefined;
    const cmd = entry.commandExecute;
    if (!cmd?.logTime || !cmd.response?.logTime) return undefined;
    const startMs = getTimestampMs(cmd.logTime);
    const endMs = getTimestampMs(cmd.response.logTime);
    if (startMs <= 0 || endMs <= 0) return undefined;
    const durationMs = endMs - startMs;
    return formatDuration(durationMs);
  };

  const buildDisplayItems = (
    groupEntries: TaskRunLogEntry[],
    groupIdx: number,
    startTime: number,
    sheetContent: Uint8Array | undefined,
    sheetsMapValue: Map<string, Sheet> | undefined,
    fileVersion?: string
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
        detail: getEntryDetail(
          entry,
          sheetContent,
          sheetsMapValue,
          fileVersion
        ),
        detailClass: entryHasError ? "text-red-700" : "text-gray-600",
        affectedRows:
          entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE
            ? Number(entry.commandExecute?.response?.affectedRows ?? 0) ||
              undefined
            : undefined,
        duration: getCommandDuration(entry),
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

  // Get the actual start and end times for an entry based on its type
  const getEntryTimeRange = (
    entry: TaskRunLogEntry
  ): { start: number; end: number } => {
    switch (entry.type) {
      case TaskRunLogEntry_Type.COMMAND_EXECUTE: {
        const cmd = entry.commandExecute;
        const start = getTimestampMs(cmd?.logTime);
        const end = getTimestampMs(cmd?.response?.logTime);
        return { start: start || 0, end: end || start || 0 };
      }
      case TaskRunLogEntry_Type.SCHEMA_DUMP: {
        const dump = entry.schemaDump;
        return {
          start: getTimestampMs(dump?.startTime),
          end: getTimestampMs(dump?.endTime),
        };
      }
      case TaskRunLogEntry_Type.DATABASE_SYNC: {
        const sync = entry.databaseSync;
        return {
          start: getTimestampMs(sync?.startTime),
          end: getTimestampMs(sync?.endTime),
        };
      }
      case TaskRunLogEntry_Type.PRIOR_BACKUP: {
        const backup = entry.priorBackup;
        return {
          start: getTimestampMs(backup?.startTime),
          end: getTimestampMs(backup?.endTime),
        };
      }
      case TaskRunLogEntry_Type.COMPUTE_DIFF: {
        const diff = entry.computeDiff;
        return {
          start: getTimestampMs(diff?.startTime),
          end: getTimestampMs(diff?.endTime),
        };
      }
      default: {
        const time = getTimestampMs(entry.logTime);
        return { start: time, end: time };
      }
    }
  };

  const buildSectionsFromEntries = (
    entriesList: TaskRunLogEntry[],
    sheetContent: Uint8Array | undefined,
    sheetsMapValue: Map<string, Sheet> | undefined,
    idPrefix = "",
    forceError = false,
    fileVersion?: string
  ): Section[] => {
    if (!entriesList.length) return [];

    const groups = groupEntriesByType(entriesList);

    return groups.map((group, groupIdx) => {
      const { type, entries: groupEntries } = group;

      // For interrupted replicas, treat "running" as "error"
      let status = calculateSectionStatus(groupEntries);
      if (forceError && status === "running") {
        status = "error";
      }

      // Calculate duration using actual operation start/end times
      const timeRanges = groupEntries.map(getEntryTimeRange);
      const startTimes = timeRanges.map((r) => r.start).filter((t) => t > 0);
      const endTimes = timeRanges.map((r) => r.end).filter((t) => t > 0);
      const startTime = startTimes.length ? Math.min(...startTimes) : 0;
      const endTime = endTimes.length ? Math.max(...endTimes) : 0;
      const durationMs = endTime - startTime;

      const items = buildDisplayItems(
        groupEntries,
        groupIdx,
        startTime,
        sheetContent,
        sheetsMapValue,
        fileVersion
      );
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
        duration:
          startTime > 0 && endTime > 0 ? formatDuration(durationMs) : "",
        entryCount: groupEntries.length,
        items,
      };
    });
  };

  const sections = computed((): Section[] => {
    const entriesValue = toValue(entries);
    const sheetContent = toValue(sheet)?.content;
    const sheetsMapValue = toValue(sheetsMap);
    return buildSectionsFromEntries(entriesValue, sheetContent, sheetsMapValue);
  });

  const hasMultipleReplicas = computed((): boolean => {
    const entriesValue = toValue(entries);
    const replicaIds = getUniqueReplicaIds(entriesValue);
    return replicaIds.length > 1;
  });

  const hasReleaseFiles = computed((): boolean => {
    const entriesValue = toValue(entries);
    return hasReleaseFileMarkers(entriesValue);
  });

  // Build release file groups from entries
  const buildReleaseFileGroupsFromEntries = (
    entriesList: TaskRunLogEntry[],
    sheetContent: Uint8Array | undefined,
    sheetsMapValue: Map<string, Sheet> | undefined,
    idPrefix = "",
    forceError = false
  ): ReleaseFileGroup[] => {
    const fileGroups = groupEntriesByReleaseFile(entriesList);

    return fileGroups
      .filter((group) => group.file !== null) // Only include groups with file markers
      .map((group, idx) => {
        const fileId = idPrefix ? `${idPrefix}-file-${idx}` : `file-${idx}`;
        const version = group.file!.version;
        return {
          version,
          filePath: group.file!.filePath,
          // Pass version so COMMAND_EXECUTE entries can look up the correct sheet
          sections: buildSectionsFromEntries(
            group.entries,
            sheetContent,
            sheetsMapValue,
            fileId,
            forceError,
            version
          ),
        };
      });
  };

  // Get orphan sections (entries before any release file marker)
  const getOrphanSections = (
    entriesList: TaskRunLogEntry[],
    sheetContent: Uint8Array | undefined,
    sheetsMapValue: Map<string, Sheet> | undefined,
    idPrefix = "",
    forceError = false
  ): Section[] => {
    const fileGroups = groupEntriesByReleaseFile(entriesList);
    const orphanGroup = fileGroups.find((group) => group.file === null);
    if (!orphanGroup || orphanGroup.entries.length === 0) {
      return [];
    }
    return buildSectionsFromEntries(
      orphanGroup.entries,
      sheetContent,
      sheetsMapValue,
      idPrefix ? `${idPrefix}-orphan` : "orphan",
      forceError
    );
  };

  // Release file groups for single-replica view
  const releaseFileGroups = computed((): ReleaseFileGroup[] => {
    const entriesValue = toValue(entries);
    if (!hasReleaseFileMarkers(entriesValue)) return [];
    const sheetContent = toValue(sheet)?.content;
    const sheetsMapValue = toValue(sheetsMap);
    return buildReleaseFileGroupsFromEntries(
      entriesValue,
      sheetContent,
      sheetsMapValue
    );
  });

  const replicaGroups = computed((): ReplicaGroup[] => {
    const entriesValue = toValue(entries);
    if (!entriesValue.length) return [];

    const replicaIds = getUniqueReplicaIds(entriesValue);
    if (replicaIds.length <= 1) return [];

    const sheetContent = toValue(sheet)?.content;
    const sheetsMapValue = toValue(sheetsMap);
    const entriesByReplica = groupEntriesByReplica(entriesValue);

    return replicaIds.map((replicaId, idx) => {
      const replicaEntries = entriesByReplica.get(replicaId) || [];
      const isLatestReplica = idx === replicaIds.length - 1;
      const forceError = !isLatestReplica;

      // Check if this replica has release file markers
      const hasFiles = hasReleaseFileMarkers(replicaEntries);

      if (hasFiles) {
        return {
          replicaId,
          releaseFileGroups: buildReleaseFileGroupsFromEntries(
            replicaEntries,
            sheetContent,
            sheetsMapValue,
            replicaId,
            forceError
          ),
          sections: getOrphanSections(
            replicaEntries,
            sheetContent,
            sheetsMapValue,
            replicaId,
            forceError
          ),
        };
      }

      return {
        replicaId,
        releaseFileGroups: [],
        sections: buildSectionsFromEntries(
          replicaEntries,
          sheetContent,
          sheetsMapValue,
          replicaId,
          forceError
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

  const toggleReplica = (replicaId: string) => {
    if (expandedReplicas.value.has(replicaId)) {
      deleteFromSet(expandedReplicas, replicaId);
      addToSet(userCollapsedReplicas, replicaId);
    } else {
      addToSet(expandedReplicas, replicaId);
      deleteFromSet(userCollapsedReplicas, replicaId);
    }
  };

  const isSectionExpanded = (sectionId: string): boolean => {
    return expandedSections.value.has(sectionId);
  };

  const isReplicaExpanded = (replicaId: string): boolean => {
    return expandedReplicas.value.has(replicaId);
  };

  const toggleReleaseFile = (fileId: string) => {
    if (expandedReleaseFiles.value.has(fileId)) {
      deleteFromSet(expandedReleaseFiles, fileId);
      addToSet(userCollapsedReleaseFiles, fileId);
    } else {
      addToSet(expandedReleaseFiles, fileId);
      deleteFromSet(userCollapsedReleaseFiles, fileId);
    }
  };

  const isReleaseFileExpanded = (fileId: string): boolean => {
    return expandedReleaseFiles.value.has(fileId);
  };

  // Get all section IDs (from flat sections, release file groups, and replica groups)
  const getAllSectionIds = (): string[] => {
    if (hasMultipleReplicas.value) {
      return replicaGroups.value.flatMap((group) => {
        const orphanSectionIds = group.sections.map((s) => s.id);
        const fileSectionIds = group.releaseFileGroups.flatMap((fg) =>
          fg.sections.map((s) => s.id)
        );
        return [...orphanSectionIds, ...fileSectionIds];
      });
    }
    if (hasReleaseFiles.value) {
      return releaseFileGroups.value.flatMap((fg) =>
        fg.sections.map((s) => s.id)
      );
    }
    return sections.value.map((section) => section.id);
  };

  // Get all replica IDs
  const getAllReplicaIds = (): string[] => {
    return replicaGroups.value.map((group) => group.replicaId);
  };

  // Get all release file IDs
  const getAllReleaseFileIds = (): string[] => {
    if (hasMultipleReplicas.value) {
      return replicaGroups.value.flatMap((group) =>
        group.releaseFileGroups.map(
          (_, fileIdx) => `${group.replicaId}-file-${fileIdx}`
        )
      );
    }
    return releaseFileGroups.value.map((_, idx) => `file-${idx}`);
  };

  const expandAll = () => {
    // Expand all sections
    for (const sectionId of getAllSectionIds()) {
      addToSet(expandedSections, sectionId);
      deleteFromSet(userCollapsedSections, sectionId);
    }
    // Expand all replica groups (for multi-replica view)
    if (hasMultipleReplicas.value) {
      for (const replicaId of getAllReplicaIds()) {
        addToSet(expandedReplicas, replicaId);
        deleteFromSet(userCollapsedReplicas, replicaId);
      }
    }
    // Expand all release file groups
    for (const fileId of getAllReleaseFileIds()) {
      addToSet(expandedReleaseFiles, fileId);
      deleteFromSet(userCollapsedReleaseFiles, fileId);
    }
  };

  const collapseAll = () => {
    // Collapse all sections
    for (const sectionId of getAllSectionIds()) {
      deleteFromSet(expandedSections, sectionId);
      addToSet(userCollapsedSections, sectionId);
    }
    // Collapse all replica groups (for multi-replica view)
    if (hasMultipleReplicas.value) {
      for (const replicaId of getAllReplicaIds()) {
        deleteFromSet(expandedReplicas, replicaId);
        addToSet(userCollapsedReplicas, replicaId);
      }
    }
    // Collapse all release file groups
    for (const fileId of getAllReleaseFileIds()) {
      deleteFromSet(expandedReleaseFiles, fileId);
      addToSet(userCollapsedReleaseFiles, fileId);
    }
  };

  const areAllExpanded = computed((): boolean => {
    const allSectionIds = getAllSectionIds();
    if (allSectionIds.length === 0) return false;

    const allSectionsExpanded = allSectionIds.every((id) =>
      expandedSections.value.has(id)
    );

    // Check release file groups
    const allReleaseFileIds = getAllReleaseFileIds();
    const allReleaseFilesExpanded =
      allReleaseFileIds.length === 0 ||
      allReleaseFileIds.every((id) => expandedReleaseFiles.value.has(id));

    // For multi-replica view, also check replica groups
    if (hasMultipleReplicas.value) {
      const allReplicaIds = getAllReplicaIds();
      const allReplicasExpanded = allReplicaIds.every((id) =>
        expandedReplicas.value.has(id)
      );
      return (
        allSectionsExpanded && allReplicasExpanded && allReleaseFilesExpanded
      );
    }

    return allSectionsExpanded && allReleaseFilesExpanded;
  });

  const totalSections = computed((): number => {
    if (hasMultipleReplicas.value) {
      return replicaGroups.value.reduce((sum, group) => {
        const orphanCount = group.sections.length;
        const fileCount = group.releaseFileGroups.reduce(
          (fSum, fg) => fSum + fg.sections.length,
          0
        );
        return sum + orphanCount + fileCount;
      }, 0);
    }
    if (hasReleaseFiles.value) {
      return releaseFileGroups.value.reduce(
        (sum, fg) => sum + fg.sections.length,
        0
      );
    }
    return sections.value.length;
  });

  const totalEntries = computed((): number => {
    if (hasMultipleReplicas.value) {
      return replicaGroups.value.reduce((sum, group) => {
        const orphanEntries = group.sections.reduce(
          (sSum, section) => sSum + section.entryCount,
          0
        );
        const fileEntries = group.releaseFileGroups.reduce(
          (fSum, fg) =>
            fSum + fg.sections.reduce((sSum, s) => sSum + s.entryCount, 0),
          0
        );
        return sum + orphanEntries + fileEntries;
      }, 0);
    }
    if (hasReleaseFiles.value) {
      return releaseFileGroups.value.reduce(
        (sum, fg) =>
          sum + fg.sections.reduce((sSum, s) => sSum + s.entryCount, 0),
        0
      );
    }
    return sections.value.reduce((sum, section) => sum + section.entryCount, 0);
  });

  const totalDuration = computed((): string => {
    const entriesValue = toValue(entries);
    if (!entriesValue.length) return "";

    const timeRanges = entriesValue.map(getEntryTimeRange);
    const startTimes = timeRanges.map((r) => r.start).filter((t) => t > 0);
    const endTimes = timeRanges.map((r) => r.end).filter((t) => t > 0);

    if (!startTimes.length || !endTimes.length) return "";

    const startTime = Math.min(...startTimes);
    const endTime = Math.max(...endTimes);
    const durationMs = endTime - startTime;

    return formatDuration(durationMs);
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

  // Auto-expand all replicas and error sections in replica groups
  watch(
    replicaGroups,
    (newReplicaGroups) => {
      for (const replicaGroup of newReplicaGroups) {
        // Auto-expand all replica groups by default
        if (!userCollapsedReplicas.value.has(replicaGroup.replicaId)) {
          addToSet(expandedReplicas, replicaGroup.replicaId);
        }
        // Auto-expand error sections within replica groups (orphan sections)
        for (const section of replicaGroup.sections) {
          if (
            section.status === "error" &&
            !userCollapsedSections.value.has(section.id)
          ) {
            addToSet(expandedSections, section.id);
          }
        }
        // Auto-expand release file groups and their error sections
        replicaGroup.releaseFileGroups.forEach((fg, fileIdx) => {
          const fileId = `${replicaGroup.replicaId}-file-${fileIdx}`;
          // Auto-expand all release file groups by default
          if (!userCollapsedReleaseFiles.value.has(fileId)) {
            addToSet(expandedReleaseFiles, fileId);
          }
          // Auto-expand error sections within file groups
          for (const section of fg.sections) {
            if (
              section.status === "error" &&
              !userCollapsedSections.value.has(section.id)
            ) {
              addToSet(expandedSections, section.id);
            }
          }
        });
      }
    },
    { immediate: true }
  );

  // Auto-expand release file groups (single-replica view)
  watch(
    releaseFileGroups,
    (newReleaseFileGroups) => {
      newReleaseFileGroups.forEach((fg, idx) => {
        const fileId = `file-${idx}`;
        // Auto-expand all release file groups by default
        if (!userCollapsedReleaseFiles.value.has(fileId)) {
          addToSet(expandedReleaseFiles, fileId);
        }
        // Auto-expand error sections within file groups
        for (const section of fg.sections) {
          if (
            section.status === "error" &&
            !userCollapsedSections.value.has(section.id)
          ) {
            addToSet(expandedSections, section.id);
          }
        }
      });
    },
    { immediate: true }
  );

  return {
    sections,
    hasMultipleReplicas,
    hasReleaseFiles,
    releaseFileGroups,
    replicaGroups,
    expandedSections,
    expandedReplicas,
    expandedReleaseFiles,
    toggleSection,
    toggleReplica,
    toggleReleaseFile,
    isSectionExpanded,
    isReplicaExpanded,
    isReleaseFileExpanded,
    expandAll,
    collapseAll,
    areAllExpanded,
    totalSections,
    totalEntries,
    totalDuration,
  };
};
