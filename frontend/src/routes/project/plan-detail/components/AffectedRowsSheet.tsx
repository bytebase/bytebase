import { CircleX, TriangleAlert } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/components/HumanizeTs";
import { SegmentedControl } from "@/components/ui/segmented-control";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  type TableHeadSortDirection,
  TableRow,
} from "@/components/ui/table";
import { getTimeForPbTimestampProtoEs } from "@/types";
import {
  type AffectedRowsSummary,
  formatRows,
  needsLook,
  type TargetAffectedRows,
} from "../utils/affectedRows";
import { PlanTargetDisplay } from "./PlanTargetDisplay";

export type AffectedRowsFilter = "all" | "needs-look";

// Every database's estimate, with why any falls short.
export function AffectedRowsSheet({
  filter: initialFilter,
  onOpenChange,
  open,
  summary,
}: {
  filter: AffectedRowsFilter;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  summary: AffectedRowsSummary;
}) {
  const { t } = useTranslation();
  const [filter, setFilter] = useState(initialFilter);
  const [sortDir, setSortDir] = useState<TableHeadSortDirection>();
  const lookCount = summary.targets.filter(needsLook).length;
  const rows = useMemo(() => {
    const shown =
      filter === "needs-look"
        ? summary.targets.filter(needsLook)
        : summary.targets;
    if (!sortDir) return shown;
    return [...shown].sort((left, right) => {
      const order =
        left.rows < right.rows ? -1 : left.rows > right.rows ? 1 : 0;
      return sortDir === "asc" ? order : -order;
    });
  }, [filter, sortDir, summary.targets]);

  return (
    <Sheet onOpenChange={onOpenChange} open={open}>
      <SheetContent width="standard">
        <SheetHeader>
          <SheetTitle>{t("plan.affected-rows.self")}</SheetTitle>
          <SheetDescription className="flex flex-wrap items-center gap-x-1">
            {t("plan.affected-rows.estimated-on", {
              count: summary.targets.length,
            })}
            {summary.time && (
              <>
                <span>·</span>
                <HumanizeTs tsMs={getTimeForPbTimestampProtoEs(summary.time)} />
              </>
            )}
          </SheetDescription>
        </SheetHeader>
        <SheetBody className="gap-y-4">
          {lookCount > 0 && (
            <SegmentedControl
              ariaLabel={t("plan.affected-rows.filter")}
              onValueChange={setFilter}
              options={[
                {
                  value: "all",
                  label: t("plan.affected-rows.filter-all", {
                    n: summary.targets.length,
                  }),
                },
                {
                  value: "needs-look",
                  label: t("plan.affected-rows.filter-incomplete", {
                    n: lookCount,
                  }),
                },
              ]}
              size="sm"
              value={filter}
            />
          )}
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("common.database")}</TableHead>
                <TableHead
                  className="whitespace-nowrap text-right"
                  onSort={() =>
                    setSortDir((dir) => (dir === "desc" ? "asc" : "desc"))
                  }
                  sortActive={sortDir !== undefined}
                  sortDir={sortDir}
                  sortable
                >
                  {t("plan.affected-rows.self")}
                </TableHead>
                <TableHead>{t("plan.affected-rows.estimate")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((target) => (
                <TableRow key={target.target}>
                  <TableCell className="align-top">
                    <PlanTargetDisplay showEnvironment target={target.target} />
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-right align-top tabular-nums">
                    {target.status === "not-estimated"
                      ? "—"
                      : formatRows(target.rows)}
                  </TableCell>
                  <TableCell className="align-top">
                    <EstimateCell target={target} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <p className="text-xs text-control-light">
            {t("plan.affected-rows.explanation")}
          </p>
        </SheetBody>
      </SheetContent>
    </Sheet>
  );
}

function EstimateCell({ target }: { target: TargetAffectedRows }) {
  const { t } = useTranslation();
  if (target.status === "estimated") {
    return (
      <span className="text-control-light">
        {t("plan.affected-rows.estimated")}
      </span>
    );
  }
  const head =
    target.status === "not-estimated" ? (
      <span className="inline-flex items-center gap-x-1 text-error">
        <CircleX aria-hidden="true" className="size-3.5 shrink-0" />
        {t("plan.affected-rows.not-estimated")}
      </span>
    ) : (
      <span className="inline-flex items-center gap-x-1 text-warning-hover">
        <TriangleAlert
          aria-hidden="true"
          className="size-3.5 shrink-0 text-warning"
        />
        {t("plan.affected-rows.incomplete")}
      </span>
    );
  const detail = target.status === "not-estimated" ? target.title : "";
  return (
    <div className="flex flex-col gap-y-1">
      {head}
      {detail && <span className="text-xs text-control">{detail}</span>}
      {target.reason && (
        <span className="break-words text-xs text-control-light">
          {target.reason}
        </span>
      )}
    </div>
  );
}
