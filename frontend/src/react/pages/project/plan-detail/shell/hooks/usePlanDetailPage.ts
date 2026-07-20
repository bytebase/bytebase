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
import type { PlanDetailPhase } from "../../shared/stores/types";
import { usePlanDetailStoreApi } from "../../shared/stores/usePlanDetailStore";
import { getPlanCheckSummary } from "../../utils/phaseSummary";
import { fetchPlanSnapshot } from "./fetchPlanSnapshot";
import type { PlanDetailPageSnapshot, PlanDetailPageState } from "./types";
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
  const [creationIssueLabels, setCreationIssueLabels] = useState<string[]>([]);
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
  useEffect(() => {
    setCreationIssueLabels([]);
  }, [pageIdentityKey]);
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
    // The page isn't remounted on plan->plan navigation, so a poll started for
    // the previous plan can still be in flight after the switch. The seq guard
    // doesn't bump on navigation, so also drop the patch when the page identity
    // has changed — otherwise the old plan's data would merge onto the new
    // plan's snapshot (wrong data under a correct URL, until the next poll).
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
      const hasCanceledPlanChecks =
        (snapshot.plan.planCheckRunStatusCount?.CANCELED ?? 0) > 0;
      // The backend considers an approval template with zero roles approved,
      // although its API approval status can still be PENDING. Treat the
      // resolved empty flow as passed so we keep polling through rollout
      // creation instead of falling back to the idle cadence in that window.
      const hasNoApprovalSteps =
        snapshot.issue?.approvalTemplate?.flow?.roles.length === 0;
      const approvalPassed =
        snapshot.issue?.approvalStatus === ApprovalStatus.APPROVED ||
        snapshot.issue?.approvalStatus === ApprovalStatus.SKIPPED ||
        hasNoApprovalSteps;
      const approvalChecking =
        snapshot.issue?.status === IssueStatus.OPEN &&
        !snapshot.issue.draft &&
        snapshot.issue.approvalStatus === ApprovalStatus.CHECKING;
      const rolloutExpected =
        snapshot.plan.hasRollout ||
        snapshot.issue?.status === IssueStatus.DONE ||
        (snapshot.issue?.status === IssueStatus.OPEN &&
          !snapshot.issue.draft &&
          approvalPassed &&
          !hasActivePlanChecks &&
          !hasCanceledPlanChecks &&
          checks.error === 0);
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
    enabled: snapshot.ready && !snapshot.isCreating && !isPlanDone,
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
