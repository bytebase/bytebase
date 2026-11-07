import {
  AlertTriangle,
  ChevronsDownUp,
  ChevronsUpDown,
  FileCode,
  List,
  Server,
} from "lucide-react";
import { memo, type ReactNode, useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { useSeededState } from "@/hooks/useSeededState";
import { Release_Type } from "@/types/proto-es/v1/release_service_pb";
import {
  type Task_ReleaseState,
  TaskRun_Status,
  TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import { LogRows, ROW_DIVIDER } from "./LogRows";
import { countRetryMarkers, type TaskRunLogDetailText } from "./model";
import {
  type OpenReleaseFile,
  ReleaseSpineView,
  RevisionsLink,
  type RevisionsLinkTarget,
} from "./ReleaseSpineView";
import { SectionContent } from "./SectionContent";
import { SectionStatusIcon } from "./SectionHeader";
import type { LogRow } from "./types";
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
  // The run's error (TaskRun.detail): a versioned release's spine places it on
  // the section it belongs to.
  runError?: string;
  // The task's stored release state. The plan page passes it; the revision
  // page renders the run's log alone.
  releaseState?: Task_ReleaseState;
  onOpenFile?: OpenReleaseFile;
  // The run's error alert. The spine places the error on its section, so the
  // alert shows only when the spine does not render.
  errorAlert?: ReactNode;
  // The database's revisions, shown once the run applied a file. Without the
  // spine there are no per-file states, so only a done run shows it.
  revisionsLink?: RevisionsLinkTarget;
}

// memo: callers pass scalars and identity-stable objects (the stored state,
// a memoized error alert and file opener), and data flows in via zustand
// subscriptions, so re-renders of the surrounding card don't cascade into the
// log tree unless the run itself changed.
export const TaskRunLogViewer = memo(function TaskRunLogViewer({
  taskRunName,
  taskRunStatus,
  active = true,
  runError,
  releaseState,
  onOpenFile,
  errorAlert,
  revisionsLink,
}: TaskRunLogViewerProps) {
  const { t } = useTranslation();
  const {
    entries,
    logFetch,
    sheet,
    sheetsMap,
    release,
    taskRun: fetchedRun,
  } = useTaskRunLogData(taskRunName, taskRunStatus, active);

  // The reader's explicit row folds. They live here, above every conditional
  // SectionContent mount, so collapsing a section or the sole-section layout
  // giving way to the section tree does not discard them. A new run starts
  // clean, as does a new phase: both deploy surfaces key this viewer on status.
  const [foldOverrides, setFoldOverrides] = useSeededState<
    ReadonlyMap<string, boolean>
  >(taskRunName, () => new Map());
  const handleFoldChange = useCallback(
    (key: string, open: boolean) =>
      setFoldOverrides((previous) => new Map(previous).set(key, open)),
    [setFoldOverrides]
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
    }),
    [t]
  );

  // A retry marker regroups the positional rows, so it resets expansion the
  // way switching runs does; an ordinary append leaves the key alone.
  const datasetKey = `${taskRunName}:${countRetryMarkers(entries)}`;
  const {
    rows,
    hasMultipleReplicas,
    hasReleaseFiles,
    releaseFileGroups,
    replicaGroups,
    toggleRow,
    toggleReplica,
    toggleReleaseFile,
    isRowExpanded,
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
    datasetKey,
    taskRunStatus,
  });

  const spineFiles = useMemo(
    () =>
      release?.type === Release_Type.VERSIONED
        ? release.files.map((file) => ({
            version: file.version,
            path: file.path,
          }))
        : [],
    [release]
  );
  const spineRun = useMemo(() => {
    const status = taskRunStatus ?? fetchedRun?.status;
    return status === undefined
      ? undefined
      : { status, error: runError ?? fetchedRun?.detail ?? "" };
  }, [taskRunStatus, runError, fetchedRun]);

  const hasRenderableReleaseFiles =
    hasReleaseFiles && releaseFileGroups.length > 0;
  const hasContent =
    rows.length > 0 || hasMultipleReplicas || hasRenderableReleaseFiles;

  if (!taskRunName) {
    return null;
  }
  const showRevisions =
    revisionsLink !== undefined && taskRunStatus === TaskRun_Status.DONE;
  const withAlert = (node: ReactNode) =>
    errorAlert || showRevisions ? (
      <div className="flex flex-col gap-2">
        {errorAlert}
        {node}
        {showRevisions && revisionsLink ? (
          <RevisionsLink target={revisionsLink} />
        ) : null}
      </div>
    ) : (
      node
    );

  // A versioned release's run reads as one spine of file sections. Several
  // replicas, or a run whose status is unknown, keep the plain log. The log
  // decides the spine, so it waits for a first load (an empty log reads as
  // every file applied); a failed refresh keeps its entries.
  if (
    spineRun &&
    spineFiles.length > 0 &&
    !hasMultipleReplicas &&
    (logFetch.status === "success" || entries.length > 0)
  ) {
    return (
      <ReleaseSpineView
        key={datasetKey}
        files={spineFiles}
        entries={entries}
        run={spineRun}
        state={releaseState}
        sheetsMap={sheetsMap}
        getSectionLabel={getSectionLabel}
        detailText={detailText}
        onOpenFile={onOpenFile}
        revisionsLink={revisionsLink}
      />
    );
  }

  if (!hasContent) {
    // Reserve the approximate height of a small loaded log (label row plus a
    // couple of entries) while the first fetch is in flight, so entries fill
    // the box in place instead of popping the card taller.
    if (logFetch.status === "loading") {
      return withAlert(
        <div className="flex min-h-20 items-center justify-center rounded-sm border border-block-border bg-control-bg/50 px-3 py-2 text-sm text-control-light">
          {t("common.loading")}
        </div>
      );
    }
    return errorAlert ?? null;
  }

  // A single section with no replicas or release files carries no structure
  // worth disclosing — skip the summary bar and the collapsible section header
  // and show the entries directly under a lightweight, non-collapsible label.
  const firstRow = rows[0];
  const soleSection =
    !hasMultipleReplicas &&
    !hasRenderableReleaseFiles &&
    rows.length === 1 &&
    firstRow?.kind === "section"
      ? firstRow
      : undefined;
  if (soleSection) {
    return withAlert(
      <div
        className="w-full font-mono text-xs"
        data-testid="task-run-log-viewer"
      >
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
          <SectionContent
            key={datasetKey}
            section={soleSection}
            foldOverrides={foldOverrides}
            onFoldChange={handleFoldChange}
          />
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

  // Keyed so every row's local state (load more) resets with the regrouping.
  const logRows = (groupRows: LogRow[], indent = false) => (
    <LogRows
      key={datasetKey}
      rows={groupRows}
      indent={indent}
      running={taskRunStatus === TaskRun_Status.RUNNING}
      isRowExpanded={isRowExpanded}
      onToggleRow={toggleRow}
      foldOverrides={foldOverrides}
      onFoldChange={handleFoldChange}
    />
  );

  const renderReleaseFileGroup = (
    fileGroup: (typeof releaseFileGroups)[number],
    indent = false
  ) => {
    if (fileGroup.isOrphan) {
      return <div key={fileGroup.id}>{logRows(fileGroup.rows, indent)}</div>;
    }

    return (
      <div key={fileGroup.id} className={ROW_DIVIDER}>
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
              {logRows(fileGroup.rows, true)}
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
              {logRows(replicaGroup.rows, true)}
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
    logRows(rows)
  );

  return withAlert(
    <div className="w-full font-mono text-xs" data-testid="task-run-log-viewer">
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
