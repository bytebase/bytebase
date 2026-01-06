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
import { addToSet, deleteFromSet } from "../utils/reactivity";
import type {
  DeployGroup,
  DisplayItem,
  ReleaseFileGroup,
  Section,
  SectionStatus,
} from "./types";
import {
  formatDuration,
  formatRelativeTime,
  formatTime,
  getTimestampMs,
  getUniqueDeployIds,
  groupEntriesByDeploy,
  groupEntriesByReleaseFile,
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
  hasMultipleDeploys: ComputedRef<boolean>;
  hasReleaseFiles: ComputedRef<boolean>;
  releaseFileGroups: ComputedRef<ReleaseFileGroup[]>;
  deployGroups: ComputedRef<DeployGroup[]>;
  expandedSections: Ref<Set<string>>;
  expandedDeploys: Ref<Set<string>>;
  expandedReleaseFiles: Ref<Set<string>>;
  toggleSection: (sectionId: string) => void;
  toggleDeploy: (deployId: string) => void;
  toggleReleaseFile: (fileId: string) => void;
  isSectionExpanded: (sectionId: string) => boolean;
  isDeployExpanded: (deployId: string) => boolean;
  isReleaseFileExpanded: (fileId: string) => boolean;
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

  const hasReleaseFiles = computed((): boolean => {
    const entriesValue = toValue(entries);
    return hasReleaseFileMarkers(entriesValue);
  });

  // Build release file groups from entries
  const buildReleaseFileGroupsFromEntries = (
    entriesList: TaskRunLogEntry[],
    idPrefix = "",
    forceError = false
  ): ReleaseFileGroup[] => {
    const fileGroups = groupEntriesByReleaseFile(entriesList);

    return fileGroups
      .filter((group) => group.file !== null) // Only include groups with file markers
      .map((group, idx) => {
        const fileId = idPrefix ? `${idPrefix}-file-${idx}` : `file-${idx}`;
        return {
          version: group.file!.version,
          filePath: group.file!.filePath,
          sections: buildSectionsFromEntries(group.entries, fileId, forceError),
        };
      });
  };

  // Get orphan sections (entries before any release file marker)
  const getOrphanSections = (
    entriesList: TaskRunLogEntry[],
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
      idPrefix ? `${idPrefix}-orphan` : "orphan",
      forceError
    );
  };

  // Release file groups for single-deploy view
  const releaseFileGroups = computed((): ReleaseFileGroup[] => {
    const entriesValue = toValue(entries);
    if (!hasReleaseFileMarkers(entriesValue)) return [];
    return buildReleaseFileGroupsFromEntries(entriesValue);
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
      const forceError = !isLatestDeploy;

      // Check if this deploy has release file markers
      const hasFiles = hasReleaseFileMarkers(deployEntries);

      if (hasFiles) {
        return {
          deployId,
          releaseFileGroups: buildReleaseFileGroupsFromEntries(
            deployEntries,
            deployId,
            forceError
          ),
          sections: getOrphanSections(deployEntries, deployId, forceError),
        };
      }

      return {
        deployId,
        releaseFileGroups: [],
        sections: buildSectionsFromEntries(deployEntries, deployId, forceError),
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

  // Get all section IDs (from flat sections, release file groups, and deploy groups)
  const getAllSectionIds = (): string[] => {
    if (hasMultipleDeploys.value) {
      return deployGroups.value.flatMap((group) => {
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

  // Get all deploy IDs
  const getAllDeployIds = (): string[] => {
    return deployGroups.value.map((group) => group.deployId);
  };

  // Get all release file IDs
  const getAllReleaseFileIds = (): string[] => {
    if (hasMultipleDeploys.value) {
      return deployGroups.value.flatMap((group) =>
        group.releaseFileGroups.map(
          (_, fileIdx) => `${group.deployId}-file-${fileIdx}`
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
    // Expand all deploy groups (for multi-deploy view)
    if (hasMultipleDeploys.value) {
      for (const deployId of getAllDeployIds()) {
        addToSet(expandedDeploys, deployId);
        deleteFromSet(userCollapsedDeploys, deployId);
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
    // Collapse all deploy groups (for multi-deploy view)
    if (hasMultipleDeploys.value) {
      for (const deployId of getAllDeployIds()) {
        deleteFromSet(expandedDeploys, deployId);
        addToSet(userCollapsedDeploys, deployId);
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

    // For multi-deploy view, also check deploy groups
    if (hasMultipleDeploys.value) {
      const allDeployIds = getAllDeployIds();
      const allDeploysExpanded = allDeployIds.every((id) =>
        expandedDeploys.value.has(id)
      );
      return (
        allSectionsExpanded && allDeploysExpanded && allReleaseFilesExpanded
      );
    }

    return allSectionsExpanded && allReleaseFilesExpanded;
  });

  const totalSections = computed((): number => {
    if (hasMultipleDeploys.value) {
      return deployGroups.value.reduce((sum, group) => {
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
    if (hasMultipleDeploys.value) {
      return deployGroups.value.reduce((sum, group) => {
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
        // Auto-expand error sections within deploy groups (orphan sections)
        for (const section of deployGroup.sections) {
          if (
            section.status === "error" &&
            !userCollapsedSections.value.has(section.id)
          ) {
            addToSet(expandedSections, section.id);
          }
        }
        // Auto-expand release file groups and their error sections
        deployGroup.releaseFileGroups.forEach((fg, fileIdx) => {
          const fileId = `${deployGroup.deployId}-file-${fileIdx}`;
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

  // Auto-expand release file groups (single-deploy view)
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
    hasMultipleDeploys,
    hasReleaseFiles,
    releaseFileGroups,
    deployGroups,
    expandedSections,
    expandedDeploys,
    expandedReleaseFiles,
    toggleSection,
    toggleDeploy,
    toggleReleaseFile,
    isSectionExpanded,
    isDeployExpanded,
    isReleaseFileExpanded,
    expandAll,
    collapseAll,
    areAllExpanded,
    totalSections,
    totalEntries,
  };
};
