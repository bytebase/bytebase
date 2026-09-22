import {
  AlertTriangle,
  ChevronsDownUp,
  ChevronsUpDown,
  FileCode,
  List,
  Server,
} from "lucide-react";
import { memo, useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  type TaskRun_Status,
  TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { TaskRunLogDetailText } from "./model";
import { SectionContent } from "./SectionContent";
import { SectionHeader, SectionStatusIcon } from "./SectionHeader";
import { useTaskRunLogData } from "./useTaskRunLogData";
import { useTaskRunLogSections } from "./useTaskRunLogSections";

export interface TaskRunLogViewerProps {
  taskRunName: string;
  // Pass when known: a terminal status lets a cached log render without a
  // refetch, since a finished run's log is immutable. Callers without a
  // TaskRun in hand (changelog/revision pages) omit it and always revalidate,
  // even though those runs are terminal by construction.
  taskRunStatus?: TaskRun_Status;
  // When false, pause the live log poll — the card is mounted but its stage is
  // hidden. Defaults to live.
  active?: boolean;
}

// memo: both props are scalars and data flows in via zustand subscriptions
// (not React context), so re-renders of the surrounding card don't cascade
// into the log tree unless the run itself changed.
export const TaskRunLogViewer = memo(function TaskRunLogViewer({
  taskRunName,
  taskRunStatus,
  active = true,
}: TaskRunLogViewerProps) {
  const { t } = useTranslation();
  const { entries, logFetch, sheet, sheetsMap } = useTaskRunLogData(
    taskRunName,
    taskRunStatus,
    active
  );

  const getSectionLabel = useCallback(
    (type: TaskRunLogEntry_Type) => {
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
        [TaskRunLogEntry_Type.PRIOR_BACKUP]: t(
          "task-run.log-type.prior-backup"
        ),
        [TaskRunLogEntry_Type.RETRY_INFO]: t("task-run.log-type.retry"),
        [TaskRunLogEntry_Type.COMPUTE_DIFF]: t(
          "task-run.log-type.compute-diff"
        ),
        [TaskRunLogEntry_Type.GHOST_MIGRATION]: t(
          "task-run.log-type.ghost-migration"
        ),
        [TaskRunLogEntry_Type.RELEASE_FILE_EXECUTE]: t(
          "task-run.log-type.release-file-execute"
        ),
      };
      return labelMap[type] ?? t("common.unknown");
    },
    [t]
  );

  const detailText = useMemo<TaskRunLogDetailText>(
    () => ({
      completed: t("task-run.log-detail.completed"),
      backingUp: t("task-run.log-detail.backing-up"),
      runningByType: {
        [TaskRunLogEntry_Type.SCHEMA_DUMP]: t("task-run.log-detail.dumping"),
        [TaskRunLogEntry_Type.DATABASE_SYNC]: t("task-run.log-detail.syncing"),
        [TaskRunLogEntry_Type.COMPUTE_DIFF]: t("task-run.log-detail.computing"),
      },
      backupCompleted: (count: number) =>
        t("task-run.log-detail.backup-completed", { count }),
      retryAttempt: (current: number, max: number) =>
        t("task-run.log-detail.retry-attempt", { current, max }),
    }),
    [t]
  );

  const {
    sections,
    hasMultipleReplicas,
    hasReleaseFiles,
    releaseFileGroups,
    replicaGroups,
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
  } = useTaskRunLogSections({
    entries,
    sheet,
    sheetsMap,
    getSectionLabel,
    detailText,
    datasetKey: taskRunName,
  });

  const hasRenderableReleaseFiles =
    hasReleaseFiles && releaseFileGroups.length > 0;
  const hasContent =
    sections.length > 0 || hasMultipleReplicas || hasRenderableReleaseFiles;

  if (!taskRunName) {
    return null;
  }

  if (!hasContent) {
    // Reserve the approximate height of a small loaded log (label row plus a
    // couple of entries) while the first fetch is in flight, so entries fill
    // the box in place instead of popping the card taller.
    if (logFetch.status === "loading") {
      return (
        <div className="flex min-h-20 items-center justify-center rounded-sm border border-block-border bg-control-bg/50 px-3 py-2 text-sm text-control-light">
          {t("common.loading")}
        </div>
      );
    }
    return null;
  }

  // A single section with no replicas or release files carries no structure
  // worth disclosing — skip the summary bar and the collapsible section header
  // and show the entries directly under a lightweight, non-collapsible label.
  const soleSection =
    !hasMultipleReplicas && !hasRenderableReleaseFiles && sections.length === 1
      ? sections[0]
      : undefined;
  if (soleSection) {
    return (
      <div className="w-full font-mono text-xs">
        <div className="w-full overflow-hidden rounded-sm border border-block-border bg-control-bg/50">
          <div className="flex items-center gap-x-2 px-3 py-1.5 text-control">
            <SectionStatusIcon section={soleSection} />
            <span>{soleSection.label}</span>
            {soleSection.duration ? (
              <span className="ml-auto tabular-nums text-control-light">
                {soleSection.duration}
              </span>
            ) : null}
          </div>
          <SectionContent section={soleSection} datasetKey={taskRunName} />
        </div>
      </div>
    );
  }

  const toggleExpandAll = () => {
    if (areAllExpanded) {
      collapseAll();
      return;
    }
    expandAll();
  };

  const renderSection = (
    section: (typeof sections)[number],
    indent = false
  ) => (
    <div
      key={section.id}
      className="border-block-border border-b last:border-b-0"
    >
      <SectionHeader
        section={section}
        indent={indent}
        isExpanded={isSectionExpanded(section.id)}
        onToggle={() => toggleSection(section.id)}
      />
      {isSectionExpanded(section.id) ? (
        <SectionContent
          section={section}
          indent={indent}
          datasetKey={taskRunName}
        />
      ) : null}
    </div>
  );

  const renderReleaseFileGroup = (
    fileGroup: (typeof releaseFileGroups)[number],
    indent = false
  ) => {
    if (fileGroup.isOrphan) {
      return (
        <div key={fileGroup.id}>
          {fileGroup.sections.map((section) => renderSection(section, indent))}
        </div>
      );
    }

    return (
      <div
        key={fileGroup.id}
        className="border-block-border border-b last:border-b-0"
      >
        <div className={indent ? "pl-4" : ""}>
          <Button
            type="button"
            appearance="secondary"
            size="sm"
            className="w-full justify-start rounded-none bg-info/10 text-info hover:bg-info/20"
            onClick={() => toggleReleaseFile(fileGroup.id)}
          >
            {isReleaseFileExpanded(fileGroup.id) ? (
              <ChevronsDownUp className="size-4 shrink-0" />
            ) : (
              <ChevronsUpDown className="size-4 shrink-0" />
            )}
            <FileCode className="size-4 shrink-0" />
            <span className="font-medium">
              {fileGroup.filePath
                ? `${fileGroup.version}: ${fileGroup.filePath}`
                : fileGroup.version}
            </span>
          </Button>
          {isReleaseFileExpanded(fileGroup.id) ? (
            <div className={indent ? "pl-4" : ""}>
              {fileGroup.sections.map((section) =>
                renderSection(section, true)
              )}
            </div>
          ) : null}
        </div>
      </div>
    );
  };

  const content = hasMultipleReplicas ? (
    <>
      <div className="flex items-center gap-x-2 border-warning/40 border-b bg-warning/10 px-3 py-2 text-warning">
        <AlertTriangle className="size-4 shrink-0" />
        <span>{t("task-run.log-viewer.multiple-replicas-notice")}</span>
      </div>
      {replicaGroups.map((replicaGroup, replicaIdx) => (
        <div
          key={replicaGroup.replicaId}
          className="border-control-border border-b last:border-b-0"
        >
          <Button
            type="button"
            appearance="secondary"
            size="sm"
            className="w-full justify-start rounded-none bg-control-bg text-control hover:bg-control-bg-hover"
            onClick={() => toggleReplica(replicaGroup.replicaId)}
          >
            {isReplicaExpanded(replicaGroup.replicaId) ? (
              <ChevronsDownUp className="size-4 shrink-0 text-control-light" />
            ) : (
              <ChevronsUpDown className="size-4 shrink-0 text-control-light" />
            )}
            <Server className="size-4 shrink-0 text-control-light" />
            <span className="font-medium text-control">
              {t("task-run.log-viewer.replica-n", { n: replicaIdx + 1 })}
            </span>
            <span className="font-normal text-control-placeholder text-xs">
              {replicaGroup.replicaId.substring(0, 8)}
            </span>
          </Button>

          {isReplicaExpanded(replicaGroup.replicaId) ? (
            <div>
              {replicaGroup.sections.map((section) =>
                renderSection(section, true)
              )}
              {replicaGroup.releaseFileGroups.map((fileGroup) =>
                renderReleaseFileGroup(fileGroup, true)
              )}
            </div>
          ) : null}
        </div>
      ))}
    </>
  ) : hasRenderableReleaseFiles ? (
    <>
      {releaseFileGroups.map((fileGroup) => renderReleaseFileGroup(fileGroup))}
    </>
  ) : (
    <>{sections.map((section) => renderSection(section))}</>
  );

  return (
    <div className="w-full font-mono text-xs">
      <div className="w-full overflow-hidden rounded-sm border border-block-border bg-control-bg/50">
        <div className="flex items-center justify-between border-block-border border-b bg-control-bg px-2 py-1">
          <div className="flex items-center gap-x-2 text-control-light">
            <List className="size-3.5" />
            <span>
              {t("task-run.log-viewer.summary", {
                sections: totalSections,
                entries: totalEntries,
              })}
            </span>
          </div>
          <Button
            type="button"
            appearance="secondary"
            size="sm"
            className="gap-x-1 text-control-light hover:text-control"
            onClick={toggleExpandAll}
          >
            {areAllExpanded ? (
              <ChevronsDownUp className="size-3.5" />
            ) : (
              <ChevronsUpDown className="size-3.5" />
            )}
            <span>
              {areAllExpanded
                ? t("task-run.log-viewer.collapse-all")
                : t("task-run.log-viewer.expand-all")}
            </span>
          </Button>
        </div>
        {content}
      </div>
    </div>
  );
});

export default TaskRunLogViewer;
