import type { ComputedRef } from "vue";
import type { State } from "@/types/proto-es/v1/common_pb";
import type {
  Issue,
  Issue_ApprovalStatus,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import type { UnifiedAction } from "../unified/types";

// Re-export action types from unified
export type {
  ExportAction,
  IssueAction,
  IssueReviewAction,
  IssueStatusAction,
  PlanAction,
  RolloutAction,
  UnifiedAction,
} from "../unified/types";

export type ExecuteType =
  | "immediate"
  | "confirm-dialog"
  | "popover:labels"
  | "popover:review"
  | "panel:issue-status"
  | "panel:rollout";

export interface ActionPermissions {
  updatePlan: boolean;
  createIssue: boolean;
  updateIssue: boolean;
  createRollout: boolean;
  runTasks: boolean;
  isApprovalCandidate: boolean;
  canApprove: boolean;
  canReject: boolean;
}

export interface ActionValidation {
  hasEmptySpec: boolean;
  planChecksRunning: boolean;
  planChecksFailed: boolean;
}

export interface ActionContext {
  // Entities
  plan: Plan;
  issue: Issue | undefined;
  rollout: Rollout | undefined;
  project: Project;

  // Proto enums directly
  planState: State;
  issueStatus: IssueStatus | undefined;
  approvalStatus: Issue_ApprovalStatus | undefined;

  // Derived flags
  isCreating: boolean;
  isIssueOnly: boolean;
  isExportPlan: boolean;
  isCreator: boolean;
  exportArchiveReady: boolean;
  allTasksFinished: boolean;
  hasDatabaseCreateOrExportTasks: boolean;
  hasStartableTasks: boolean;
  hasRunningTasks: boolean;
  rolloutPreconditionsMet: boolean;

  // Grouped
  permissions: ActionPermissions;
  validation: ActionValidation;
}

export interface ActionDefinition {
  id: UnifiedAction;
  label: (ctx: ActionContext) => string;
  buttonType: "primary" | "success" | "default";
  category: "primary" | "secondary";
  priority: number;

  isVisible: (ctx: ActionContext) => boolean;
  isDisabled: (ctx: ActionContext) => boolean;
  disabledReason: (ctx: ActionContext) => string | undefined;

  executeType: ExecuteType;
  execute?: (ctx: ActionContext) => Promise<void>;
  confirmTitle?: (ctx: ActionContext) => string;
  confirmContent?: (ctx: ActionContext) => string;
}

export interface ActionRegistryReturn {
  context: ComputedRef<ActionContext>;
  primaryAction: ComputedRef<ActionDefinition | undefined>;
  secondaryActions: ComputedRef<ActionDefinition[]>;
  isActionDisabled: (action: ActionDefinition) => boolean;
  getDisabledReason: (action: ActionDefinition) => string | undefined;
}
