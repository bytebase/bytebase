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
  preserveRolloutIdentities,
  preserveTaskRunIdentities,
  sameMessage,
  sameMessageList,
} from "@/react/lib/protoIdentity";
import {
  PLAN_DETAIL_PHASE_CHANGES,
  PLAN_DETAIL_PHASE_DEPLOY,
  PLAN_DETAIL_PHASE_REVIEW,
} from "@/react/router/handles";
import { State } from "@/types/proto-es/v1/common_pb";
import { IssueSchema, IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import {
  PlanCheckRunSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import { unknownPlan } from "@/types/v1/issue/plan";
import { unknownProject } from "@/types/v1/project";
import { unknownUser } from "@/types/v1/user";
import { setDocumentTitle } from "@/utils";
import type { PlanDetailPhase } from "../../shared/stores/types";
import { usePlanDetailStoreApi } from "../../shared/stores/usePlanDetailStore";
import { fetchPlanSnapshot } from "./fetchPlanSnapshot";
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
  out.rollout = preserveRolloutIdentities(prev.rollout, next.rollout);
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
  const fetchState = useCallback(async () => {
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

  const isPlanDone = useMemo(() => {
    if (!snapshot.rollout) {
      return false;
    }
    const allTasks = snapshot.rollout.stages.flatMap((stage) => stage.tasks);
    return (
      allTasks.length > 0 &&
      allTasks.every(
        (task) =>
          task.status === Task_Status.DONE ||
          task.status === Task_Status.SKIPPED
      )
    );
  }, [snapshot.rollout]);

  const { restart: restartPolling } = usePolling({
    enabled: snapshot.ready && !snapshot.isCreating && !isPlanDone,
    refreshState: fetchState,
  });

  // Public refresh used by user actions (run/skip/cancel a task, edits, etc.).
  // Resets the poll backoff first — not after the fetch — so the follow-up
  // status transition (e.g. a task moving PENDING -> RUNNING) is watched on
  // ~1s ticks from the moment of the action, then fetches immediately. If the
  // immediate fetch is still in flight when the restarted tick fires, the
  // fetch sequence guard keeps the newest result authoritative.
  const refreshState = useCallback(async () => {
    restartPolling();
    await fetchState();
  }, [fetchState, restartPolling]);

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
