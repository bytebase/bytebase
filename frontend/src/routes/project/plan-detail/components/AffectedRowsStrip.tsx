import { CircleX, TriangleAlert } from "lucide-react";
import { type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/components/HumanizeTs";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { getTimeForPbTimestampProtoEs } from "@/types";
import { usePlanCheckActions } from "../hooks/usePlanCheckActions";
import { useSqlReviewV2 } from "../hooks/useSqlReviewV2";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import {
  type AffectedRowsSummary,
  databaseLabel,
  formatRows,
  needsLook,
} from "../utils/affectedRows";
import {
  type AffectedRowsFilter,
  AffectedRowsSheet,
} from "./AffectedRowsSheet";

// Beyond this many databases the strip names only the largest estimate.
const LISTED_DATABASES = 3;

// The statement report's estimate of the rows the SQL changes, per database,
// under the SQL it describes. It stays neutral: a problem shows only as a
// small icon and a colored word, and its reason lives in Details.
export function AffectedRowsStrip({
  onGoToLine,
  summary,
}: {
  onGoToLine?: (line: number) => void;
  summary: AffectedRowsSummary;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { allowRunChecks, isRunningChecks, runChecks } = usePlanCheckActions();
  const review = useSqlReviewV2()?.review;
  const [detailsState, setDetailsState] = useState<{
    open: boolean;
    filter: AffectedRowsFilter;
    // Remounts the sheet's contents on each open, so its filter starts over.
    key: number;
  }>({ open: false, filter: "all", key: 0 });
  const openDetails = (filter: AffectedRowsFilter) =>
    setDetailsState((prev) => ({ open: true, filter, key: prev.key + 1 }));

  const { targets } = summary;
  const lookCount = targets.filter(needsLook).length;
  const figureTargets = targets.filter(
    (target) => target.status !== "not-estimated"
  );
  const time = summary.time && (
    <HumanizeTs
      className="text-control-light"
      tsMs={getTimeForPbTimestampProtoEs(summary.time)}
    />
  );
  const label = (
    <span className="font-medium text-control">
      {t("plan.affected-rows.self")}
    </span>
  );

  const details = (
    <>
      <InlineAction onClick={() => openDetails("all")}>
        {t("plan.affected-rows.details")}
      </InlineAction>
      <AffectedRowsSheet
        filter={detailsState.filter}
        key={detailsState.key}
        onOpenChange={(open) => setDetailsState((prev) => ({ ...prev, open }))}
        open={detailsState.open}
        summary={summary}
      />
    </>
  );

  if (figureTargets.length === 0) {
    const first = targets[0];
    const errorLine = first.line;
    return (
      <Strip>
        {label}
        <span className="inline-flex items-center gap-x-1 text-error">
          <CircleX aria-hidden="true" className="size-3.5 shrink-0" />
          {t("plan.affected-rows.not-estimated")}
        </span>
        <span className="min-w-0 truncate text-control-light">
          {first.title}
        </span>
        <span className="flex-1" />
        {errorLine !== undefined && onGoToLine && (
          <InlineAction onClick={() => onGoToLine(errorLine)}>
            {t("plan.affected-rows.go-to-line", { line: errorLine })}
          </InlineAction>
        )}
        {details}
      </Strip>
    );
  }

  const listsEach = figureTargets.length <= LISTED_DATABASES;
  const largest = figureTargets.reduce((max, target) =>
    target.rows > max.rows ? target : max
  );
  const figures: ReactNode[] = listsEach
    ? figureTargets.map((target) => (
        <span key={target.target}>
          {t("plan.affected-rows.on", {
            rows: formatRows(target.rows),
            database: databaseLabel(target.target),
          })}
        </span>
      ))
    : [
        <span key="largest">
          {t("plan.affected-rows.up-to-on", {
            rows: formatRows(largest.rows),
            database: databaseLabel(largest.target),
          })}
        </span>,
        <span key="count">
          {t("plan.affected-rows.n-databases", { count: targets.length })}
        </span>,
      ];
  // Rerunning helps when EXPLAIN failed, as for a missing privilege; a syntax
  // error or an oversized sheet needs a change to the SQL.
  const showRerun =
    allowRunChecks &&
    !page.isCreating &&
    targets.some((target) => target.status === "incomplete");

  return (
    <Strip>
      {label}
      {lookCount > 0 && (
        <InlineAction
          className="text-warning-hover"
          onClick={() => openDetails("needs-look")}
        >
          <TriangleAlert
            aria-hidden="true"
            className="size-3.5 shrink-0 text-warning"
          />
          {listsEach
            ? t("plan.affected-rows.incomplete")
            : t("plan.affected-rows.incomplete-on", { count: lookCount })}
        </InlineAction>
      )}
      {figures.map((figure, index) => (
        <span className="inline-flex items-center gap-x-2" key={index}>
          {index > 0 && <span className="text-control-light">·</span>}
          {figure}
        </span>
      ))}
      {time}
      <span className="flex-1" />
      {showRerun && (
        <InlineAction
          disabled={isRunningChecks}
          onClick={() => void runChecks().then(() => review?.reload())}
        >
          {t("plan.affected-rows.rerun")}
        </InlineAction>
      )}
      {details}
    </Strip>
  );
}

function Strip({ children }: { children: ReactNode }) {
  return (
    <div
      className="flex flex-wrap items-center gap-x-2 gap-y-1 rounded-b-sm border border-t-0 border-control-border bg-background px-3 py-1.5 text-xs text-control"
      data-testid="affected-rows"
    >
      {children}
    </div>
  );
}

function InlineAction({
  children,
  className,
  disabled,
  onClick,
}: {
  children: ReactNode;
  className?: string;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <Button
      appearance="link"
      className={cn("h-auto px-0", className)}
      disabled={disabled}
      onClick={onClick}
      size="xs"
    >
      {children}
    </Button>
  );
}
