import { ChevronDown, ChevronRight } from "lucide-react";
import { type ReactNode, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { CopyButton } from "@/components/ui/copy-button";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import {
  type Task_ReleaseState,
  TaskRun_Status,
  type TaskRunLogEntry,
  type TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import { formatAbsoluteDateTime } from "@/utils";
import { LogRows, ROW_DIVIDER } from "./LogRows";
import {
  APPLIED_FOLD_PAGE_SIZE,
  assignEntryKeys,
  buildReleaseSpine,
  buildRowsFromEntries,
  collectLeafSections,
  findFailedCommand,
  firstLineOf,
  formatEntriesDuration,
  isSpineItemOpenByDefault,
  limitSpineSections,
  type TaskRunLogDetailText,
} from "./model";
import type {
  LogRow,
  ReleaseSpineFile,
  SpineAppliedFold,
  SpineFileSection,
  SpineFileStatus,
  SpineItem,
} from "./types";

// What a file path knows about the file when it opens the viewer.
export type ReleaseFileOpenContext =
  | {
      kind: "failed";
      error: string;
      // Byte offsets of the failed command in the file's sheet.
      range?: { start: number; end: number };
    }
  | { kind: "applied" }
  // `run` is false before the first run: no run passed over it yet.
  | { kind: "already-applied"; run: boolean }
  | { kind: "pending" }
  // Running or canceled: no settled outcome on this database.
  | { kind: "unsettled" };

export type OpenReleaseFile = (
  file: ReleaseSpineFile,
  context: ReleaseFileOpenContext
) => void;

export interface ReleaseSpineViewProps {
  files: ReleaseSpineFile[];
  entries: TaskRunLogEntry[];
  // The latest run; absent before the first run.
  run?: { status: TaskRun_Status; error: string };
  state?: Task_ReleaseState;
  sheetsMap?: Map<string, Sheet>;
  getSectionLabel: (type: TaskRunLogEntry_Type) => string;
  detailText?: TaskRunLogDetailText;
  onOpenFile?: OpenReleaseFile;
}

const STATUS_VARIANT: Record<
  Exclude<SpineFileStatus, "pending">,
  "success" | "secondary" | "destructive" | "default"
> = {
  applied: "success",
  running: "secondary",
  failed: "destructive",
  canceled: "default",
};

// One task's per-database file state: the already-applied fold, a section per
// file the run reached, task-level steps as siblings, the files a running or
// pending task will apply, and a Run failed section for an error no section
// owns.
export function ReleaseSpineView({
  files,
  entries,
  run,
  state,
  sheetsMap,
  getSectionLabel,
  detailText,
  onOpenFile,
}: ReleaseSpineViewProps) {
  const { t } = useTranslation();
  const spine = useMemo(
    () => buildReleaseSpine({ files, entries, run, state }),
    [files, entries, run, state]
  );
  const items = useMemo(() => limitSpineSections(spine.items), [spine]);
  const entryKeys = useMemo(() => assignEntryKeys(entries), [entries]);
  // Below 640px the failed row opens with its statement block folded.
  const narrow = useMediaQuery(NARROW_MEDIA_QUERY);
  const running = run?.status === TaskRun_Status.RUNNING;

  // Each section's rows, built once per log change rather than per render.
  const rowsById = useMemo(() => {
    const byId = new Map<string, LogRow[]>();
    const sections = items.flatMap((item) =>
      item.kind === "applied-sections" || item.kind === "pending-sections"
        ? item.sections
        : [item]
    );
    for (const item of sections) {
      if (item.kind !== "file" && item.kind !== "step") continue;
      if (item.entries.length === 0) continue;
      byId.set(
        item.id,
        buildRowsFromEntries(item.entries, {
          getSectionLabel,
          entryKeys,
          sheetsMap,
          detailText,
          fileVersion: item.kind === "file" ? item.version : undefined,
          idPrefix: item.id,
          taskRunStatus: run?.status,
        })
      );
    }
    return byId;
  }, [items, getSectionLabel, entryKeys, sheetsMap, detailText, run?.status]);
  // The failed row: the one the log marks as explaining the outcome.
  const markedKeys = useMemo(
    () =>
      [...rowsById.values()]
        .flatMap((rows) => collectLeafSections(rows))
        .flatMap((section) => section.items)
        .filter((item) => item.marked)
        .map((item) => item.key),
    [rowsById]
  );

  const [overrides, setOverrides] = useState<ReadonlyMap<string, boolean>>(
    () => new Map()
  );
  const [foldOverrides, setFoldOverrides] = useState<
    ReadonlyMap<string, boolean>
  >(() => new Map());
  const effectiveFolds = useMemo(() => {
    const folds = new Map<string, boolean>(
      markedKeys.map((key) => [key, !narrow])
    );
    for (const [key, open] of foldOverrides) folds.set(key, open);
    return folds;
  }, [markedKeys, narrow, foldOverrides]);
  const isOpen = (id: string, item: SpineItem) =>
    overrides.get(id) ?? isSpineItemOpenByDefault(item);
  const toggle = (id: string, item: SpineItem) =>
    setOverrides((prev) => new Map(prev).set(id, !isOpen(id, item)));
  // As in the UIOps log: only the rows that hold an error start open.
  const errorRowIds = useMemo(
    () =>
      new Set(
        [...rowsById.values()].flatMap((rows) =>
          rows
            .filter((row) => row.kind === "section" && row.status === "error")
            .map((row) => row.id)
        )
      ),
    [rowsById]
  );
  // Literal keys, so the locale scanner sees each one.
  const statusLabel = (status: Exclude<SpineFileStatus, "pending">) => {
    switch (status) {
      case "applied":
        return t("task-run.spine.status.applied");
      case "running":
        return t("task-run.spine.status.running");
      case "failed":
        return t("task-run.spine.status.failed");
      case "canceled":
        return t("task-run.spine.status.canceled");
    }
  };
  const deletedNote = (section: SpineFileSection) =>
    section.revisionDeleted
      ? {
          date: section.revisionDeleted.time
            ? formatAbsoluteDateTime(
                Number(section.revisionDeleted.time.seconds) * 1000
              )
            : "",
          user: section.revisionDeleted.deleter.replace(/^users\//, ""),
        }
      : undefined;

  const renderRows = (id: string) => {
    const rows = rowsById.get(id);
    if (!rows) return null;
    return (
      <LogRows
        rows={rows}
        indent
        running={running}
        isRowExpanded={(rowId) =>
          overrides.get(rowId) ?? errorRowIds.has(rowId)
        }
        onToggleRow={(rowId) =>
          setOverrides((prev) =>
            new Map(prev).set(
              rowId,
              !(prev.get(rowId) ?? errorRowIds.has(rowId))
            )
          )
        }
        foldOverrides={effectiveFolds}
        onFoldChange={(key, open) =>
          setFoldOverrides((prev) => new Map(prev).set(key, open))
        }
      />
    );
  };

  const openContextOf = (section: SpineFileSection): ReleaseFileOpenContext => {
    switch (section.status) {
      case "failed": {
        const failed = findFailedCommand(section.entries);
        return {
          kind: "failed",
          error: failed?.error || section.error || "",
          range: failed?.range,
        };
      }
      case "applied":
        return { kind: "applied" };
      case "pending":
        return { kind: "pending" };
      default:
        // A running or canceled file has no settled outcome to state.
        return { kind: "unsettled" };
    }
  };

  const renderFile = (section: SpineFileSection) => {
    const file = { version: section.version, path: section.path };
    const note = deletedNote(section);
    const pathLink = (
      <span className="flex min-w-0 items-center gap-x-1">
        <FilePath
          label={section.path || section.version}
          onOpen={
            onOpenFile
              ? () => onOpenFile(file, openContextOf(section))
              : undefined
          }
        />
        {section.path ? <CopyButton content={section.path} /> : null}
      </span>
    );
    if (section.status === "pending") {
      return (
        <div
          key={section.id}
          className={cn(ROW_DIVIDER, "flex items-center gap-x-2 px-3 py-1.5")}
          data-testid={`spine-file-${section.version}`}
        >
          <span className="w-3.5 shrink-0" />
          <span className="shrink-0 tabular-nums text-control-light">
            {section.version}
          </span>
          {pathLink}
          <span className="flex-1" />
          <Badge>
            {note
              ? t("task-run.spine.will-apply-again", note)
              : t("task-run.spine.will-apply")}
          </Badge>
        </div>
      );
    }
    const open = isOpen(section.id, section);
    const duration = formatEntriesDuration(section.entries);
    return (
      <Collapsible
        key={section.id}
        open={open}
        onOpenChange={() => toggle(section.id, section)}
        className={ROW_DIVIDER}
      >
        <div className="flex items-center gap-x-2 px-3 py-1.5">
          <CollapsibleTrigger
            className="w-auto shrink-0"
            data-testid={`spine-file-${section.version}`}
          >
            <Chevron open={open} />
            <span className="tabular-nums text-control">{section.version}</span>
            <Badge variant={STATUS_VARIANT[section.status]}>
              {statusLabel(section.status)}
            </Badge>
          </CollapsibleTrigger>
          {pathLink}
          <span className="flex-1" />
          {duration ? (
            <span className="shrink-0 tabular-nums text-control-light">
              {duration}
            </span>
          ) : null}
        </div>
        {note ? (
          <div
            className="px-9 pb-1 text-control-light"
            data-testid={`spine-file-${section.version}-note`}
          >
            {t("task-run.spine.revision-deleted", note)}
          </div>
        ) : null}
        {!open && section.status === "failed" && section.error ? (
          <div
            className="truncate px-9 pb-1.5 text-error"
            data-testid={`spine-file-${section.version}-error`}
          >
            {firstLineOf(section.error)}
          </div>
        ) : null}
        <CollapsiblePanel data-testid={`spine-file-${section.version}-content`}>
          {section.noStatement ? (
            <div className="px-9 py-1.5">
              <div className="text-control-light">
                {t("task-run.spine.no-statement")}
              </div>
              <div className="whitespace-pre-wrap text-error">
                {section.error}
              </div>
            </div>
          ) : null}
          {renderRows(section.id)}
        </CollapsiblePanel>
      </Collapsible>
    );
  };

  const fold = (
    id: string,
    label: ReactNode,
    children: ReactNode,
    item: SpineItem,
    testId?: string
  ) => {
    const open = isOpen(id, item);
    return (
      <Collapsible
        key={id}
        open={open}
        onOpenChange={() => toggle(id, item)}
        className={ROW_DIVIDER}
      >
        <CollapsibleTrigger className={TRIGGER_ROW} data-testid={testId}>
          <Chevron open={open} />
          {label}
        </CollapsibleTrigger>
        <CollapsiblePanel>{children}</CollapsiblePanel>
      </Collapsible>
    );
  };

  const renderItem = (item: SpineItem): ReactNode => {
    switch (item.kind) {
      case "applied-fold":
        return (
          <AppliedFold
            key="applied-fold"
            fold={item}
            open={isOpen("applied-fold", item)}
            onToggle={() => toggle("applied-fold", item)}
            onOpenFile={onOpenFile}
            hasRun={run !== undefined}
          />
        );
      case "file":
        return renderFile(item);
      case "step": {
        const duration = formatEntriesDuration(item.entries);
        return fold(
          item.id,
          <>
            <span className="text-control">{getSectionLabel(item.type)}</span>
            <span className="flex-1" />
            {duration ? <span className="tabular-nums">{duration}</span> : null}
          </>,
          renderRows(item.id),
          item
        );
      }
      case "run-failed":
        return (
          <div
            key="run-failed"
            className={cn(ROW_DIVIDER, "px-3 py-1.5")}
            data-testid="spine-run-failed"
          >
            <div className="flex items-center gap-x-2">
              <ChevronDown className="size-3.5 shrink-0 text-control-placeholder" />
              <span className="text-control">
                {t("task-run.spine.run-failed")}
              </span>
            </div>
            <div className="whitespace-pre-wrap px-6 pt-1 text-error">
              {item.error}
            </div>
          </div>
        );
      case "applied-sections":
        return fold(
          item.id,
          t("task-run.spine.applied-sections", { count: item.sections.length }),
          item.sections.map(renderFile),
          item
        );
      case "pending-sections":
        return fold(
          item.id,
          t("task-run.spine.pending-sections", { count: item.sections.length }),
          item.sections.map(renderFile),
          item
        );
    }
  };

  if (items.length === 0) return null;
  return (
    <div className="w-full font-mono text-xs" data-testid="release-spine">
      <div className="w-full overflow-hidden rounded-sm border border-block-border bg-control-bg/50">
        {items.map(renderItem)}
      </div>
    </div>
  );
}

const NARROW_MEDIA_QUERY = "(max-width: 639px)";
const TRIGGER_ROW =
  "w-full justify-start bg-background px-3 py-1.5 text-control-light hover:bg-control-bg";

// A path opens the file viewer; where there is none it is plain text.
function FilePath({ label, onOpen }: { label: string; onOpen?: () => void }) {
  if (!onOpen) {
    return <span className="min-w-0 truncate font-mono text-xs">{label}</span>;
  }
  return (
    <Button
      type="button"
      appearance="link"
      size="sm"
      className="h-auto min-w-0 truncate p-0 font-mono text-xs"
      onClick={onOpen}
    >
      {label}
    </Button>
  );
}

function Chevron({ open }: { open: boolean }) {
  return (
    <ChevronRight
      className={cn(
        "size-3.5 shrink-0 text-control-placeholder transition-transform",
        open && "rotate-90"
      )}
    />
  );
}

function AppliedFold({
  fold,
  open,
  onToggle,
  onOpenFile,
  hasRun,
}: {
  fold: SpineAppliedFold;
  open: boolean;
  onToggle: () => void;
  onOpenFile?: OpenReleaseFile;
  hasRun: boolean;
}) {
  const { t } = useTranslation();
  const [shown, setShown] = useState(APPLIED_FOLD_PAGE_SIZE);
  const newest = fold.files[0]?.version ?? "";
  const oldest = fold.files.at(-1)?.version ?? "";
  return (
    <Collapsible open={open} onOpenChange={onToggle} className={ROW_DIVIDER}>
      <CollapsibleTrigger className={TRIGGER_ROW}>
        <Chevron open={open} />
        <span>
          {t("task-run.spine.already-applied", { count: fold.files.length })}
        </span>
        <span className="tabular-nums text-control-placeholder">
          {oldest === newest ? newest : `${oldest} – ${newest}`}
        </span>
      </CollapsibleTrigger>
      <CollapsiblePanel>
        <div className="flex flex-col py-1 pl-9">
          {fold.files.slice(0, shown).map((file) => (
            <div key={file.version} data-testid="spine-applied-row">
              <FilePath
                label={file.path || file.version}
                onOpen={
                  onOpenFile
                    ? () =>
                        onOpenFile(file, {
                          kind: "already-applied",
                          run: hasRun,
                        })
                    : undefined
                }
              />
            </div>
          ))}
          {shown < fold.files.length ? (
            <div>
              <Button
                type="button"
                appearance="link"
                size="sm"
                className="h-auto p-0 text-xs"
                onClick={() => setShown((n) => n + APPLIED_FOLD_PAGE_SIZE)}
              >
                {t("common.show-more")}
              </Button>
            </div>
          ) : null}
        </div>
      </CollapsiblePanel>
    </Collapsible>
  );
}
