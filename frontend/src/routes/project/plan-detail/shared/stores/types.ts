import type { StateCreator } from "zustand";
import type { Issue, IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import type {
  Plan,
  Plan_Spec,
  PlanCheckRun,
} from "@/types/proto-es/v1/plan_service_pb";
import type { Rollout, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import type { Placement } from "../../components/threads/placement/place";

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
  setActivePhases: (phases: Iterable<PlanDetailPhase>) => void;
  togglePhase: (phase: PlanDetailPhase) => void;
  expandPhase: (phase: PlanDetailPhase) => void;
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
  pollTimerId: number | undefined;
  setRefreshing: (v: boolean) => void;
  setRunningChecks: (v: boolean) => void;
  setPollTimerId: (id: number | undefined) => void;
};

export interface ThreadFocusRequest {
  // Root comment name of the thread to expand.
  commentName: string;
  // Spec the thread anchors to; the editor of this spec consumes the request.
  specId: string;
  nonce: number;
}

export type ThreadFocusSlice = {
  threadFocus: ThreadFocusRequest | undefined;
  requestThreadFocus: (request: Omit<ThreadFocusRequest, "nonce">) => void;
  clearThreadFocus: (nonce: number) => void;
};

// Measurements of one placement run, kept so the decision to move the rule
// server-side can be made on data.
export interface PlacementMetrics {
  durationMs: number;
  // Saved/current pairs the run needed, including ones served from cache.
  pairCount: number;
  // Pairs actually sent to the worker.
  computedPairCount: number;
  // Sheets downloaded complete, and their bytes as sent to the worker.
  sheetCount: number;
  bytes: number;
  // Diff work spent by the worker.
  work: number;
}

export type PlacementSlice = {
  // Placement by comment name, for every anchored comment the last run
  // settled. A comment absent here is unplaced until the next run.
  placements: ReadonlyMap<string, Placement>;
  // The sheet hash each spec pointed at when the latest run started, so a
  // consumer showing a different sheet ignores `placements` instead of
  // drawing another revision's ranges on it.
  placementTargets: ReadonlyMap<string, string>;
  placementMetrics: PlacementMetrics | undefined;
  computePlacements: (input: {
    comments: readonly IssueComment[];
    // "projects/{project}", the issue's project.
    projectName: string;
    // The plan's current specs; their sheets are the diff targets.
    specs: readonly Plan_Spec[];
  }) => Promise<void>;
};

export type PlanDetailStore = SnapshotSlice &
  PhaseSlice &
  EditingSlice &
  SelectionSlice &
  PollingSlice &
  ThreadFocusSlice &
  PlacementSlice;

export type PlanDetailSliceCreator<Slice> = StateCreator<
  PlanDetailStore,
  [],
  [],
  Slice
>;
