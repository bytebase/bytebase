import { Info, ShieldAlert, ShieldCheck } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Tooltip } from "@/components/ui/tooltip";
import { RiskLevel } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { extractIssueUID } from "@/utils/v1/issue/issue";

// The Review advance (Comment / Approve / Reject) lives in the page header's
// lifecycle slot now (BYT-9722), so this section header is title + metadata only
// — no duplicate Review action.
export function PlanReviewSectionHeader({ issue }: { issue: Issue }) {
  const { t } = useTranslation();
  const issueUID = extractIssueUID(issue.name);

  return (
    <div className="flex flex-col gap-y-1.5 px-4 pt-3 sm:flex-row sm:items-center sm:gap-x-2 sm:gap-y-0">
      <div className="flex shrink-0 items-center gap-x-1">
        <h3 className="text-base font-medium text-main">
          {t("issue.approval-flow.self")}
        </h3>
        {issue.approvalTemplate?.title?.trim() && (
          <Tooltip content={issue.approvalTemplate.title.trim()}>
            <Info className="size-3.5 text-control-light" />
          </Tooltip>
        )}
      </div>
      <div className="flex flex-wrap items-center gap-x-2 gap-y-1.5">
        <RiskChip riskLevel={issue.riskLevel} />
        {/* Static label, not a link: for change plans the issue route redirects
            back to this Plan Detail page (BYT-9721), so a link would be a no-op. */}
        <span className="inline-flex shrink-0 items-center rounded-full border px-2 py-0.5 text-xs text-control">
          {t("common.issue")} #{issueUID}
        </span>
      </div>
    </div>
  );
}

function RiskChip({ riskLevel }: { riskLevel: RiskLevel }) {
  const { t } = useTranslation();
  if (riskLevel === RiskLevel.RISK_LEVEL_UNSPECIFIED) return null;
  const label =
    riskLevel === RiskLevel.LOW
      ? t("issue.risk-level.low")
      : riskLevel === RiskLevel.MODERATE
        ? t("issue.risk-level.moderate")
        : t("issue.risk-level.high");
  const Icon = riskLevel === RiskLevel.LOW ? ShieldCheck : ShieldAlert;
  return (
    <span className="inline-flex shrink-0 items-center gap-x-1 rounded-full border px-2 py-0.5 text-xs text-control">
      <Icon
        className={
          riskLevel === RiskLevel.LOW
            ? "size-3.5 text-success"
            : riskLevel === RiskLevel.MODERATE
              ? "size-3.5 text-warning"
              : "size-3.5 text-error"
        }
      />
      {label}
    </span>
  );
}
