import type { ConditionGroupExpr } from "@/plugins/cel";
import type { ApprovalFlow } from "@/types/proto-es/v1/issue_service_pb";
import type { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";

// A single approval rule with inline flow definition
export type LocalApprovalRule = {
  uid: string; // Local unique identifier for UI tracking
  source: WorkspaceApprovalSetting_Rule_Source;
  title: string; // Human-readable title
  description: string; // Description of the rule
  condition: string; // CEL expression string
  conditionExpr?: ConditionGroupExpr; // Parsed CEL for editor
  flow: ApprovalFlow; // Inline approval flow (roles array)
};

// The local config is just a list of rules per source
export type LocalApprovalConfig = {
  rules: LocalApprovalRule[];
};
