import type { StateCreator } from "zustand";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan, PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import type { Rollout, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";

export type PlanDetailPhase = "changes" | "review" | "deploy";

export interface PlanDetailPageSnapshot {
  plan?: Plan;
  issue?: Issue;
  rollout?: Rollout;
  taskRuns: TaskRun[];
  planCheckRuns: PlanCheckRun[];
  isInitializing: boolean;
  isNotFound: boolean;
  isPermissionDenied: boolean;
}

export type SnapshotSlice = {
  snapshot: PlanDetailPageSnapshot;
  setSnapshot: (snapshot: PlanDetailPageSnapshot) => void;
  patchSnapshot: (patch: Partial<PlanDetailPageSnapshot>) => void;
};

export type PhaseSlice = {
  activePhases: Set<PlanDetailPhase>;
  togglePhase: (phase: PlanDetailPhase) => void;
  expandPhase: (phase: PlanDetailPhase) => void;
  collapsePhase: (phase: PlanDetailPhase) => void;
};

export type EditingSlice = {
  editingScopes: Record<string, true>;
  setEditing: (scope: string, editing: boolean) => void;
  bypassLeaveGuardOnce: () => void;
  isLeaveGuardBypassed: () => boolean;
  pendingLeaveTarget: string | null;
  setPendingLeaveTarget: (target: string | null) => void;
  pendingLeaveConfirm: boolean;
  setPendingLeaveConfirm: (open: boolean) => void;
};

export type SelectionSlice = {
  routePhase: PlanDetailPhase | undefined;
  selectedSpecId: string | undefined;
  selectedStageId: string | undefined;
  selectedTaskName: string | undefined;
  setRouteSelection: (selection: {
    phase?: PlanDetailPhase;
    specId?: string;
    stageId?: string;
    taskName?: string;
  }) => void;
};

export type PollingSlice = {
  isRefreshing: boolean;
  isRunningChecks: boolean;
  lastRefreshTime: number;
  pollTimerId: number | undefined;
  setRefreshing: (v: boolean) => void;
  setRunningChecks: (v: boolean) => void;
  setLastRefreshTime: (t: number) => void;
  setPollTimerId: (id: number | undefined) => void;
};

export type PlanDetailStore = SnapshotSlice &
  PhaseSlice &
  EditingSlice &
  SelectionSlice &
  PollingSlice;

export type PlanDetailSliceCreator<Slice> = StateCreator<
  PlanDetailStore,
  [],
  [],
  Slice
>;
