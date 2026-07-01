import { Ban, CircleCheck, CircleX, Clock3, Loader2 } from "lucide-react";
import { Fragment, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { BypassAndDeploySheet, useBypassDeploy } from "./bypassDeploy";

export function ReviewReadinessFooter({
  issue,
  plan,
}: {
  issue: Issue;
  plan: Plan;
}) {
  const { t } = useTranslation();
  const bypass = useBypassDeploy(issue, plan);
  const { state, checks, weight, creating, triggerDeploy } = bypass;

  if (state.kind === "hidden") return null;

  // Each non-zero count is its own "·"-separated segment with a leading dot, so
  // empty buckets (e.g. "0 checks passed") are dropped rather than shown.
  const checkSegments: { key: string; node: ReactNode }[] = [];
  if (checks.error > 0) {
    checkSegments.push({
      key: "failed",
      node: (
        <span className="whitespace-nowrap text-error">
          {t("plan.review.footer.n-checks-failed", { count: checks.error })}
        </span>
      ),
    });
  }
  if (checks.success > 0) {
    checkSegments.push({
      key: "passed",
      node: (
        <span className="whitespace-nowrap">
          {t("plan.review.footer.n-checks-passed", { count: checks.success })}
        </span>
      ),
    });
  }
  const checkCounts = checkSegments.map((segment) => (
    <Fragment key={segment.key}>
      <span>·</span>
      {segment.node}
    </Fragment>
  ));

  return (
    <div
      className={
        state.kind === "approved-checks-failed"
          ? "flex flex-wrap items-center gap-x-2 gap-y-1 border-t px-4 py-2.5"
          : "flex flex-wrap items-center gap-x-2 gap-y-1 border-t px-4 py-2.5 text-sm text-control-placeholder"
      }
    >
      {state.kind === "waiting-review" && (
        <>
          <Clock3 className="size-4 shrink-0" />
          <span className="whitespace-nowrap font-medium text-control">
            {t("plan.review.footer.waiting-on-review")}
          </span>
          <span className="hidden sm:inline">·</span>
          <span className="hidden whitespace-nowrap sm:inline">
            {checks.error > 0
              ? t("plan.review.footer.rollout-blocked-by-failed-checks")
              : t("plan.review.footer.auto-rollout-after-approval")}
          </span>
          {checkCounts}
        </>
      )}
      {state.kind === "all-gates-passed" && (
        <>
          <CircleCheck className="size-4 shrink-0 text-success" />
          <span className="whitespace-nowrap font-medium text-control">
            {t("plan.review.footer.all-gates-passed")}
          </span>
          <span className="hidden sm:inline">·</span>
          <span className="hidden whitespace-nowrap sm:inline">
            {t("plan.review.footer.creating-rollout-automatically")}
          </span>
          {checkCounts}
        </>
      )}
      {state.kind === "approved-checks-failed" && (
        <>
          <CircleX className="size-6 shrink-0 text-error" />
          <div className="min-w-0 flex-1">
            <div className="text-sm font-semibold text-main">
              {t("plan.review.footer.approved-but-checks-failed")}
            </div>
            <div className="text-xs text-control-placeholder">
              {t("plan.review.footer.errors-passed-not-created", {
                count: checks.error,
                passed: checks.success,
              })}
            </div>
          </div>
        </>
      )}
      {state.kind === "rejected" && (
        <>
          <Ban className="size-4 shrink-0" />
          <span className="min-w-0 truncate">
            {t("plan.review.footer.blocked-by-rejection")}
          </span>
        </>
      )}

      {weight === "link" && (
        <button
          className="ml-auto inline-flex shrink-0 items-center gap-x-1 text-xs text-control-placeholder underline hover:text-control disabled:opacity-60"
          disabled={creating}
          onClick={triggerDeploy}
          type="button"
        >
          {creating && <Loader2 className="size-3 animate-spin" />}
          {t("plan.review.footer.bypass-and-deploy")}
        </button>
      )}
      {weight === "button" && (
        <Button
          className="ml-auto shrink-0"
          disabled={creating}
          onClick={triggerDeploy}
          variant="outline"
        >
          {creating && <Loader2 className="size-4 animate-spin" />}
          {t("plan.review.footer.bypass-and-deploy")}
        </Button>
      )}

      <BypassAndDeploySheet issue={issue} bypass={bypass} />
    </div>
  );
}
