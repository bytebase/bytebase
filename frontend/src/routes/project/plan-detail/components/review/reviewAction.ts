export type ReviewAction = "APPROVE" | "COMMENT" | "REJECT";

export function isReviewSubmitDisabled({
  action,
  canApprove,
  commentMissing,
  loading,
}: {
  action: ReviewAction;
  canApprove: boolean;
  commentMissing: boolean;
  loading: boolean;
}): boolean {
  return (
    loading ||
    (action === "APPROVE" && !canApprove) ||
    (action === "COMMENT" && commentMissing) ||
    (action === "REJECT" && commentMissing)
  );
}
