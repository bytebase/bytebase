import {
  LogEntity_Action,
  LogEntity_Level,
} from "@/types/proto/v1/logging_service";
import { FieldId } from "../plugins";
import { t } from "../plugins/i18n";
import { ExternalApprovalEvent } from "./externalApproval";
import {
  DatabaseId,
  InstanceId,
  IssueId,
  PrincipalId,
  SheetId,
  StageId,
  TaskId,
} from "./id";
import { IssueStatus } from "./issue";
import { MemberStatus, RoleType } from "./member";
import { StageStatusUpdateType, TaskStatus } from "./pipeline";
import { ApprovalEvent } from "./review";
import { Advice } from "./sqlAdvice";
import { VCSPushEvent } from "./vcs";

export function activityName(action: LogEntity_Action): string {
  switch (action) {
    case LogEntity_Action.ACTION_ISSUE_CREATE:
      return t("activity.type.issue-create");
    case LogEntity_Action.ACTION_ISSUE_COMMENT_CREATE:
      return t("activity.type.comment-create");
    case LogEntity_Action.ACTION_ISSUE_FIELD_UPDATE:
      return t("activity.type.issue-field-update");
    case LogEntity_Action.ACTION_ISSUE_STATUS_UPDATE:
      return t("activity.type.issue-status-update");
    case LogEntity_Action.ACTION_PIPELINE_STAGE_STATUS_UPDATE:
      return t("activity.type.pipeline-stage-status-update");
    case LogEntity_Action.ACTION_PIPELINE_TASK_STATUS_UPDATE:
      return t("activity.type.pipeline-task-status-update");
    case LogEntity_Action.ACTION_PIPELINE_TASK_RUN_STATUS_UPDATE:
      return t("activity.type.pipeline-task-run-status-update");
    case LogEntity_Action.ACTION_PIPELINE_TASK_FILE_COMMIT:
      return t("activity.type.pipeline-task-file-commit");
    case LogEntity_Action.ACTION_PIPELINE_TASK_STATEMENT_UPDATE:
      return t("activity.type.pipeline-task-statement-update");
    case LogEntity_Action.ACTION_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE:
      return t("activity.type.pipeline-task-earliest-allowed-time-update");
    case LogEntity_Action.ACTION_MEMBER_CREATE:
      return t("activity.type.member-create");
    case LogEntity_Action.ACTION_MEMBER_ROLE_UPDATE:
      return t("activity.type.member-role-update");
    case LogEntity_Action.ACTION_MEMBER_ACTIVATE:
      return t("activity.type.member-activate");
    case LogEntity_Action.ACTION_MEMBER_DEACTIVE:
      return t("activity.type.member-deactivate");
    case LogEntity_Action.ACTION_PROJECT_REPOSITORY_PUSH:
      return t("activity.type.project-repository-push");
    case LogEntity_Action.ACTION_PROJECT_DATABASE_TRANSFER:
      return t("activity.type.project-database-transfer");
    case LogEntity_Action.ACTION_PROJECT_MEMBER_CREATE:
      return t("activity.type.project-member-create");
    case LogEntity_Action.ACTION_PROJECT_MEMBER_DELETE:
      return t("activity.type.project-member-delete");
    case LogEntity_Action.ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE:
      return t("activity.type.database-recovery-pitr-done");
  }
  console.assert(false, `undefined text for activity type "${action}"`);
  return "";
}

export type ActivityLevel = "INFO" | "WARN" | "ERROR";

export type ActivityIssueCreatePayload = {
  issueName: string;
};

// TaskRollbackBy records an issue rollback activity.
// The task with taskID in IssueID is rollbacked by the task with RollbackByTaskID in RollbackByIssueID.
export type TaskRollbackBy = {
  issueId: IssueId;
  taskId: TaskId;
  rollbackByIssueId: IssueId;
  rollbackByTaskId: TaskId;
};

export type ActivityIssueCommentCreatePayload = {
  externalApprovalEvent?: ExternalApprovalEvent;
  issueName: string;
  taskRollbackBy?: TaskRollbackBy;
  approvalEvent?: ApprovalEvent;
};

export type ActivityIssueFieldUpdatePayload = {
  fieldId: FieldId;
  oldValue?: string;
  newValue?: string;
  issueName: string;
};

export type ActivityIssueStatusUpdatePayload = {
  oldStatus: IssueStatus;
  newStatus: IssueStatus;
  issueName: string;
};
export type ActivityStageStatusUpdatePayload = {
  stageId: StageId;
  stageStatusUpdateType: StageStatusUpdateType;
  issueName: string;
  stageName: string;
};

export type ActivityTaskStatusUpdatePayload = {
  taskId: TaskId;
  oldStatus: TaskStatus;
  newStatus: TaskStatus;
  issueName: string;
  taskName: string;
};

export type ActivityTaskFileCommitPayload = {
  taskId: TaskId;
  vcsInstanceUrl: string;
  repositoryFullPath: string;
  branch: string;
  filePath: string;
  commitId: string;
};

export type ActivityTaskStatementUpdatePayload = {
  taskId: TaskId;
  oldStatement: string;
  newStatement: string;
  oldSheetId: SheetId;
  newSheetId: SheetId;
  issueName: string;
  taskName: string;
};

export type ActivityTaskEarliestAllowedTimeUpdatePayload = {
  taskId: TaskId;
  oldEarliestAllowedTs: number;
  newEarliestAllowedTs: number;
  issueName: string;
  taskName: string;
};

export type ActivityMemberCreatePayload = {
  principalId: PrincipalId;
  principalName: string;
  principalEmail: string;
  memberStatus: MemberStatus;
  role: RoleType;
};

export type ActivityMemberRoleUpdatePayload = {
  principalId: PrincipalId;
  principalName: string;
  principalEmail: string;
  oldRole: RoleType;
  newRole: RoleType;
};

export type ActivityMemberActivateDeactivatePayload = {
  principalId: PrincipalId;
  principalName: string;
  principalEmail: string;
  role: RoleType;
};

export type ActivityProjectRepositoryPushPayload = {
  pushEvent: VCSPushEvent;
  issueId?: number;
  issueName?: string;
};

export type ActivityProjectDatabaseTransferPayload = {
  databaseId: number;
  databaseName: string;
};

export type ActivitySQLEditorQueryPayload = {
  statement: string;
  durationNs: number;
  instanceId: InstanceId;
  instanceName: string;
  databaseId: DatabaseId;
  databaseName: string;
  error: string;
  adviceList: Advice[];
};

export type ActionPayloadType =
  | ActivityIssueCreatePayload
  | ActivityIssueCommentCreatePayload
  | ActivityIssueFieldUpdatePayload
  | ActivityIssueStatusUpdatePayload
  | ActivityTaskStatusUpdatePayload
  | ActivityTaskFileCommitPayload
  | ActivityTaskStatementUpdatePayload
  | ActivityTaskEarliestAllowedTimeUpdatePayload
  | ActivityMemberCreatePayload
  | ActivityMemberRoleUpdatePayload
  | ActivityMemberActivateDeactivatePayload
  | ActivityProjectRepositoryPushPayload
  | ActivityProjectDatabaseTransferPayload
  | ActivitySQLEditorQueryPayload;

export interface FindActivityMessage {
  resource?: string;
  creatorEmail?: string;
  level?: LogEntity_Level[];
  action?: LogEntity_Action[];
  createdTsAfter?: number;
  createdTsBefore?: number;
  order?: "asc" | "desc";
  pageSize?: number;
  pageToken?: string;
}
