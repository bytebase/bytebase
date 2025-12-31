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

// Action type definitions
export type IssueReviewAction = "ISSUE_REVIEW";

export type IssueStatusAction =
  | "ISSUE_STATUS_CLOSE"
  | "ISSUE_STATUS_REOPEN"
  | "ISSUE_STATUS_RESOLVE";

export type IssueAction =
  | IssueReviewAction
  | IssueStatusAction
  | "ISSUE_CREATE";

export type PlanAction = "PLAN_CLOSE" | "PLAN_REOPEN";

export type RolloutAction =
  | "ROLLOUT_CREATE"
  | "ROLLOUT_START"
  | "ROLLOUT_CANCEL";

export type ExportAction = "EXPORT_DOWNLOAD";

export type UnifiedAction =
  | IssueAction
  | PlanAction
  | RolloutAction
  | ExportAction;

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
  // Plans where rollout is created on-demand (export, create database)
  hasDeferredRollout: boolean;
  isCreator: boolean;
  issueApproved: boolean; // approval is APPROVED or SKIPPED
  exportArchiveReady: boolean;
  allTasksFinished: boolean;
  hasDatabaseCreateOrExportTasks: boolean;
  hasStartableTasks: boolean;
  hasRunningTasks: boolean;

  // Grouped
  permissions: ActionPermissions;
  validation: ActionValidation;
}

export type ActionCategory = "primary" | "secondary";

export interface ActionDefinition {
  id: UnifiedAction;
  label: (ctx: ActionContext) => string;
  buttonType: "primary" | "success" | "default";
  category: ActionCategory | ((ctx: ActionContext) => ActionCategory);
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
