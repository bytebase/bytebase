import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/react/components/ui/alert";
import { PlanCheckRun_Status } from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { useIssueDetailContext } from "../context/IssueDetailContext";
import { isApprovalCompleted } from "../utils/approval";
import { IssueDetailApprovalFlow } from "./IssueDetailApprovalFlow";
import { IssueDetailChecks } from "./IssueDetailChecks";
import { IssueDetailLabels } from "./IssueDetailLabels";

export function IssueDetailSidebar() {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const showChecks = useMemo(() => {
    const specs = page.plan?.specs ?? [];
    return (
      specs.length > 0 &&
      specs.every((spec) => spec.config?.case === "changeDatabaseConfig")
    );
  }, [page.plan?.specs]);
  const showChecksManualRolloutHint = useMemo(() => {
    if (!page.plan || !page.issue || page.plan.hasRollout) {
      return false;
    }
    if (!isApprovalCompleted(page.issue)) {
      return false;
    }
    const statusCount = page.plan.planCheckRunStatusCount ?? {};
    return (
      (statusCount.ERROR ?? 0) > 0 ||
      (statusCount.FAILED ?? 0) > 0 ||
      page.planCheckRuns.some(
        (run) =>
          run.status === PlanCheckRun_Status.FAILED ||
          run.results.some((result) => result.status === Advice_Level.ERROR)
      )
    );
  }, [page.issue, page.plan, page.planCheckRuns]);

  return (
    <div className="flex w-full flex-col gap-4 p-4">
      {showChecks && (
        <div className="flex flex-col gap-2">
          {showChecksManualRolloutHint && (
            <Alert
              variant="warning"
              description={t("issue.checks-manual-rollout-hint")}
            />
          )}
          <IssueDetailChecks />
        </div>
      )}
      <IssueDetailApprovalFlow />
      <IssueDetailLabels />
    </div>
  );
}
