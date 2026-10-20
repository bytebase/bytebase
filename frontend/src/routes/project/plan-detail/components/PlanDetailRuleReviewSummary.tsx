import { AlertCircle, CheckCircle, Loader2, XCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/components/HumanizeTs";
import { Button } from "@/components/ui/button";
import { ReviewRun_Status } from "@/types/proto-es/v1/issue_service_pb";
import { getTimeForPbTimestampProtoEs } from "@/types/timestamp";
import type { RuleReviewRunState } from "../hooks/useRuleReviewRun";

// The standard rule review's part of the Checks summary line.
export function PlanDetailRuleReviewSummary({
  state,
}: {
  state: RuleReviewRunState;
}) {
  const { t } = useTranslation();
  const { loadFailed, openFindings, run } = state;
  if (loadFailed) {
    return (
      <span className="flex items-center gap-1 text-sm">
        <span className="text-error">{t("plan.rule-review.load-failed")}</span>
        <Button appearance="link" onClick={() => void state.reload()} size="xs">
          {t("sql-review.standard-rules.retry")}
        </Button>
      </span>
    );
  }
  if (run === undefined) return null;
  if (run === null) {
    return (
      <span className="text-sm text-control-placeholder">
        {t("plan.rule-review.not-run")}
      </span>
    );
  }
  switch (run.status) {
    case ReviewRun_Status.AVAILABLE:
    case ReviewRun_Status.RUNNING:
      return (
        <span className="flex items-center gap-1 text-sm text-control">
          <Loader2 className="size-4 animate-spin" />
          {run.status === ReviewRun_Status.AVAILABLE
            ? t("plan.rule-review.queued")
            : t("plan.rule-review.running")}
        </span>
      );
    case ReviewRun_Status.DONE:
      return (
        <span className="flex items-center gap-x-3 text-sm">
          {openFindings > 0 ? (
            <span className="flex items-center gap-1 text-warning">
              <AlertCircle className="size-4" />
              {t("plan.rule-review.open-findings", { count: openFindings })}
            </span>
          ) : (
            <span className="flex items-center gap-1 text-success">
              <CheckCircle className="size-4" />
              {t("plan.rule-review.no-open-findings")}
            </span>
          )}
          <HumanizeTs
            className="text-xs text-control-placeholder"
            tsMs={getTimeForPbTimestampProtoEs(run.endTime)}
          />
        </span>
      );
    case ReviewRun_Status.FAILED:
      return (
        <span className="flex basis-full flex-col gap-y-1 text-sm">
          <span className="flex items-center gap-1 text-error">
            <XCircle className="size-4" />
            {t("plan.rule-review.failed")}
          </span>
          <span className="text-xs break-words text-control-light">
            {run.error}
          </span>
        </span>
      );
    default:
      return null;
  }
}
