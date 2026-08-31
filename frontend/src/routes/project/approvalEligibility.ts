import type { Project } from "@/types/proto-es/v1/project_service_pb";

export type ApprovalIneligibility = "last-plan-editor" | "self-approval";

export interface ApprovalEligibility {
  canApprove: boolean;
  canReview: boolean;
  ineligibilities: ApprovalIneligibility[];
}

export function getApprovalEligibility({
  actor,
  issueCreator,
  lastPlanEditor,
  project,
}: {
  actor: string;
  issueCreator: string;
  lastPlanEditor: string;
  project: Pick<Project, "allowLastPlanEditorApproval" | "allowSelfApproval">;
}): ApprovalEligibility {
  const ineligibilities: ApprovalIneligibility[] = [];

  if (!project.allowSelfApproval && actor === issueCreator) {
    ineligibilities.push("self-approval");
  }
  if (!project.allowLastPlanEditorApproval && actor === lastPlanEditor) {
    ineligibilities.push("last-plan-editor");
  }

  return {
    // Self-approval is a review-wide restriction: the backend rejects both an
    // approval and a rejection. Last-plan-editor is deliberately narrower,
    // so that editor can still request changes.
    canApprove: ineligibilities.length === 0,
    canReview: !ineligibilities.includes("self-approval"),
    ineligibilities,
  };
}
