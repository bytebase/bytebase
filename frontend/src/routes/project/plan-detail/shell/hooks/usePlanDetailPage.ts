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
  isPlanDetailPhase,
  PLAN_DETAIL_PHASE_CHANGES,
  PLAN_DETAIL_PHASE_DEPLOY,
  PLAN_DETAIL_PHASE_REVIEW,
} from "@/app/router/handles";
import {
  preserveTaskRunIdentities,
  sameMessage,
  sameMessageList,
} from "@/lib/protoIdentity";
import { useAppStore } from "@/stores/app";
import { ApprovalStatus, State } from "@/types/proto-es/v1/common_pb";
import { IssueSchema, IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import {
  PlanCheckRunSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import {
  RolloutSchema,
  Task_Status,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import { unknownPlan } from "@/types/v1/issue/plan";
import { unknownProject } from "@/types/v1/project";
import { unknownUser } from "@/types/v1/user";
import { setDocumentTitle } from "@/utils";
import { isTaskActivelyTransitioning } from "@/utils/v1/issue/rollout";
import { invalidateProjectPagedDataCacheIfChanged } from "../../../pagedDataCacheScope";
import type { PlanDetailPhase } from "../../shared/stores/types";
import { usePlanDetailStoreApi } from "../../shared/stores/usePlanDetailStore";
import {
  getPlanCheckSummary,
  isRolloutExpected,
} from "../../utils/phaseSummary";
import { fetchPlanSnapshot } from "./fetchPlanSnapshot";
import type { PlanDetailPageSnapshot, PlanDetailPageState } from "./types";
import { useCanonicalPlanDetailRoute } from "./useCanonicalPlanDetailRoute";
import { useDerivedPlanState } from "./useDerivedPlanState";
import { useEditingScopes } from "./useEditingScopes";
import { useInitialFetch } from "./useInitialFetch";
import { useLeaveGuard } from "./useLeaveGuard";
import { usePhaseState } from "./usePhaseState";
import { type PlanPollingMode, usePlanPolling } from "./usePlanPolling";
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
  routeSelectionKey: string;
};

// The phase to focus by default: an explicit URL selection wins — a stage/task
// selection already resolves to an explicit deploy phase in useRouteSelection —
// otherwise the furthest-progressed phase the plan has reached.
const getCurrentPhase = (
  snapshot: PlanDetailPageSnapshot,
  selection: PhaseSelection
): PlanDetailPhase => {
  const { routePhase } = selection;
  if (isPlanDetailPhase(routePhase)) {
    return routePhase;
  }
  if (snapshot.rollout || isRolloutExpected(snapshot)) {
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
  routeHash,
  routeName,
  routeQuery = {},
  specId,
  stageId,
  taskId,
}: {
  projectId: string;
  planId: string;
  routeHash?: string;
  routeName?: string;
  routeQuery?: Record<string, unknown>;
  specId?: string;
  stageId?: string;
  taskId?: string;
}): PlanDetailPageState => {
  const { t } = useTranslation();
  const [snapshot, setSnapshot] = useState<PlanDetailPageSnapshot>(() =>
    buildDefaultSnapshot(projectId, planId)
  );
  const phase = usePhaseState();
  const editing = useEditingScopes();
  const storeApi = usePlanDetailStoreApi();
  const [creationIssueLabels, setCreationIssueLabels] = useState<string[]>([]);
  const [isRunningChecks, setIsRunningChecks] = useState(false);
  const latestSnapshotRef = useRef(snapshot);
  const { resolveLeaveConfirm } = useLeaveGuard();
  const isEditing = editing.isEditing;

  // routeQuery is a fresh object on every router navigation. Stash the latest
  // value in a ref and depend only on the individual keys we actually consume,
  // so the init/refresh/poll effects don't churn (and reset isInitializing)
  // every time unrelated secondary query state changes.
  const routeQueryRef = useRef(routeQuery);
  routeQueryRef.current = routeQuery;
  const route = useRouteSelection({
    routeName,
    routeQuery,
    specId,
    stageId,
    taskId,
  });
  const routePhase = route.phase;
  const routeSelectionKey = route.selectionKey;
  const routeStageId = route.stageId;
  const routeTaskId = route.taskId;
  const setActivePhases = phase.setActivePhases;
  const expandPhase = phase.expandPhase;
  const pageIdentityKey = `${projectId}/${planId}`;
  const snapshotBelongsToRoute = snapshot.pageKey === pageIdentityKey;
  useEffect(() => {
    setCreationIssueLabels([]);
  }, [pageIdentityKey]);
  const phaseSelectionRef = useRef<PhaseSelection>({ routeSelectionKey });
  phaseSelectionRef.current = {
    routePhase,
    routeSelectionKey,
  };
  const syncedDefaultPhaseRef = useRef<
    | {
        pageIdentityKey: string;
        selectionKey: string;
      }
    | undefined
  >(undefined);
  const syncDefaultActivePhases = useCallback(
    (nextSnapshot: PlanDetailPageSnapshot) => {
      if (!nextSnapshot.ready || nextSnapshot.pageKey !== pageIdentityKey) {
        return;
      }
      const nextPhase = getCurrentPhase(
        nextSnapshot,
        phaseSelectionRef.current
      );
      const explicitPhase = phaseSelectionRef.current.routePhase;
      const selectionKey = phaseSelectionRef.current.routeSelectionKey;
      const synced = syncedDefaultPhaseRef.current;
      if (synced?.pageIdentityKey !== pageIdentityKey) {
        setActivePhases(getDefaultActivePhases(nextPhase));
        syncedDefaultPhaseRef.current = { pageIdentityKey, selectionKey };
        return;
      }
      if (synced.selectionKey === selectionKey) return;
      // Same-plan resource navigation only reveals the destination phase. It
      // never collapses phases the user already opened, so Back/Forward and
      // rapid spec/stage/task selection cannot reset disclosure state.
      if (explicitPhase) {
        expandPhase(explicitPhase);
      }
      syncedDefaultPhaseRef.current = { pageIdentityKey, selectionKey };
    },
    [expandPhase, pageIdentityKey, setActivePhases]
  );

  const patchState = useCallback(
    (patch: Partial<PlanDetailPageSnapshot>) => {
      const prevSnapshot = latestSnapshotRef.current;
      invalidateProjectPagedDataCacheIfChanged(
        prevSnapshot.projectId,
        "plans",
        prevSnapshot.plan,
        patch.plan
      );
      invalidateProjectPagedDataCacheIfChanged(
        prevSnapshot.projectId,
        "issues",
        prevSnapshot.issue,
        patch.issue
      );
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
      // Reveal a rollout that materializes while this plan is already open,
      // regardless of which backend gate completed last. Expanding before the
      // snapshot update keeps the old future state visually unchanged, so the
      // first render with rollout data is already expanded without changing
      // selection, URL, scroll, or the user's other disclosure choices.
      if (
        prevSnapshot.ready &&
        prevSnapshot.pageKey === nextSnapshot.pageKey &&
        !prevSnapshot.rollout &&
        nextSnapshot.rollout
      ) {
        expandPhase(PLAN_DETAIL_PHASE_DEPLOY);
      }
      if (nextSnapshot === prevSnapshot) {
        // Content-identical (the common quiet poll tick) — no state update,
        // so nothing under the provider re-renders.
        return;
      }
      latestSnapshotRef.current = nextSnapshot;
      setSnapshot(nextSnapshot);
    },
    [expandPhase, syncDefaultActivePhases]
  );

  // Monotonic fetch sequence: a fetch already in flight when a newer one started
  // (a poll tick overlapping a user-action refresh) carries older data and must
  // not overwrite the fresher snapshot when it resolves late.
  const fetchSeqRef = useRef(0);
  const fetchState = useCallback(async () => {
    const current = latestSnapshotRef.current;
    fetchSeqRef.current += 1;
    const seq = fetchSeqRef.current;
    const patch = await fetchPlanSnapshot(
      current.projectId,
      current.planId,
      routeQueryRef.current,
      // Background poll / post-action refresh: stay silent so a transient
      // failure doesn't spam the global toast (the initial load is loud).
      true
    );
    // The keyed page provider remounts for normal plan-to-plan navigation. Keep
    // the page-identity check as a defensive hook-level guard for any caller
    // that reuses the hook instance while a request is still in flight.
    if (
      seq !== fetchSeqRef.current ||
      latestSnapshotRef.current.pageKey !== current.pageKey
    ) {
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
    if (!snapshotBelongsToRoute) {
      return;
    }
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
    snapshotBelongsToRoute,
    t,
  ]);

  useRedirects({
    ready: snapshotBelongsToRoute && snapshot.ready,
    plan: snapshot.plan,
    issue: snapshot.issue,
  });

  // Reveal a same-plan route destination before the browser paints. A normal
  // effect would commit the previous collapsed state for one frame first.
  useLayoutEffect(() => {
    syncDefaultActivePhases(latestSnapshotRef.current);
  }, [pageIdentityKey, routeSelectionKey, syncDefaultActivePhases]);

  useCanonicalPlanDetailRoute({
    projectId,
    planId,
    routeHash,
    routeName,
    routeQuery,
    specId,
    stageId,
    taskId,
    snapshot,
    isEditing,
    bypassLeaveGuardOnce: editing.bypassLeaveGuardOnce,
  });

  const { activePollingKey, isPlanDone, pollingMode } = useMemo((): {
    activePollingKey: string;
    isPlanDone: boolean;
    pollingMode: PlanPollingMode;
  } => {
    if (!snapshot.rollout) {
      const checks = getPlanCheckSummary(snapshot.plan);
      // A newly queued check is AVAILABLE until the scheduler claims it. The
      // public check-run enum maps that store status to UNSPECIFIED, but the
      // plan status-count map preserves the key. Keep polling actively across
      // that short queueing window as well as while the check is RUNNING.
      const hasActivePlanChecks =
        checks.running > 0 ||
        (snapshot.plan.planCheckRunStatusCount?.AVAILABLE ?? 0) > 0;
      const approvalChecking =
        snapshot.issue?.status === IssueStatus.OPEN &&
        !snapshot.issue.draft &&
        snapshot.issue.approvalStatus === ApprovalStatus.CHECKING;
      const rolloutExpected = isRolloutExpected(snapshot);
      // Once every gate passes, rollout creation is asynchronous. Keep polling
      // at the active cadence through that gap, as well as when either the plan
      // or issue already proves the rollout committed but its data is missing.
      return {
        activePollingKey: [
          snapshot.plan.hasRollout,
          snapshot.issue?.status,
          snapshot.issue?.approvalStatus,
          JSON.stringify(snapshot.plan.planCheckRunStatusCount),
        ].join(":"),
        isPlanDone: false,
        pollingMode:
          hasActivePlanChecks || approvalChecking || rolloutExpected
            ? "active"
            : "idle",
      };
    }
    const nowMs = Date.now();
    let taskCount = 0;
    let allSettled = true;
    let shouldPollActively = false;
    for (const stage of snapshot.rollout.stages) {
      for (const task of stage.tasks) {
        taskCount++;
        if (
          task.status !== Task_Status.DONE &&
          task.status !== Task_Status.SKIPPED
        ) {
          allSettled = false;
        }
        // Poll fast for a task actually transitioning (RUNNING, or a PENDING
        // task due to run now); a task scheduled for a future maintenance window
        // stays on the idle cadence rather than polling every second while it
        // waits — the idle poll still catches it once the backend starts it.
        if (isTaskActivelyTransitioning(task, nowMs)) {
          shouldPollActively = true;
        }
      }
    }
    const hasActiveTaskRuns = snapshot.taskRuns.some(
      (taskRun) =>
        taskRun.status === TaskRun_Status.PENDING ||
        taskRun.status === TaskRun_Status.AVAILABLE ||
        taskRun.status === TaskRun_Status.RUNNING
    );
    const activePollingKey = [
      ...snapshot.rollout.stages.flatMap((stage) =>
        stage.tasks.map((task) => `${task.name}:${task.status}`)
      ),
      ...snapshot.taskRuns.map(
        (taskRun) => `${taskRun.name}:${taskRun.status}`
      ),
    ].join("|");
    return {
      activePollingKey,
      isPlanDone: taskCount > 0 && allSettled && !hasActiveTaskRuns,
      pollingMode: shouldPollActively || hasActiveTaskRuns ? "active" : "idle",
    };
  }, [snapshot.issue, snapshot.plan, snapshot.rollout, snapshot.taskRuns]);

  // Poll the complete snapshot. Idle polling backs off from 1s to 16s. Active
  // plan checks, approval finding, rollout creation, and task execution share a
  // faster 500ms -> 1s -> 2s -> 4s cadence which resets when state changes.
  // Independent jitter prevents synchronized clients; hidden tabs pause until
  // they become visible again.
  const { restart: restartPolling } = usePlanPolling({
    enabled:
      snapshotBelongsToRoute &&
      snapshot.ready &&
      !snapshot.isCreating &&
      !isPlanDone,
    mode: pollingMode,
    refreshState: fetchState,
    resetKey: pollingMode === "active" ? activePollingKey : undefined,
  });

  // Public refresh used by user actions (run/skip/cancel a task, edits, etc.).
  const refreshState = useCallback(async () => {
    await fetchState();
    restartPolling();
  }, [fetchState, restartPolling]);

  return useDerivedPlanState({
    snapshot,
    creationIssueLabels,
    setCreationIssueLabels,
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
