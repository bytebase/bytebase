import { ChevronRight, RotateCw } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { buttonVariants } from "@/components/ui/button";
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { cn } from "@/lib/utils";
import { soleAttemptOf } from "./model";
import { SectionContent } from "./SectionContent";
import { SectionHeader } from "./SectionHeader";
import type { AttemptGroup, LogRow, Section } from "./types";

export interface LogRowsProps {
  rows: LogRow[];
  indent: boolean;
  // Whether the task run is still running: the log alone cannot tell an
  // attempt in flight from one whose server stopped.
  running: boolean;
  isRowExpanded: (rowId: string) => boolean;
  onToggleRow: (rowId: string) => void;
  foldOverrides: ReadonlyMap<string, boolean>;
  onFoldChange: (key: string, open: boolean) => void;
}

type RowState = Omit<LogRowsProps, "rows">;

export const ROW_DIVIDER = "border-block-border border-b last:border-b-0";

function SectionRow({
  section,
  indent,
  isRowExpanded,
  onToggleRow,
  foldOverrides,
  onFoldChange,
}: { section: Section } & RowState) {
  const expanded = isRowExpanded(section.id);
  return (
    <div className={ROW_DIVIDER}>
      <SectionHeader
        section={section}
        indent={indent}
        isExpanded={expanded}
        onToggle={() => onToggleRow(section.id)}
      />
      {expanded ? (
        <SectionContent
          section={section}
          indent={indent}
          foldOverrides={foldOverrides}
          onFoldChange={onFoldChange}
        />
      ) : null}
    </div>
  );
}

// The umbrella row and each attempt row inside it.
function HistoryRow({
  testId,
  className,
  expanded,
  onToggle,
  indent,
  label,
  note,
  duration,
  children,
}: {
  testId: string;
  className?: string;
  expanded: boolean;
  onToggle: () => void;
  indent: boolean;
  label: string;
  note?: string;
  duration: string;
  children: ReactNode;
}) {
  return (
    <Collapsible open={expanded} onOpenChange={onToggle} className={className}>
      <CollapsibleTrigger
        data-testid={testId}
        className={
          // History reads like the section rows' Button except for the rotate
          // icon and muted text: no fill, never a warning hue.
          buttonVariants({
            appearance: "secondary",
            size: "sm",
            className: cn(
              "w-full justify-start rounded-none bg-background text-control-light hover:bg-control-bg",
              indent ? "px-6" : "px-3"
            ),
          })
        }
      >
        <ChevronRight
          className={cn(
            "size-3.5 shrink-0 text-control-placeholder transition-transform",
            expanded && "rotate-90"
          )}
        />
        <RotateCw className="size-3.5 shrink-0" />
        <span>{label}</span>
        {note ? (
          <span className="min-w-0 truncate text-control-placeholder">
            {note}
          </span>
        ) : null}
        <span className="flex-1" />
        {duration ? <span className="tabular-nums">{duration}</span> : null}
      </CollapsibleTrigger>
      <CollapsiblePanel className="border-block-border border-t pl-4">
        {children}
      </CollapsiblePanel>
    </Collapsible>
  );
}

function PreviousAttemptsRow({
  group,
  ...state
}: { group: AttemptGroup } & RowState) {
  const { t } = useTranslation();
  const soleAttempt = soleAttemptOf(group);
  return (
    <HistoryRow
      testId="task-run-log-previous-attempts"
      className={ROW_DIVIDER}
      expanded={state.isRowExpanded(group.id)}
      onToggle={() => state.onToggleRow(group.id)}
      indent={state.indent}
      label={t("task-run.log-viewer.previous-attempts")}
      note={`${t("task-run.log-viewer.attempt-count", {
        count: group.attempts.length,
      })}${
        group.retrying && state.running
          ? ` · ${t("task-run.log-viewer.retrying")}`
          : ""
      }`}
      duration={group.duration}
    >
      {soleAttempt ? (
        <LogRows rows={soleAttempt.sections} {...state} />
      ) : (
        <ul role="list">
          {group.attempts.map((attempt) => (
            <li key={attempt.id} className={ROW_DIVIDER}>
              <HistoryRow
                testId="task-run-log-attempt"
                expanded={state.isRowExpanded(attempt.id)}
                onToggle={() => state.onToggleRow(attempt.id)}
                indent={state.indent}
                label={t("task-run.log-viewer.attempt-n", {
                  n: attempt.number,
                })}
                note={
                  attempt.reason
                    ? t("task-run.log-viewer.attempt-reason", {
                        reason: attempt.reason,
                      })
                    : undefined
                }
                duration={attempt.duration}
              >
                <LogRows rows={attempt.sections} {...state} />
              </HistoryRow>
            </li>
          ))}
        </ul>
      )}
    </HistoryRow>
  );
}

// One list of rows at any nesting level: sections, and umbrellas that open
// onto their own rows.
export function LogRows({ rows, ...state }: LogRowsProps) {
  return (
    <>
      {rows.map((row) =>
        row.kind === "section" ? (
          <SectionRow key={row.id} section={row} {...state} />
        ) : (
          <PreviousAttemptsRow key={row.id} group={row} {...state} />
        )
      )}
    </>
  );
}

export default LogRows;
