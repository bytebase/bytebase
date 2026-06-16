import { ChevronDown, ShieldAlert, ShieldCheck } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { RouterLink } from "@/react/components/RouterLink";
import { Button } from "@/react/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/react/router/handles";
import { ApprovalStatus, RiskLevel } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { extractIssueUID } from "@/utils/v1/issue/issue";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { ReviewActionPopover } from "./ReviewActionPopover";
import { deriveSteps } from "./ReviewApprovalFlow";
import { useApprovalCandidates } from "./useApprovalCandidates";

export function PlanReviewSectionHeader({ issue }: { issue: Issue }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const [open, setOpen] = useState(false);
  const issueUID = extractIssueUID(issue.name);

  const currentRole = useMemo(() => {
    const steps = deriveSteps(issue);
    return steps.find((s) => s.status === "current")?.role ?? "";
  }, [issue]);
  const { isCurrentUserCandidate } = useApprovalCandidates(
    issue,
    page.projectId,
    currentRole
  );

  const showReview =
    !page.readonly &&
    issue.status === IssueStatus.OPEN &&
    issue.approvalStatus === ApprovalStatus.PENDING &&
    currentRole !== "" &&
    isCurrentUserCandidate;

  return (
    // The Review button stays pinned top-right on the first row; the title and
    // its badges share the remaining space and wrap (badges drop to a second
    // row) when the width is too narrow to fit them beside the button.
    <div className="flex items-start gap-x-2 px-4 pt-3 sm:items-center">
      <div className="flex min-w-0 flex-1 flex-col gap-y-1.5 sm:flex-row sm:items-center sm:gap-x-2 sm:gap-y-0">
        <h3 className="shrink-0 text-base font-medium text-main">
          {t("issue.approval-flow.self")}
        </h3>
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1.5">
          <RiskChip riskLevel={issue.riskLevel} />
          <RouterLink
            className="inline-flex shrink-0 items-center rounded-full border px-2 py-0.5 text-xs text-control hover:border-control-border"
            to={{
              name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
              params: { issueId: issueUID, projectId: page.projectId },
            }}
          >
            {t("common.issue")} #{issueUID}
          </RouterLink>
        </div>
      </div>
      {showReview && (
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger render={<Button className="shrink-0 gap-x-1.5" />}>
            {t("plan.review.action")}
            <ChevronDown className="size-4" />
          </PopoverTrigger>
          <PopoverContent align="end" className="px-4 py-4">
            <ReviewActionPopover issue={issue} onClose={() => setOpen(false)} />
          </PopoverContent>
        </Popover>
      )}
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
