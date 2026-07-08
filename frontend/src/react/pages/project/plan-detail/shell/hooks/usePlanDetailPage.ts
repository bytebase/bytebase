import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import {
  preserveTaskRunIdentities,
  sameMessage,
  sameMessageList,
} from "@/react/lib/protoIdentity";
import {
  PLAN_DETAIL_PHASE_CHANGES,
  PLAN_DETAIL_PHASE_DEPLOY,
  PLAN_DETAIL_PHASE_REVIEW,
} from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import { State } from "@/types/proto-es/v1/common_pb";
import { IssueSchema, IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import {
  PlanCheckRunSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import {
  RolloutSchema,
  Task_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import { unknownPlan } from "@/types/v1/issue/plan";
import { unknownProject } from "@/types/v1/project";
import { unknownUser } from "@/types/v1/user";
import { setDocumentTitle } from "@/utils";
import { isTaskActivelyTransitioning } from "@/utils/v1/issue/rollout";
import type { PlanDetailPhase } from "../../shared/stores/types";
import { usePlanDetailStoreApi } from "../../shared/stores/usePlanDetailStore";
import { fetchPlanSnapshot, fetchRolloutState } from "./fetchPlanSnapshot";
import type { PlanDetailPageSnapshot, PlanDetailPageState } from "./types";
import { useDerivedPlanState } from "./useDerivedPlanState";
import { useEditingScopes } from "./useEditingScopes";
import { useInitialFetch } from "./useInitialFetch";
import { useLeaveGuard } from "./useLeaveGuard";
import { usePhaseState } from "./usePhaseState";
import { usePolling } from "./usePolling";
import { useRedirects } from "./useRedirects";
import { useRouteSelection } from "./useRouteSelection";

const buildDefaultSnapshot = (
  projectId: string,
  planId: string
): PlanDetailPageSnapshot => ({
  projectId,
  planId,
  pageKey: `${projectId}/${planId}`,
  projectTitle: "",
  projectRequireIssueApproval: true,
  projectRequirePlanCheckNoError: true,
  projectCanCreateRollout: false,
  currentUser: unknownUser(),
  project: unknownProject(),
  isCreating: planId.toLowerCase() === "create",
  isInitializing: true,
  ready: false,
  readonly: true,
  plan: unknownPlan(),
  issue: undefined,
  rollout: undefined,
  planCheckRuns: [],
  taskRuns: [],
});

const applyDerivedState = (
  snapshot: PlanDetailPageSnapshot
): PlanDetailPageSnapshot => {
  const readonly =
    snapshot.plan.state === State.DELETED ||
    (snapshot.issue ? snapshot.issue.status !== IssueStatus.OPEN : false);
  return {
    ...snapshot,
    readonly,
    ready: !snapshot.isInitializing && !!snapshot.plan.name,
  };
};

// Structural sharing across snapshot updates: every poll tick rebuilds all
// objects from the wire, but a slice whose content is unchanged keeps its
// previous reference here, and a fully-unchanged snapshot returns `prev`
// itself so the caller can skip the state update entirely. This is what makes
// a quiet poll tick render nothing — object identity is the contract every
// downstream memo, effect dep, and prop comparison keys off.
const preserveSnapshotIdentities = (
  prev: PlanDetailPageSnapshot,
  next: PlanDetailPageSnapshot
): PlanDetailPageSnapshot => {
  const out = { ...next };
  if (sameMessage(UserSchema, prev.currentUser, next.currentUser)) {
    out.currentUser = prev.currentUser;
  }
  if (sameMessage(ProjectSchema, prev.project, next.project)) {
    out.project = prev.project;
  }
  if (sameMessage(PlanSchema, prev.plan, next.plan)) {
    out.plan = prev.plan;
  }
  if (sameMessage(IssueSchema, prev.issue, next.issue)) {
    out.issue = prev.issue;
  }
  // The rollout store already deep-preserves per-stage/per-task identity and
  // hands patchState the merged instance, so here a plain reference/structural
  // guard is enough — keeping the snapshot's rollout identical to the store's.
  if (sameMessage(RolloutSchema, prev.rollout, next.rollout)) {
    out.rollout = prev.rollout;
  }
  if (
    sameMessageList(PlanCheckRunSchema, prev.planCheckRuns, next.planCheckRuns)
  ) {
    out.planCheckRuns = prev.planCheckRuns;
  }
  out.taskRuns = preserveTaskRunIdentities(prev.taskRuns, next.taskRuns);
  const changed = (Object.keys(out) as (keyof PlanDetailPageSnapshot)[]).some(
    (key) => out[key] !== prev[key]
  );
  return changed ? out : prev;
};

const getDefaultActivePhases = (phase: PlanDetailPhase): PlanDetailPhase[] => {
  if (phase === PLAN_DETAIL_PHASE_REVIEW) {
    return [PLAN_DETAIL_PHASE_CHANGES, PLAN_DETAIL_PHASE_REVIEW];
  }
  return [phase];
};

type PhaseSelection = {
  routePhase?: PlanDetailPhase;
  routeStageId?: string;
  routeTaskId?: string;
};

// The phase to focus by default: an explicit URL selection (phase / stage /
// task) wins, otherwise the furthest-progressed phase the plan has reached.
const getCurrentPhase = (
  snapshot: PlanDetailPageSnapshot,
  selection: PhaseSelection
): PlanDetailPhase => {
  const { routePhase, routeStageId, routeTaskId } = selection;
  if (
    routePhase === PLAN_DETAIL_PHASE_CHANGES ||
    routePhase === PLAN_DETAIL_PHASE_REVIEW ||
    routePhase === PLAN_DETAIL_PHASE_DEPLOY
  ) {
    return routePhase;
  }
  if (routeStageId || routeTaskId) {
    return PLAN_DETAIL_PHASE_DEPLOY;
  }
  if (snapshot.rollout) {
    return PLAN_DETAIL_PHASE_DEPLOY;
  }
  if (snapshot.issue) {
    return PLAN_DETAIL_PHASE_REVIEW;
  }
  return PLAN_DETAIL_PHASE_CHANGES;
};

export const usePlanDetailPage = ({
  projectId,
  planId,
  routeName,
  routeQuery = {},
  specId,
}: {
  projectId: string;
  planId: string;
  routeName?: string;
  routeQuery?: Record<string, unknown>;
  specId?: string;
}): PlanDetailPageState => {
  const { t } = useTranslation();
  const [snapshot, setSnapshot] = useState<PlanDetailPageSnapshot>(() =>
    buildDefaultSnapshot(projectId, planId)
  );
  const phase = usePhaseState();
  const editing = useEditingScopes();
  const storeApi = usePlanDetailStoreApi();
  const [isRunningChecks, setIsRunningChecks] = useState(false);
  const latestSnapshotRef = useRef(snapshot);
  const { resolveLeaveConfirm } = useLeaveGuard();
  const isEditing = editing.isEditing;

  // routeQuery is a fresh object on every router navigation. Stash the latest
  // value in a ref and depend only on the individual keys we actually consume,
  // so the init/refresh/poll effects don't churn (and reset isInitializing)
  // every time an unrelated query param like ?taskId changes.
  const routeQueryRef = useRef(routeQuery);
  routeQueryRef.current = routeQuery;
  const route = useRouteSelection({ routeQuery, specId });
  const routePhase = route.phase;
  const routeStageId = route.stageId;
  const routeTaskId = route.taskId;
  const setActivePhases = phase.setActivePhases;
  const pageIdentityKey = `${projectId}/${planId}`;
  const phaseSelectionRef = useRef<PhaseSelection>({});
  phaseSelectionRef.current = {
    routePhase,
    routeStageId,
    routeTaskId,
  };
  const syncedDefaultPhaseRef = useRef<
    | {
        pageIdentityKey: string;
        phase: PlanDetailPhase;
      }
    | undefined
  >(undefined);
  const syncDefaultActivePhases = useCallback(
    (nextSnapshot: PlanDetailPageSnapshot) => {
      const nextPhase = getCurrentPhase(
        nextSnapshot,
        phaseSelectionRef.current
      );
      const synced = syncedDefaultPhaseRef.current;
      if (
        synced?.pageIdentityKey === pageIdentityKey &&
        synced.phase === nextPhase
      ) {
        return;
      }
      setActivePhases(getDefaultActivePhases(nextPhase));
      syncedDefaultPhaseRef.current = {
        pageIdentityKey,
        phase: nextPhase,
      };
    },
    [pageIdentityKey, setActivePhases]
  );

  const patchState = useCallback(
    (patch: Partial<PlanDetailPageSnapshot>) => {
      const prevSnapshot = latestSnapshotRef.current;
      // Seed the shared rollout store from this (already staleness-guarded)
      // patch, and adopt the store's identity-preserved instance so the deploy
      // view and the log viewer read the exact same rollout object.
      if (patch.rollout) {
        patch = {
          ...patch,
          rollout: useAppStore.getState().upsertRollout(patch.rollout),
        };
      }
      const nextSnapshot = preserveSnapshotIdentities(
        prevSnapshot,
        applyDerivedState({
          ...prevSnapshot,
          ...patch,
        })
      );
      syncDefaultActivePhases(nextSnapshot);
      if (nextSnapshot === prevSnapshot) {
        // Content-identical (the common quiet poll tick) — no state update,
        // so nothing under the provider re-renders.
        return;
      }
      latestSnapshotRef.current = nextSnapshot;
      setSnapshot(nextSnapshot);
    },
    [syncDefaultActivePhases]
  );

  // Monotonic fetch sequence: a fetch that was already in flight when a newer
  // one started (e.g. a poll tick overlapping a user-action refresh) carries
  // older data and must not overwrite the fresher snapshot when it resolves
  // late — after a task rerun that would flip the status back to the old one.
  const fetchSeqRef = useRef(0);
  const fetchFullState = useCallback(async () => {
    const current = latestSnapshotRef.current;
    fetchSeqRef.current += 1;
    const seq = fetchSeqRef.current;
    const patch = await fetchPlanSnapshot(
      current.projectId,
      current.planId,
      routeQueryRef.current
    );
    if (seq !== fetchSeqRef.current) {
      return;
    }
    patchState(patch);
  }, [patchState]);

  // The slim "status lane": while a task is transitioning, poll only the rollout
  // and its task runs instead of the full 7-RPC page snapshot. Cheaper per tick,
  // so it can run at the fast floor without adding load, and the status
  // transition surfaces in ~0.5s instead of after the full fetch. Shares
  // fetchSeqRef with the full fetch, so whichever started last wins and a late
  // slim tick can never rewind a fresher full result.
  const fetchStatusState = useCallback(async () => {
    const rolloutName = latestSnapshotRef.current.rollout?.name;
    // Only reached on active ticks, which imply a rollout exists; bail rather
    // than re-decide slim-vs-full (pollTick already gated that) if it doesn't.
    if (!rolloutName) {
      return;
    }
    fetchSeqRef.current += 1;
    const seq = fetchSeqRef.current;
    const patch = await fetchRolloutState(rolloutName);
    if (seq !== fetchSeqRef.current) {
      return;
    }
    patchState(patch);
  }, [patchState]);

  useInitialFetch({
    projectId,
    planId,
    routeQueryRef,
    storeApi,
    patchState,
  });

  useEffect(() => {
    if (snapshot.isCreating) {
      setDocumentTitle(t("plan.new-plan"), snapshot.projectTitle);
      return;
    }
    if (snapshot.ready && snapshot.plan.title) {
      setDocumentTitle(snapshot.plan.title, snapshot.projectTitle);
    }
  }, [
    snapshot.isCreating,
    snapshot.plan.title,
    snapshot.projectTitle,
    snapshot.ready,
    t,
  ]);

  useRedirects({
    ready: snapshot.ready,
    plan: snapshot.plan,
    issue: snapshot.issue,
  });

  useLayoutEffect(() => {
    syncDefaultActivePhases(latestSnapshotRef.current);
  }, [
    pageIdentityKey,
    routePhase,
    routeStageId,
    routeTaskId,
    syncDefaultActivePhases,
  ]);

  const { isPlanDone, hasActiveTasks } = useMemo(() => {
    if (!snapshot.rollout) {
      return { isPlanDone: false, hasActiveTasks: false };
    }
    const allTasks = snapshot.rollout.stages.flatMap((stage) => stage.tasks);
    const nowMs = Date.now();
    return {
      isPlanDone:
        allTasks.length > 0 &&
        allTasks.every(
          (task) =>
            task.status === Task_Status.DONE ||
            task.status === Task_Status.SKIPPED
        ),
      // Fast-poll only tasks actually transitioning (RUNNING, or PENDING and
      // due) so PENDING -> RUNNING -> DONE is observed promptly — but a task
      // scheduled for a future maintenance window stays on the backed-off poll
      // instead of hammering the rollout RPCs while it waits.
      hasActiveTasks: allTasks.some((task) =>
        isTaskActivelyTransitioning(task, nowMs)
      ),
    };
  }, [snapshot.rollout]);

  // The poller passes the live `fast` flag into each tick, so a tick uses the
  // slim status fetch exactly while work is in flight and the full fetch once the
  // plan settles (refreshing the rest of the page that the slim lane skips).
  const pollTick = useCallback(
    (fast: boolean) => (fast ? fetchStatusState() : fetchFullState()),
    [fetchStatusState, fetchFullState]
  );

  const { restart: restartPolling } = usePolling({
    enabled: snapshot.ready && !snapshot.isCreating && !isPlanDone,
    refreshState: pollTick,
    fast: hasActiveTasks,
  });

  // Public refresh used by user actions (run/skip/cancel a task, edits, etc.).
  // Resets the poll backoff first — not after the fetch — so the follow-up
  // status transition (e.g. a task moving PENDING -> RUNNING) is watched from
  // the fast floor from the moment of the action, then does a full fetch
  // immediately (the action may have changed more than task status, e.g. created
  // the rollout). If the immediate fetch is still in flight when the restarted
  // tick fires, the fetch sequence guard keeps the newest result authoritative.
  const refreshState = useCallback(async () => {
    restartPolling();
    await fetchFullState();
  }, [fetchFullState, restartPolling]);

  return useDerivedPlanState({
    snapshot,
    isEditing,
    isRunningChecks,
    setIsRunningChecks,
    phase,
    editing,
    routeName,
    routePhase,
    routeStageId,
    routeTaskId,
    patchState,
    refreshState,
    resolveLeaveConfirm,
  });
};
