export type ExternalApprovalType = "bb.plugin.app.feishu";

export type ExternalApprovalEvent = {
  type: ExternalApprovalType;
  action: ExternalApprovalEventActionType;
  stageName: string;
};

export type ExternalApprovalEventActionType = "APPROVE" | "REJECT";
