import { create } from "@bufbuild/protobuf";
import {
  AlertCircle,
  CheckCircle,
  Loader2,
  RefreshCw,
  XCircle,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/connect";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";
import {
  extractUserEmail,
  pushNotification,
  useEnvironmentV1Store,
} from "@/store";
import { getTimeForPbTimestampProtoEs } from "@/types";
import {
  IssueSchema,
  IssueStatus,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  formatAbsoluteDateTime,
  getStageStatus,
  hasProjectPermissionV2,
  humanizeTs,
} from "@/utils";
import { usePlanDetailContext } from "../context/PlanDetailContext";
import { getPlanDetailSidebarStatusInfo } from "../utils/sidebarStatus";
import { PlanDetailSidebarApprovalFlow } from "./PlanDetailApprovalFlow";

export function PlanDetailMetadataSidebar() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { patchState, refreshState } = page;
  const environmentStore = useEnvironmentV1Store();
  const project = page.project;
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  const creatorEmail = useMemo(
    () => extractUserEmail(page.plan.creator),
    [page.plan.creator]
  );
  const createdTimeDisplay = useMemo(() => {
    const ts = getTimeForPbTimestampProtoEs(page.plan.createTime, 0);
    if (!ts) return "";
    void now;
    return humanizeTs(ts / 1000);
  }, [now, page.plan.createTime]);
  const statusInfo = useMemo(() => {
    return getPlanDetailSidebarStatusInfo({
      isCreating: page.isCreating,
      issue: page.issue,
      planState: page.plan.state,
      rollout: page.rollout,
      t,
    });
  }, [page.isCreating, page.issue, page.plan.state, page.rollout, t]);
  const planCheckSummary = useMemo(() => {
    const counts = page.plan.planCheckRunStatusCount || {};
    const running = counts.RUNNING || 0;
    const success = counts.SUCCESS || 0;
    const warning = counts.WARNING || 0;
    const error = (counts.ERROR || 0) + (counts.FAILED || 0);
    return { error, running, success, warning };
  }, [page.plan.planCheckRunStatusCount]);
  const allowChangeLabels = useMemo(() => {
    if (!project || !page.issue || page.issue.status !== IssueStatus.OPEN) {
      return false;
    }
    return hasProjectPermissionV2(project, "bb.issues.update");
  }, [page.issue, project]);
  const lastRefreshDisplay = useMemo(() => {
    if (!page.lastRefreshTime) return "";
    void now;
    const diff = Date.now() - page.lastRefreshTime;
    if (diff < 3_600_000) {
      return humanizeTs(Math.floor(page.lastRefreshTime / 1000));
    }
    return formatAbsoluteDateTime(page.lastRefreshTime);
  }, [now, page.lastRefreshTime]);

  return (
    <div className="flex w-full shrink-0 flex-col text-sm">
      <div className="flex flex-col gap-3 pb-3">
        {!page.isCreating && (
          <div className="flex flex-col">
            <p className="text-xs text-control-placeholder">
              {t("plan.sidebar.created-by-at", {
                time: createdTimeDisplay,
                user: creatorEmail,
              })}
            </p>
          </div>
        )}

        <div>
          <h4 className="mb-1 textinfolabel">{t("common.status")}</h4>
          <div className="flex items-center gap-2">
            <div
              className={cn("h-2.5 w-2.5 rounded-full", statusInfo.dotClass)}
            />
            <span className="text-sm font-medium text-main">
              {statusInfo.label}
            </span>
          </div>
        </div>

        {(planCheckSummary.running > 0 ||
          planCheckSummary.success > 0 ||
          planCheckSummary.warning > 0 ||
          planCheckSummary.error > 0) && (
          <div>
            <h4 className="mb-1 textinfolabel">{t("plan.navigator.checks")}</h4>
            <div className="flex flex-wrap items-center gap-3">
              {planCheckSummary.running > 0 && (
                <span className="flex items-center gap-1 text-control">
                  <Loader2 className="size-4 animate-spin" />
                  <span>{t("common.running")}</span>
                  <span>{planCheckSummary.running}</span>
                </span>
              )}
              {planCheckSummary.error > 0 && (
                <span className="flex items-center gap-1 text-error">
                  <XCircle className="size-4" />
                  <span>{t("common.error")}</span>
                  <span>{planCheckSummary.error}</span>
                </span>
              )}
              {planCheckSummary.warning > 0 && (
                <span className="flex items-center gap-1 text-warning">
                  <AlertCircle className="size-4" />
                  <span>{t("common.warning")}</span>
                  <span>{planCheckSummary.warning}</span>
                </span>
              )}
              {planCheckSummary.success > 0 && (
                <span className="flex items-center gap-1 text-success">
                  <CheckCircle className="size-4" />
                  <span>{t("common.success")}</span>
                  <span>{planCheckSummary.success}</span>
                </span>
              )}
            </div>
          </div>
        )}
      </div>

      {page.issue && (
        <div className="flex flex-col gap-3 border-t py-3">
          <PlanDetailSidebarApprovalFlow />
          <IssueLabelsSection
            issueLabels={project?.issueLabels ?? []}
            labels={page.issue.labels || []}
            allowChange={allowChangeLabels}
            onUpdate={async (labels) => {
              const issuePatch = create(IssueSchema, page.issue);
              issuePatch.labels = labels;
              try {
                const response = await issueServiceClientConnect.updateIssue(
                  create(UpdateIssueRequestSchema, {
                    issue: issuePatch,
                    updateMask: { paths: ["labels"] },
                  })
                );
                patchState({ issue: response });
                pushNotification({
                  module: "bytebase",
                  style: "SUCCESS",
                  title: t("common.updated"),
                });
              } catch (error) {
                pushNotification({
                  module: "bytebase",
                  style: "CRITICAL",
                  title: t("common.failed"),
                  description: String(error),
                });
              }
            }}
          />
        </div>
      )}

      {page.rollout && page.rollout.stages.length > 0 && (
        <div className="flex flex-col gap-3 border-t py-3">
          <h4 className="textinfolabel">
            {t("rollout.stage.self", { count: 2 })}
          </h4>
          {page.rollout.stages.map((stage) => {
            const completed = stage.tasks.filter(
              (task) =>
                task.status === Task_Status.DONE ||
                task.status === Task_Status.SKIPPED
            ).length;
            const env = environmentStore.getEnvironmentByName(
              stage.environment
            );
            const stageStatus = getStageStatus(stage);
            return (
              <div key={stage.name} className="flex items-center gap-2">
                <StageStatusDot status={stageStatus} />
                <span className="text-main">{env.title}</span>
                <span className="text-control-placeholder">
                  ({completed}/{stage.tasks.length})
                </span>
              </div>
            );
          })}
        </div>
      )}

      {!page.issue && (
        <div className="border-t pt-3">
          <p className="text-sm text-control-placeholder">
            {t("plan.sidebar.future-sections-hint")}
          </p>
        </div>
      )}

      {!page.isCreating && (
        <div className="pt-1">
          <div className="hidden items-center gap-2 sm:flex">
            {lastRefreshDisplay && (
              <Tooltip content={formatAbsoluteDateTime(page.lastRefreshTime)}>
                <span className="text-xs text-gray-400">
                  {lastRefreshDisplay}
                </span>
              </Tooltip>
            )}
            <Button
              className="opacity-60"
              disabled={page.isRefreshing}
              onClick={() => void refreshState()}
              size="xs"
              variant="ghost"
            >
              <RefreshCw
                className={cn("h-3 w-3", page.isRefreshing && "animate-spin")}
              />
              <span>{t("common.refresh")}</span>
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

function IssueLabelsSection({
  allowChange,
  issueLabels,
  labels,
  onUpdate,
}: {
  allowChange: boolean;
  issueLabels: Array<{ color: string; value: string }>;
  labels: string[];
  onUpdate: (labels: string[]) => Promise<void>;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);

  const options = useMemo(
    () =>
      issueLabels.map((label) => ({
        color: label.color,
        value: label.value,
      })),
    [issueLabels]
  );

  const toggleLabel = async (value: string) => {
    const next = labels.includes(value)
      ? labels.filter((label) => label !== value)
      : [...labels, value];
    try {
      setIsUpdating(true);
      await onUpdate(next);
    } finally {
      setIsUpdating(false);
    }
  };

  return (
    <div className="flex flex-col gap-y-1">
      <div className="flex items-center gap-x-1 textinfolabel">
        <span>{t("issue.labels")}</span>
      </div>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger
          render={
            <button
              className={cn(
                "flex min-h-9 w-full items-center justify-between gap-2 rounded-sm border border-control-border bg-white px-3 py-1.5 text-left text-sm transition-colors",
                allowChange && !isUpdating && "hover:bg-control-bg",
                open && "border-accent shadow-[0_0_0_1px_var(--color-accent)]",
                (!allowChange || isUpdating) && "cursor-not-allowed opacity-60"
              )}
              disabled={!allowChange || isUpdating}
              type="button"
            />
          }
        >
          <div className="flex min-w-0 flex-1 flex-wrap items-center gap-1.5">
            {labels.length > 0 ? (
              labels.map((value) => {
                const option = options.find((item) => item.value === value);
                return (
                  <span
                    key={value}
                    className="inline-flex items-center gap-1 rounded-xs bg-control-bg px-1.5 py-0.5 text-xs"
                  >
                    <span
                      className="size-2.5 shrink-0 rounded-sm"
                      style={{ backgroundColor: option?.color }}
                    />
                    <span className="truncate">{value}</span>
                  </span>
                );
              })
            ) : (
              <span className="text-control-placeholder">
                {t("common.select")}
              </span>
            )}
          </div>
        </PopoverTrigger>

        <PopoverContent
          side="bottom"
          align="start"
          initialFocus={false}
          finalFocus={false}
          style={{ width: "var(--anchor-width)" }}
          className="overflow-hidden bg-white p-0"
        >
          <div className="max-h-60 overflow-y-auto">
            {options.length === 0 ? (
              <div className="px-3 py-6 text-sm text-control-placeholder">
                {t("common.no-data")}
              </div>
            ) : (
              options.map((option) => {
                const isSelected = labels.includes(option.value);
                return (
                  <button
                    key={option.value}
                    className="flex w-full items-center gap-x-2 px-3 py-2 text-left text-sm transition-colors hover:bg-control-bg"
                    disabled={isUpdating}
                    onClick={() => void toggleLabel(option.value)}
                    type="button"
                  >
                    <Checkbox checked={isSelected} />
                    <span
                      className="size-4 shrink-0 rounded-sm"
                      style={{ backgroundColor: option.color }}
                    />
                    <span>{option.value}</span>
                  </button>
                );
              })
            )}
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}

function StageStatusDot({ status }: { status: Task_Status }) {
  const dotClass =
    status === Task_Status.DONE || status === Task_Status.SKIPPED
      ? "bg-success"
      : status === Task_Status.FAILED
        ? "bg-error"
        : status === Task_Status.RUNNING || status === Task_Status.PENDING
          ? "bg-accent"
          : "bg-control-placeholder";
  return <div className={cn("h-2.5 w-2.5 rounded-full", dotClass)} />;
}
