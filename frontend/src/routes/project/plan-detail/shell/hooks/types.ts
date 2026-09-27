import { type Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan, PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  type Rollout,
  type TaskRun,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import type { PlanDetailPhase } from "../../shared/stores/types";

export type { PlanDetailPhase } from "../../shared/stores/types";

export interface PlanDetailPageSnapshot {
  projectId: string;
  planId: string;
  specId?: string;
  pageKey: string;
  projectTitle: string;
  projectRequireIssueApproval: boolean;
  projectRequirePlanCheckNoError: boolean;
  projectCanCreateRollout: boolean;
  currentUser: User;
  project: Project;
  isCreating: boolean;
  isInitializing: boolean;
  ready: boolean;
  readonly: boolean;
  plan: Plan;
  issue?: Issue;
  rollout?: Rollout;
  planCheckRuns: PlanCheckRun[];
  taskRuns: TaskRun[];
}

export interface PlanDetailPageState extends PlanDetailPageSnapshot {
  creationIssueLabels: string[];
  setCreationIssueLabels: (labels: string[]) => void;
  isEditing: boolean;
  isRunningChecks: boolean;
  setIsRunningChecks: (running: boolean) => void;
  activePhases: Set<PlanDetailPhase>;
  routeName?: string;
  routePhase?: string;
  routeStageId?: string;
  selectedTaskName?: string;
  // The plan's task runs grouped by task name, newest first, with group
  // identities preserved across poll ticks that didn't touch the task.
  taskRunsByTaskName: Map<string, TaskRun[]>;
  pendingLeaveConfirm: boolean;
  patchState: (patch: Partial<PlanDetailPageSnapshot>) => void;
  refreshState: () => Promise<void>;
  bypassLeaveGuardOnce: () => void;
  setEditing: (scope: string, editing: boolean) => void;
  togglePhase: (phase: PlanDetailPhase) => void;
  expandPhase: (phase: PlanDetailPhase) => void;
  resolveLeaveConfirm: (confirmed: boolean) => void;
}
