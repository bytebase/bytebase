import {
  extractReviewContext,
  useWrappedReviewStepsV1,
} from "@/components/IssueV1/logic";
import { ComposedIssue, IssueReviewStatus, UIIssueFilter } from "@/types";
import { Issue_Approver_Status } from "@/types/proto/v1/issue_service";

export const filterIssueByApprover = (
  issue: ComposedIssue,
  approver: string | undefined
) => {
  if (!approver) return true;

  const reviewContext = extractReviewContext(issue);
  const steps = useWrappedReviewStepsV1(issue, reviewContext);

  const currentStep = steps.value?.find((step) => step.status === "CURRENT");
  if (!currentStep) return false;
  // We support "approver:{email}" by now
  // Planning to support "approver:[{email_1}, {email_2}, ...]" and
  // "approver:roles/{role}" in the future
  return (
    currentStep.candidates.findIndex((user) => user.email === approver) >= 0
  );
};

export const filterIssueByReviewStatus = (
  issue: ComposedIssue,
  status: IssueReviewStatus | undefined
) => {
  if (!status) return true;

  const reviewContext = extractReviewContext(issue);
  if (status === "pending_review") {
    return reviewContext.status.value === Issue_Approver_Status.PENDING;
  }
  if (status === "approved") {
    return reviewContext.status.value === Issue_Approver_Status.APPROVED;
  }

  console.error(
    "[filterIssueByReviewStatus] should never reach this line",
    status
  );
  return false;
};

export const applyUIIssueFilter = (
  list: ComposedIssue[],
  filter: UIIssueFilter | undefined
) => {
  if (!filter) return list;
  return list
    .filter((issue) => filterIssueByApprover(issue, filter.approver))
    .filter((issue) => filterIssueByReviewStatus(issue, filter.review_status));
};
