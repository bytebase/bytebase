import { useMemo, useState } from "react";
import { useOnKeyChange } from "@/hooks/useOnKeyChange";
import type {
  TaskRun_Status,
  TaskRunLogEntry,
  TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import type { TaskRunLogDetailText } from "./model";
import {
  assignEntryKeys,
  buildReleaseFileGroups,
  buildRowsFromEntries,
  collectAttemptRowIds,
  collectLeafSections,
  getUniqueReplicaIds,
  groupEntriesByReleaseFile,
  groupEntriesByReplica,
  hasReleaseFileMarkers,
} from "./model";
import type { LogRow, ReleaseFileGroup, ReplicaGroup } from "./types";

// One expansion lane: every id is open by its default until the reader, or
// Expand all / Collapse all, overrides it. `ids` are exactly the rows a reader
// can toggle, so the toolbar's all-expanded state matches the screen.
const useExpansionLane = (
  ids: string[],
  defaultOpen: (id: string) => boolean
) => {
  const [overrides, setOverrides] = useState<ReadonlyMap<string, boolean>>(
    () => new Map()
  );
  const isExpanded = (id: string): boolean =>
    overrides.get(id) ?? defaultOpen(id);
  return {
    isExpanded,
    toggle: (id: string) =>
      setOverrides((previous) =>
        new Map(previous).set(id, !(previous.get(id) ?? defaultOpen(id)))
      ),
    setAll: (open: boolean) =>
      setOverrides(new Map(ids.map((id) => [id, open]))),
    reset: () => setOverrides(new Map()),
    allExpanded: ids.every(isExpanded),
  };
};

export interface UseTaskRunLogSectionsOptions {
  entries: TaskRunLogEntry[];
  sheet?: Sheet;
  sheetsMap?: Map<string, Sheet>;
  getSectionLabel: (type: TaskRunLogEntry_Type) => string;
  detailText?: TaskRunLogDetailText;
  // Changing it clears every override: row ids are positional, so they only
  // mean something within one grouping of one run.
  datasetKey: string;
  // Forwarded into every builder call below; see pickMarkedEntry.
  taskRunStatus?: TaskRun_Status;
}

export interface UseTaskRunLogSectionsResult {
  rows: LogRow[];
  hasMultipleReplicas: boolean;
  hasReleaseFiles: boolean;
  releaseFileGroups: ReleaseFileGroup[];
  replicaGroups: ReplicaGroup[];
  // Rows are sections, umbrellas and attempt rows, by id.
  toggleRow: (rowId: string) => void;
  toggleReplica: (replicaId: string) => void;
  toggleReleaseFile: (releaseFileId: string) => void;
  isRowExpanded: (rowId: string) => boolean;
  isReplicaExpanded: (replicaId: string) => boolean;
  isReleaseFileExpanded: (releaseFileId: string) => boolean;
  expandAll: () => void;
  collapseAll: () => void;
  areAllExpanded: boolean;
  totalSections: number;
  totalEntries: number;
}

export const useTaskRunLogSections = ({
  entries,
  sheet,
  sheetsMap,
  getSectionLabel,
  detailText,
  datasetKey,
  taskRunStatus,
}: UseTaskRunLogSectionsOptions): UseTaskRunLogSectionsResult => {
  // Every builder below sees a filtered slice of the run, so the keys are
  // assigned here, over all of it.
  const entryKeys = useMemo(() => assignEntryKeys(entries), [entries]);

  const rows = useMemo(() => {
    return buildRowsFromEntries(entries, {
      getSectionLabel,
      entryKeys,
      sheet,
      sheetsMap,
      detailText,
      taskRunStatus,
    });
  }, [
    detailText,
    entries,
    entryKeys,
    getSectionLabel,
    sheet,
    sheetsMap,
    taskRunStatus,
  ]);

  const releaseFileGroups = useMemo(() => {
    if (!hasReleaseFileMarkers(entries)) return [];
    return buildReleaseFileGroups(entries, {
      getSectionLabel,
      entryKeys,
      sheet,
      sheetsMap,
      detailText,
      includeOrphanGroup: true,
      taskRunStatus,
    });
  }, [
    detailText,
    entries,
    entryKeys,
    getSectionLabel,
    sheet,
    sheetsMap,
    taskRunStatus,
  ]);

  const replicaGroups = useMemo<ReplicaGroup[]>(() => {
    const replicaIds = getUniqueReplicaIds(entries);
    if (replicaIds.length <= 1) return [];

    const entriesByReplica = groupEntriesByReplica(entries);
    return replicaIds.map((replicaId, index) => {
      const replicaEntries = entriesByReplica.get(replicaId) ?? [];
      const forceError = index < replicaIds.length - 1;

      if (!hasReleaseFileMarkers(replicaEntries)) {
        return {
          replicaId,
          releaseFileGroups: [],
          rows: buildRowsFromEntries(replicaEntries, {
            getSectionLabel,
            entryKeys,
            sheet,
            sheetsMap,
            idPrefix: replicaId,
            forceError,
            detailText,
            taskRunStatus,
          }),
        };
      }

      const releaseEntryGroups = groupEntriesByReleaseFile(replicaEntries);
      const orphanGroup = releaseEntryGroups.find(
        (group) => group.file === null
      );
      return {
        replicaId,
        releaseFileGroups: buildReleaseFileGroups(replicaEntries, {
          getSectionLabel,
          entryKeys,
          sheet,
          sheetsMap,
          idPrefix: replicaId,
          forceError,
          detailText,
          taskRunStatus,
        }),
        rows:
          orphanGroup && orphanGroup.entries.length > 0
            ? buildRowsFromEntries(orphanGroup.entries, {
                getSectionLabel,
                entryKeys,
                sheet,
                sheetsMap,
                idPrefix: `${replicaId}-orphan`,
                forceError,
                detailText,
                taskRunStatus,
              })
            : [],
      };
    });
  }, [
    detailText,
    entries,
    entryKeys,
    getSectionLabel,
    sheet,
    sheetsMap,
    taskRunStatus,
  ]);

  const hasReleaseFiles = useMemo(
    () => hasReleaseFileMarkers(entries),
    [entries]
  );
  const hasMultipleReplicas = replicaGroups.length > 1;

  // Every row the viewer will render, in order, whichever grouping applies.
  const allRows = useMemo<LogRow[]>(() => {
    if (hasMultipleReplicas) {
      return replicaGroups.flatMap((group) => [
        ...group.rows,
        ...group.releaseFileGroups.flatMap((fileGroup) => fileGroup.rows),
      ]);
    }
    if (hasReleaseFiles) {
      return releaseFileGroups.flatMap((fileGroup) => fileGroup.rows);
    }
    return rows;
  }, [
    hasMultipleReplicas,
    hasReleaseFiles,
    releaseFileGroups,
    replicaGroups,
    rows,
  ]);

  const allLeafSections = useMemo(
    () => collectLeafSections(allRows),
    [allRows]
  );
  const allRowIds = [
    ...allLeafSections.map((section) => section.id),
    ...collectAttemptRowIds(allRows),
  ];
  const allReplicaIds = useMemo(
    () => replicaGroups.map((group) => group.replicaId),
    [replicaGroups]
  );
  const allReleaseFileIds = useMemo(() => {
    if (hasMultipleReplicas) {
      return replicaGroups.flatMap((group) =>
        group.releaseFileGroups.map((fileGroup) => fileGroup.id)
      );
    }
    // The orphan group renders without a header of its own.
    return releaseFileGroups
      .filter((fileGroup) => !fileGroup.isOrphan)
      .map((fileGroup) => fileGroup.id);
  }, [hasMultipleReplicas, releaseFileGroups, replicaGroups]);

  const errorSectionIds = useMemo(
    () =>
      new Set(
        allLeafSections
          .filter((section) => section.status === "error")
          .map((section) => section.id)
      ),
    [allLeafSections]
  );

  // Failed sections are open by default wherever they sit; umbrella and
  // attempt rows are not.
  const rowLane = useExpansionLane(allRowIds, (id) => errorSectionIds.has(id));
  const replicaLane = useExpansionLane(allReplicaIds, () => true);
  const fileLane = useExpansionLane(allReleaseFileIds, () => true);

  useOnKeyChange(datasetKey, () => {
    rowLane.reset();
    replicaLane.reset();
    fileLane.reset();
  });

  const setAll = (open: boolean) => {
    rowLane.setAll(open);
    replicaLane.setAll(open);
    fileLane.setAll(open);
  };

  return {
    rows,
    hasMultipleReplicas,
    hasReleaseFiles,
    releaseFileGroups,
    replicaGroups,
    toggleRow: rowLane.toggle,
    toggleReplica: replicaLane.toggle,
    toggleReleaseFile: fileLane.toggle,
    isRowExpanded: rowLane.isExpanded,
    isReplicaExpanded: replicaLane.isExpanded,
    isReleaseFileExpanded: fileLane.isExpanded,
    expandAll: () => setAll(true),
    collapseAll: () => setAll(false),
    areAllExpanded:
      allRowIds.length + allReleaseFileIds.length + allReplicaIds.length > 0 &&
      rowLane.allExpanded &&
      fileLane.allExpanded &&
      replicaLane.allExpanded,
    totalSections: allLeafSections.length,
    totalEntries: allLeafSections.reduce(
      (sum, section) => sum + section.entryCount,
      0
    ),
  };
};
