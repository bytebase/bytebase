import { type Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan, PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  type Rollout,
  type TaskRun,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import type { PlanDetailPhase } from "../../shared/stores/types";
import type { PlanDetailLayoutMode } from "./useLayoutMode";

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
  isEditing: boolean;
  isRefreshing: boolean;
  isRunningChecks: boolean;
  setIsRunningChecks: (running: boolean) => void;
  activePhases: Set<PlanDetailPhase>;
  routeName?: string;
  routePhase?: string;
  routeStageId?: string;
  routeTaskId?: string;
  selectedTaskName?: string;
  pendingLeaveConfirm: boolean;
  layoutMode: PlanDetailLayoutMode;
  containerWidth: number;
  patchState: (patch: Partial<PlanDetailPageSnapshot>) => void;
  refreshState: () => Promise<void>;
  bypassLeaveGuardOnce: () => void;
  setEditing: (scope: string, editing: boolean) => void;
  togglePhase: (phase: PlanDetailPhase) => void;
  expandPhase: (phase: PlanDetailPhase) => void;
  closeTaskPanel: () => void;
  resolveLeaveConfirm: (confirmed: boolean) => void;
}
