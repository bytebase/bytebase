import { useEffect, useMemo, useState } from "react";
import type {
  TaskRunLogEntry,
  TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import type { TaskRunLogDetailText } from "./model";
import {
  buildReleaseFileGroups,
  buildSectionsFromEntries,
  formatDuration,
  getEntryTimeRange,
  getUniqueReplicaIds,
  groupEntriesByReleaseFile,
  groupEntriesByReplica,
  hasReleaseFileMarkers,
} from "./model";
import type { ReleaseFileGroup, ReplicaGroup, Section } from "./types";

const addToSet = (set: Set<string>, value: string): Set<string> => {
  if (set.has(value)) return set;
  const next = new Set(set);
  next.add(value);
  return next;
};

const deleteFromSet = (set: Set<string>, value: string): Set<string> => {
  if (!set.has(value)) return set;
  const next = new Set(set);
  next.delete(value);
  return next;
};

const countSections = (sections: Section[]): number => sections.length;
const countEntries = (sections: Section[]): number =>
  sections.reduce((sum, section) => sum + section.entryCount, 0);

export interface UseTaskRunLogSectionsOptions {
  entries: TaskRunLogEntry[];
  sheet?: Sheet;
  sheetsMap?: Map<string, Sheet>;
  getSectionLabel: (type: TaskRunLogEntry_Type) => string;
  detailText?: TaskRunLogDetailText;
  datasetKey?: string;
}

export interface UseTaskRunLogSectionsResult {
  sections: Section[];
  hasMultipleReplicas: boolean;
  hasReleaseFiles: boolean;
  releaseFileGroups: ReleaseFileGroup[];
  replicaGroups: ReplicaGroup[];
  expandedSections: Set<string>;
  expandedReplicas: Set<string>;
  expandedReleaseFiles: Set<string>;
  toggleSection: (sectionId: string) => void;
  toggleReplica: (replicaId: string) => void;
  toggleReleaseFile: (releaseFileId: string) => void;
  isSectionExpanded: (sectionId: string) => boolean;
  isReplicaExpanded: (replicaId: string) => boolean;
  isReleaseFileExpanded: (releaseFileId: string) => boolean;
  expandAll: () => void;
  collapseAll: () => void;
  areAllExpanded: boolean;
  totalSections: number;
  totalEntries: number;
  totalDuration: string;
}

export const useTaskRunLogSections = ({
  entries,
  sheet,
  sheetsMap,
  getSectionLabel,
  detailText,
  datasetKey,
}: UseTaskRunLogSectionsOptions): UseTaskRunLogSectionsResult => {
  const [expandedSections, setExpandedSections] = useState<Set<string>>(
    () => new Set()
  );
  const [userCollapsedSections, setUserCollapsedSections] = useState<
    Set<string>
  >(() => new Set());
  const [expandedReplicas, setExpandedReplicas] = useState<Set<string>>(
    () => new Set()
  );
  const [userCollapsedReplicas, setUserCollapsedReplicas] = useState<
    Set<string>
  >(() => new Set());
  const [expandedReleaseFiles, setExpandedReleaseFiles] = useState<Set<string>>(
    () => new Set()
  );
  const [userCollapsedReleaseFiles, setUserCollapsedReleaseFiles] = useState<
    Set<string>
  >(() => new Set());

  const sections = useMemo(() => {
    return buildSectionsFromEntries(entries, {
      getSectionLabel,
      sheet,
      sheetsMap,
      detailText,
    });
  }, [detailText, entries, getSectionLabel, sheet, sheetsMap]);

  const releaseFileGroups = useMemo(() => {
    if (!hasReleaseFileMarkers(entries)) return [];
    return buildReleaseFileGroups(entries, {
      getSectionLabel,
      sheet,
      sheetsMap,
      detailText,
      includeOrphanGroup: true,
    });
  }, [detailText, entries, getSectionLabel, sheet, sheetsMap]);

  const replicaGroups = useMemo(() => {
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
          sections: buildSectionsFromEntries(replicaEntries, {
            getSectionLabel,
            sheet,
            sheetsMap,
            idPrefix: replicaId,
            forceError,
            detailText,
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
          sheet,
          sheetsMap,
          idPrefix: replicaId,
          forceError,
          detailText,
        }),
        sections:
          orphanGroup && orphanGroup.entries.length > 0
            ? buildSectionsFromEntries(orphanGroup.entries, {
                getSectionLabel,
                sheet,
                sheetsMap,
                idPrefix: `${replicaId}-orphan`,
                forceError,
                detailText,
              })
            : [],
      };
    });
  }, [detailText, entries, getSectionLabel, sheet, sheetsMap]);

  const hasReleaseFiles = useMemo(
    () => hasReleaseFileMarkers(entries),
    [entries]
  );
  const hasMultipleReplicas = replicaGroups.length > 1;

  const allSectionIds = useMemo(() => {
    if (hasMultipleReplicas) {
      return replicaGroups.flatMap((group) => [
        ...group.sections.map((section) => section.id),
        ...group.releaseFileGroups.flatMap((fileGroup) =>
          fileGroup.sections.map((section) => section.id)
        ),
      ]);
    }
    if (hasReleaseFiles) {
      return releaseFileGroups.flatMap((fileGroup) =>
        fileGroup.sections.map((section) => section.id)
      );
    }
    return sections.map((section) => section.id);
  }, [
    hasMultipleReplicas,
    hasReleaseFiles,
    releaseFileGroups,
    replicaGroups,
    sections,
  ]);

  const allReplicaIds = useMemo(() => {
    return replicaGroups.map((group) => group.replicaId);
  }, [replicaGroups]);

  const allReleaseFileIds = useMemo(() => {
    if (hasMultipleReplicas) {
      return replicaGroups.flatMap((group) =>
        group.releaseFileGroups.map((fileGroup) => fileGroup.id)
      );
    }
    return releaseFileGroups.map((fileGroup) => fileGroup.id);
  }, [hasMultipleReplicas, releaseFileGroups, replicaGroups]);

  const resolvedDatasetKey = useMemo(() => {
    if (datasetKey) return datasetKey;
    return entries
      .map((entry) => {
        return [
          entry.type,
          entry.replicaId,
          entry.logTime?.seconds.toString() ?? "",
          entry.logTime?.nanos ?? 0,
          entry.releaseFileExecute?.version ?? "",
          entry.releaseFileExecute?.filePath ?? "",
        ].join(":");
      })
      .join("|");
  }, [datasetKey, entries]);

  useEffect(() => {
    setExpandedSections(new Set());
    setUserCollapsedSections(new Set());
    setExpandedReplicas(new Set());
    setUserCollapsedReplicas(new Set());
    setExpandedReleaseFiles(new Set());
    setUserCollapsedReleaseFiles(new Set());
  }, [resolvedDatasetKey]);

  useEffect(() => {
    setExpandedSections((previous) => {
      let next = previous;
      for (const section of sections) {
        if (
          section.status === "error" &&
          !userCollapsedSections.has(section.id)
        ) {
          next = addToSet(next, section.id);
        }
      }
      return next;
    });
  }, [sections, userCollapsedSections]);

  useEffect(() => {
    setExpandedReplicas((previousReplicas) => {
      let next = previousReplicas;
      for (const group of replicaGroups) {
        if (!userCollapsedReplicas.has(group.replicaId)) {
          next = addToSet(next, group.replicaId);
        }
      }
      return next;
    });

    setExpandedReleaseFiles((previousFiles) => {
      let next = previousFiles;
      for (const group of replicaGroups) {
        group.releaseFileGroups.forEach((fileGroup) => {
          const fileId = fileGroup.id;
          if (!userCollapsedReleaseFiles.has(fileId)) {
            next = addToSet(next, fileId);
          }
        });
      }
      return next;
    });

    setExpandedSections((previousSections) => {
      let next = previousSections;
      for (const group of replicaGroups) {
        for (const section of group.sections) {
          if (
            section.status === "error" &&
            !userCollapsedSections.has(section.id)
          ) {
            next = addToSet(next, section.id);
          }
        }
        for (const fileGroup of group.releaseFileGroups) {
          for (const section of fileGroup.sections) {
            if (
              section.status === "error" &&
              !userCollapsedSections.has(section.id)
            ) {
              next = addToSet(next, section.id);
            }
          }
        }
      }
      return next;
    });
  }, [
    replicaGroups,
    userCollapsedReplicas,
    userCollapsedReleaseFiles,
    userCollapsedSections,
  ]);

  useEffect(() => {
    setExpandedReleaseFiles((previousFiles) => {
      let next = previousFiles;
      releaseFileGroups.forEach((fileGroup) => {
        const fileId = fileGroup.id;
        if (!userCollapsedReleaseFiles.has(fileId)) {
          next = addToSet(next, fileId);
        }
      });
      return next;
    });

    setExpandedSections((previousSections) => {
      let next = previousSections;
      for (const fileGroup of releaseFileGroups) {
        for (const section of fileGroup.sections) {
          if (
            section.status === "error" &&
            !userCollapsedSections.has(section.id)
          ) {
            next = addToSet(next, section.id);
          }
        }
      }
      return next;
    });
  }, [releaseFileGroups, userCollapsedReleaseFiles, userCollapsedSections]);

  const toggleSection = (sectionId: string) => {
    setExpandedSections((previousExpanded) => {
      const isExpanded = previousExpanded.has(sectionId);
      setUserCollapsedSections((previousCollapsed) =>
        isExpanded
          ? addToSet(previousCollapsed, sectionId)
          : deleteFromSet(previousCollapsed, sectionId)
      );
      return isExpanded
        ? deleteFromSet(previousExpanded, sectionId)
        : addToSet(previousExpanded, sectionId);
    });
  };

  const toggleReplica = (replicaId: string) => {
    setExpandedReplicas((previousExpanded) => {
      const isExpanded = previousExpanded.has(replicaId);
      setUserCollapsedReplicas((previousCollapsed) =>
        isExpanded
          ? addToSet(previousCollapsed, replicaId)
          : deleteFromSet(previousCollapsed, replicaId)
      );
      return isExpanded
        ? deleteFromSet(previousExpanded, replicaId)
        : addToSet(previousExpanded, replicaId);
    });
  };

  const toggleReleaseFile = (releaseFileId: string) => {
    setExpandedReleaseFiles((previousExpanded) => {
      const isExpanded = previousExpanded.has(releaseFileId);
      setUserCollapsedReleaseFiles((previousCollapsed) =>
        isExpanded
          ? addToSet(previousCollapsed, releaseFileId)
          : deleteFromSet(previousCollapsed, releaseFileId)
      );
      return isExpanded
        ? deleteFromSet(previousExpanded, releaseFileId)
        : addToSet(previousExpanded, releaseFileId);
    });
  };

  const isSectionExpanded = (sectionId: string): boolean => {
    return expandedSections.has(sectionId);
  };

  const isReplicaExpanded = (replicaId: string): boolean => {
    return expandedReplicas.has(replicaId);
  };

  const isReleaseFileExpanded = (releaseFileId: string): boolean => {
    return expandedReleaseFiles.has(releaseFileId);
  };

  const expandAll = () => {
    setExpandedSections(new Set(allSectionIds));
    setExpandedReplicas(new Set(allReplicaIds));
    setExpandedReleaseFiles(new Set(allReleaseFileIds));
    setUserCollapsedSections(new Set());
    setUserCollapsedReplicas(new Set());
    setUserCollapsedReleaseFiles(new Set());
  };

  const collapseAll = () => {
    setExpandedSections(new Set());
    setExpandedReplicas(new Set());
    setExpandedReleaseFiles(new Set());
    setUserCollapsedSections(new Set(allSectionIds));
    setUserCollapsedReplicas(new Set(allReplicaIds));
    setUserCollapsedReleaseFiles(new Set(allReleaseFileIds));
  };

  const areAllExpanded =
    (allSectionIds.length > 0 ||
      allReleaseFileIds.length > 0 ||
      allReplicaIds.length > 0) &&
    (allSectionIds.length === 0 ||
      allSectionIds.every((id) => expandedSections.has(id))) &&
    (allReleaseFileIds.length === 0 ||
      allReleaseFileIds.every((id) => expandedReleaseFiles.has(id))) &&
    (!hasMultipleReplicas ||
      allReplicaIds.length === 0 ||
      allReplicaIds.every((id) => expandedReplicas.has(id)));

  const totalSections = useMemo(() => {
    if (hasMultipleReplicas) {
      return replicaGroups.reduce(
        (sum, group) =>
          sum +
          countSections(group.sections) +
          group.releaseFileGroups.reduce(
            (fileSum, fileGroup) => fileSum + countSections(fileGroup.sections),
            0
          ),
        0
      );
    }
    if (hasReleaseFiles) {
      return releaseFileGroups.reduce(
        (sum, fileGroup) => sum + countSections(fileGroup.sections),
        0
      );
    }
    return sections.length;
  }, [
    hasMultipleReplicas,
    hasReleaseFiles,
    releaseFileGroups,
    replicaGroups,
    sections.length,
  ]);

  const totalEntries = useMemo(() => {
    if (hasMultipleReplicas) {
      return replicaGroups.reduce(
        (sum, group) =>
          sum +
          countEntries(group.sections) +
          group.releaseFileGroups.reduce(
            (fileSum, fileGroup) => fileSum + countEntries(fileGroup.sections),
            0
          ),
        0
      );
    }
    if (hasReleaseFiles) {
      return releaseFileGroups.reduce(
        (sum, fileGroup) => sum + countEntries(fileGroup.sections),
        0
      );
    }
    return countEntries(sections);
  }, [
    hasMultipleReplicas,
    hasReleaseFiles,
    releaseFileGroups,
    replicaGroups,
    sections,
  ]);

  const totalDuration = useMemo(() => {
    if (entries.length === 0) return "";
    const timeRanges = entries.map(getEntryTimeRange);
    const startTimes = timeRanges
      .map((range) => range.start)
      .filter((time) => time > 0);
    const endTimes = timeRanges
      .map((range) => range.end)
      .filter((time) => time > 0);
    if (startTimes.length === 0 || endTimes.length === 0) return "";

    const startTime = Math.min(...startTimes);
    const endTime = Math.max(...endTimes);
    return formatDuration(endTime - startTime);
  }, [entries]);

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
