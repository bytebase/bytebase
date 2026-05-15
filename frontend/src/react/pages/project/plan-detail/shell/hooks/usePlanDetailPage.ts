import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/router";
import {
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
} from "@/router/dashboard/projectV1";
import {
  PLAN_DETAIL_PHASE_CHANGES,
  PLAN_DETAIL_PHASE_DEPLOY,
  PLAN_DETAIL_PHASE_REVIEW,
} from "@/router/dashboard/projectV1RouteHelpers";
import { State } from "@/types/proto-es/v1/common_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
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
import { useSidebarMode } from "./useSidebarMode";

const buildDefaultSnapshot = (
  projectId: string,
  planId: string,
  specId?: string
): PlanDetailPageSnapshot => ({
  projectId,
  planId,
  specId,
  pageKey: `${projectId}/${planId}/${specId ?? ""}`,
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

export const usePlanDetailPage = ({
  projectId,
  planId,
  routeName,
  routeQuery = {},
  specId,
  pageHost,
}: {
  projectId: string;
  planId: string;
  routeName?: string;
  routeQuery?: Record<string, unknown>;
  specId?: string;
  pageHost: HTMLDivElement | null;
}): PlanDetailPageState => {
  const { t } = useTranslation();
  const [snapshot, setSnapshot] = useState<PlanDetailPageSnapshot>(() =>
    buildDefaultSnapshot(projectId, planId, specId)
  );
  const phase = usePhaseState();
  const editing = useEditingScopes();
  const storeApi = usePlanDetailStoreApi();
  const sidebar = useSidebarMode(pageHost);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [isRunningChecks, setIsRunningChecks] = useState(false);
  const [lastRefreshTime, setLastRefreshTime] = useState(0);
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
  const focusPhase = phase.focusPhase;
  const routePageKey = `${projectId}/${planId}/${specId ?? ""}`;
  const currentPhase = useMemo<PlanDetailPhase>(() => {
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
    if (
      routeName === PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS ||
      routeName === PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL
    ) {
      return PLAN_DETAIL_PHASE_CHANGES;
    }
    if (snapshot.rollout) {
      return PLAN_DETAIL_PHASE_DEPLOY;
    }
    if (snapshot.issue) {
      return PLAN_DETAIL_PHASE_REVIEW;
    }
    return PLAN_DETAIL_PHASE_CHANGES;
  }, [
    routeName,
    routePhase,
    routeStageId,
    routeTaskId,
    snapshot.issue,
    snapshot.rollout,
  ]);

  useEffect(() => {
    latestSnapshotRef.current = snapshot;
  }, [snapshot]);

  const patchState = useCallback((patch: Partial<PlanDetailPageSnapshot>) => {
    setSnapshot((prev) => applyDerivedState({ ...prev, ...patch }));
  }, []);

  const refreshState = useCallback(async () => {
    try {
      setIsRefreshing(true);
      const current = latestSnapshotRef.current;
      const patch = await fetchPlanSnapshot(
        current.projectId,
        current.planId,
        routeQueryRef.current
      );
      patchState(patch);
      setLastRefreshTime(Date.now());
    } finally {
      setIsRefreshing(false);
    }
  }, [patchState]);

  const closeTaskPanel = useCallback(() => {
    void router.replace({
      query: {
        ...(routePhase ? { phase: routePhase } : {}),
        ...(routeStageId ? { stageId: routeStageId } : {}),
      },
    });
  }, [routePhase, routeStageId]);

  useInitialFetch({
    projectId,
    planId,
    specId,
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

  useEffect(() => {
    focusPhase(currentPhase);
  }, [currentPhase, focusPhase, routePageKey]);

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

  usePolling({
    enabled: snapshot.ready && !snapshot.isCreating && !isPlanDone,
    refreshState,
  });

  return useDerivedPlanState({
    snapshot,
    isEditing,
    isRefreshing,
    isRunningChecks,
    setIsRunningChecks,
    lastRefreshTime,
    phase,
    editing,
    sidebar,
    routeName,
    routePhase,
    routeStageId,
    routeTaskId,
    patchState,
    refreshState,
    closeTaskPanel,
    resolveLeaveConfirm,
  });
};
