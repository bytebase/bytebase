import type { TFunction } from "i18next";
import type { MouseEvent, ReactNode } from "react";
import { useCallback, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL } from "@/app/router/handles";
import { TaskStatusIcon } from "@/components/TaskStatusIcon";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useColumnWidths } from "@/hooks/useColumnWidths";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import {
  formatEnvironmentName,
  getTimeForPbTimestampProtoEs,
  unknownUser,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { extractPlanUID } from "@/utils";
import {
  extractStageUID,
  getStageStatusFromCounts,
} from "@/utils/v1/issue/rollout";
import { isReleaseBackedPlan } from "./plan-detail/utils/spec";
import {
  getPlanDraftState,
  getReviewBadge,
  type PlanDraftState,
  type ReviewBadge,
} from "./utils/reviewBadge";

export interface PlanColumn {
  key: string;
  title: ReactNode;
  defaultWidth: number;
  minWidth?: number;
  resizable?: boolean;
  render: (plan: Plan, ctx: PlanRowContext) => ReactNode;
}

export interface PlanRowContext {
  creator: { title: string; name: string };
  createTimeMs: number | undefined;
  updateTimeMs: number | undefined;
  approvalTag: { label: string; variant: ReviewBadge["variant"] } | undefined;
  isDeleted: boolean;
  draftState: PlanDraftState;
}

export function PlanLifecycleBadges({ ctx }: { ctx: PlanRowContext }) {
  const { t } = useTranslation();
  if (ctx.isDeleted) {
    return (
      <Badge variant="warning" className="shrink-0 font-normal">
        {t("common.closed")}
      </Badge>
    );
  }
  if (ctx.draftState === "draft") {
    return (
      <Badge className="shrink-0 font-normal text-control-light">
        {t("common.draft")}
      </Badge>
    );
  }
  if (ctx.draftState === "incomplete") {
    return (
      <Badge variant="destructive" className="shrink-0">
        {t("plan.lifecycle.incomplete")}
      </Badge>
    );
  }
  return null;
}

// Both plan lists render stage progress through this column so they cannot drift.
export function createPlanStagesColumn(t: TFunction): PlanColumn {
  return {
    key: "stages",
    title: t("rollout.stage.self", { count: 2 }),
    defaultWidth: 260,
    minWidth: 140,
    resizable: true,
    render: (plan) => <PlanStages plan={plan} />,
  };
}

function PlanStages({ plan }: { plan: Plan }) {
  // Subscribe so stage names re-render when the environment cache loads.
  const getEnvironmentByName = useAppStore((s) => s.getEnvironmentByName);
  void useAppStore((s) => s.environmentList);

  if (plan.rolloutStageSummaries.length === 0) {
    return <span className="text-control-light">-</span>;
  }
  return (
    <div className="flex items-center gap-1 flex-wrap">
      {plan.rolloutStageSummaries.map((summary, index) => {
        const envName = formatEnvironmentName(extractStageUID(summary.stage));
        const environment = getEnvironmentByName(envName);
        const stageStatus = getStageStatusFromCounts(summary.taskStatusCounts);
        return (
          <div key={summary.stage} className="flex items-center gap-1">
            <div className="flex items-center gap-1">
              <TaskStatusIcon size="tiny" status={stageStatus} />
              <span className="text-sm">{environment?.title || envName}</span>
            </div>
            {index < plan.rolloutStageSummaries.length - 1 && (
              <span className="mx-1 text-control-light">&rarr;</span>
            )}
          </div>
        );
      })}
    </div>
  );
}

const resolvePlanDetailUrl = (plan: Plan, projectId: string): string =>
  router.resolve({
    name: PROJECT_V1_ROUTE_PLAN_DETAIL,
    params: { projectId, planId: extractPlanUID(plan.name) },
  }).fullPath;

export function PlanTable({
  plans,
  projectId,
  columns,
  isMobile,
  resolvePlanUrl = resolvePlanDetailUrl,
  onOpenPlan,
}: {
  plans: Plan[];
  projectId: string;
  columns: PlanColumn[];
  isMobile: boolean;
  resolvePlanUrl?: (plan: Plan, projectId: string) => string;
  onOpenPlan: () => void;
}) {
  const { widths, totalWidth, onResizeStart, setWidths } =
    useColumnWidths(columns);

  // useColumnWidths seeds its state only on first render, so a viewport that
  // crosses the `sm` breakpoint after mount (resize / rotate) would keep the
  // stale name-column width. Re-seed from the rebuilt column defaults whenever
  // the breakpoint flips.
  const wasMobile = useRef(isMobile);
  useEffect(() => {
    if (wasMobile.current !== isMobile) {
      wasMobile.current = isMobile;
      setWidths(columns.map((c) => c.defaultWidth));
    }
  }, [isMobile, columns, setWidths]);

  return (
    <div className="overflow-x-auto rounded-sm border border-block-border">
      <Table className="table-fixed" style={{ minWidth: `${totalWidth}px` }}>
        <colgroup>
          {widths.map((w, i) => (
            <col key={columns[i].key} style={{ width: `${w}px` }} />
          ))}
        </colgroup>
        <TableHeader>
          <TableRow className="bg-control-bg">
            {columns.map((col, colIdx) => (
              <TableHead
                key={col.key}
                resizable={col.resizable}
                onResizeStart={
                  col.resizable ? (e) => onResizeStart(colIdx, e) : undefined
                }
              >
                {col.title}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {plans.map((plan) => (
            <PlanRow
              key={plan.name}
              plan={plan}
              columns={columns}
              planUrl={resolvePlanUrl(plan, projectId)}
              onOpenPlan={onOpenPlan}
            />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function PlanRow({
  plan,
  columns,
  planUrl,
  onOpenPlan,
}: Readonly<{
  plan: Plan;
  columns: PlanColumn[];
  planUrl: string;
  onOpenPlan: () => void;
}>) {
  const { t } = useTranslation();

  const isDeleted = plan.state === State.DELETED;
  const draftState = getPlanDraftState({
    approvalStatus: plan.approvalStatus,
    hasRollout: plan.hasRollout,
    isGitOpsPlan: isReleaseBackedPlan(plan.specs),
    issueName: plan.issue,
  });

  const creatorUser = useAppStore((state) =>
    state.getUserByIdentifier(plan.creator)
  );
  const creator = creatorUser || unknownUser(plan.creator);

  const onRowClick = useCallback(
    (e: MouseEvent) => {
      if (e.ctrlKey || e.metaKey) {
        window.open(planUrl, "_blank");
      } else {
        onOpenPlan();
        router.push(planUrl);
      }
    },
    [onOpenPlan, planUrl]
  );

  const approvalTag = useMemo(() => {
    const badge = getReviewBadge({
      hasIssue: plan.issue !== "",
      issueStatus: plan.issueStatus,
      hasRollout: plan.hasRollout,
      approvalStatus: plan.approvalStatus,
    });
    if (!badge) return undefined;
    return { label: t(badge.labelKey), variant: badge.variant };
  }, [plan.approvalStatus, plan.hasRollout, plan.issue, plan.issueStatus, t]);

  const ctx: PlanRowContext = {
    creator,
    createTimeMs: getTimeForPbTimestampProtoEs(plan.createTime),
    updateTimeMs: getTimeForPbTimestampProtoEs(plan.updateTime),
    approvalTag,
    isDeleted,
    draftState,
  };

  return (
    <TableRow
      data-testid="plan-list-item"
      data-scroll-restoration-anchor={plan.name}
      className={cn("cursor-pointer", isDeleted && "opacity-60")}
      onClick={onRowClick}
    >
      {columns.map((col) => (
        <TableCell key={col.key} className="overflow-hidden">
          {col.render(plan, ctx)}
        </TableCell>
      ))}
    </TableRow>
  );
}
