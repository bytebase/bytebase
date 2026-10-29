import { ChevronDown, ChevronRight } from "lucide-react";
import { type ReactNode, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import { cn } from "@/lib/utils";
import type {
  Task_ReleaseState,
  TaskRun_Status,
  TaskRunLogEntry,
  TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import { formatAbsoluteDateTime } from "@/utils";
import { LogRows, ROW_DIVIDER } from "./LogRows";
import {
  APPLIED_FOLD_PAGE_SIZE,
  assignEntryKeys,
  buildReleaseSpine,
  buildRowsFromEntries,
  findFailedCommand,
  firstLineOf,
  formatEntriesDuration,
  isSpineItemOpenByDefault,
  limitSpineSections,
  type TaskRunLogDetailText,
} from "./model";
import type {
  ReleaseSpineFile,
  SpineAppliedFold,
  SpineFileSection,
  SpineFileStatus,
  SpineItem,
  SpineStepSection,
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
  | { kind: "already-applied" }
  | { kind: "pending" };

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
  const [overrides, setOverrides] = useState<ReadonlyMap<string, boolean>>(
    () => new Map()
  );
  const [foldOverrides, setFoldOverrides] = useState<
    ReadonlyMap<string, boolean>
  >(() => new Map());
  const isOpen = (id: string, item: SpineItem) =>
    overrides.get(id) ?? isSpineItemOpenByDefault(item);
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
  const toggle = (id: string, item: SpineItem) =>
    setOverrides((prev) => new Map(prev).set(id, !isOpen(id, item)));

  const renderRows = (
    id: string,
    sectionEntries: TaskRunLogEntry[],
    version?: string
  ) => {
    const rows = buildRowsFromEntries(sectionEntries, {
      getSectionLabel,
      entryKeys,
      sheetsMap,
      detailText,
      fileVersion: version,
      idPrefix: id,
      taskRunStatus: run?.status,
    });
    return (
      <LogRows
        rows={rows}
        indent
        running={false}
        isRowExpanded={(rowId) => overrides.get(rowId) ?? true}
        onToggleRow={(rowId) =>
          setOverrides((prev) =>
            new Map(prev).set(rowId, !(prev.get(rowId) ?? true))
          )
        }
        foldOverrides={foldOverrides}
        onFoldChange={(key, open) =>
          setFoldOverrides((prev) => new Map(prev).set(key, open))
        }
      />
    );
  };

  const openContextOf = (section: SpineFileSection): ReleaseFileOpenContext => {
    if (section.status === "failed") {
      const failed = findFailedCommand(section.entries);
      return {
        kind: "failed",
        error: failed?.error || section.error || "",
        range: failed?.range,
      };
    }
    if (section.status === "pending") return { kind: "pending" };
    return { kind: "applied" };
  };

  const renderFile = (section: SpineFileSection) => {
    const file = { version: section.version, path: section.path };
    const pathLink = (
      <span className="flex min-w-0 items-center gap-x-1">
        <Button
          type="button"
          appearance="link"
          size="sm"
          className="h-auto min-w-0 truncate p-0 font-mono text-xs"
          onClick={() => onOpenFile?.(file, openContextOf(section))}
        >
          {section.path || section.version}
        </Button>
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
            {section.revisionDeleted
              ? t("task-run.spine.will-apply-again", {
                  date: section.revisionDeleted.time
                    ? formatAbsoluteDateTime(
                        Number(section.revisionDeleted.time.seconds) * 1000
                      )
                    : "",
                  user: section.revisionDeleted.deleter.replace(/^users\//, ""),
                })
              : t("task-run.spine.will-apply")}
          </Badge>
        </div>
      );
    }
    const open = isOpen(section.id, section);
    const duration = formatEntriesDuration(section.entries);
    return (
      <div key={section.id} className={ROW_DIVIDER}>
        <div className="flex items-center gap-x-2 px-3 py-1.5">
          <Button
            type="button"
            appearance="secondary"
            size="xs"
            aria-expanded={open}
            aria-label={section.version}
            className="h-auto gap-x-2 bg-transparent p-0 hover:bg-transparent"
            data-testid={`spine-file-${section.version}`}
            onClick={() => toggle(section.id, section)}
          >
            {open ? (
              <ChevronDown className="size-3.5 shrink-0 text-control-placeholder" />
            ) : (
              <ChevronRight className="size-3.5 shrink-0 text-control-placeholder" />
            )}
            <span className="tabular-nums text-control">{section.version}</span>
            <Badge variant={STATUS_VARIANT[section.status]}>
              {statusLabel(section.status)}
            </Badge>
          </Button>
          {pathLink}
          <span className="flex-1" />
          {duration ? (
            <span className="shrink-0 tabular-nums text-control-light">
              {duration}
            </span>
          ) : null}
        </div>
        {section.revisionDeleted ? (
          <div className="px-9 pb-1 text-control-light">
            {t("task-run.spine.revision-deleted", {
              user: section.revisionDeleted.deleter.replace(/^users\//, ""),
            })}
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
        {open ? (
          <div data-testid={`spine-file-${section.version}-content`}>
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
            {renderRows(section.id, section.entries, section.version)}
          </div>
        ) : null}
      </div>
    );
  };

  const renderStep = (step: SpineStepSection) => {
    const open = isOpen(step.id, step);
    const duration = formatEntriesDuration(step.entries);
    return (
      <div key={step.id} className={ROW_DIVIDER}>
        <Button
          type="button"
          appearance="secondary"
          size="sm"
          aria-expanded={open}
          className="w-full justify-start rounded-none bg-background px-3 text-left hover:bg-control-bg"
          onClick={() => toggle(step.id, step)}
        >
          {open ? (
            <ChevronDown className="size-3.5 shrink-0 text-control-placeholder" />
          ) : (
            <ChevronRight className="size-3.5 shrink-0 text-control-placeholder" />
          )}
          <span className="text-control">{getSectionLabel(step.type)}</span>
          <span className="flex-1" />
          {duration ? (
            <span className="tabular-nums text-control-light">{duration}</span>
          ) : null}
        </Button>
        {open ? renderRows(step.id, step.entries) : null}
      </div>
    );
  };

  const fold = (
    id: string,
    label: ReactNode,
    children: () => ReactNode,
    item: SpineItem,
    testId?: string
  ) => {
    const open = isOpen(id, item);
    return (
      <div key={id} className={ROW_DIVIDER} data-testid={testId}>
        <Button
          type="button"
          appearance="secondary"
          size="sm"
          aria-expanded={open}
          className="w-full justify-start rounded-none bg-background px-3 text-left text-control-light hover:bg-control-bg"
          onClick={() => toggle(id, item)}
        >
          {open ? (
            <ChevronDown className="size-3.5 shrink-0 text-control-placeholder" />
          ) : (
            <ChevronRight className="size-3.5 shrink-0 text-control-placeholder" />
          )}
          {label}
        </Button>
        {open ? children() : null}
      </div>
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
          />
        );
      case "file":
        return renderFile(item);
      case "step":
        return renderStep(item);
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
          () => item.sections.map(renderFile),
          item
        );
      case "pending-sections":
        return fold(
          item.id,
          t("task-run.spine.pending-sections", { count: item.sections.length }),
          () => item.sections.map(renderFile),
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

function AppliedFold({
  fold,
  open,
  onToggle,
  onOpenFile,
}: {
  fold: SpineAppliedFold;
  open: boolean;
  onToggle: () => void;
  onOpenFile?: OpenReleaseFile;
}) {
  const { t } = useTranslation();
  const [shown, setShown] = useState(APPLIED_FOLD_PAGE_SIZE);
  const newest = fold.files[0]?.version ?? "";
  const oldest = fold.files.at(-1)?.version ?? "";
  return (
    <div className={ROW_DIVIDER}>
      <Button
        type="button"
        appearance="secondary"
        size="sm"
        aria-expanded={open}
        className="w-full justify-start rounded-none bg-background px-3 text-left text-control-light hover:bg-control-bg"
        onClick={onToggle}
      >
        {open ? (
          <ChevronDown className="size-3.5 shrink-0 text-control-placeholder" />
        ) : (
          <ChevronRight className="size-3.5 shrink-0 text-control-placeholder" />
        )}
        <span>
          {t("task-run.spine.already-applied", { count: fold.files.length })}
        </span>
        <span className="tabular-nums text-control-placeholder">
          {oldest === newest ? newest : `${oldest} – ${newest}`}
        </span>
      </Button>
      {open ? (
        <div className="flex flex-col py-1 pl-9">
          {fold.files.slice(0, shown).map((file) => (
            <div key={file.version} data-testid="spine-applied-row">
              <Button
                type="button"
                appearance="link"
                size="sm"
                className="h-auto p-0 font-mono text-xs"
                onClick={() => onOpenFile?.(file, { kind: "already-applied" })}
              >
                {file.path || file.version}
              </Button>
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
      ) : null}
    </div>
  );
}
