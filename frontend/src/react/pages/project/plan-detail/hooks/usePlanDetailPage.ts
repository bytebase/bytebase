import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/router";
import {
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
} from "@/router/dashboard/projectV1";
import { PLAN_DETAIL_PHASE_DEPLOY } from "@/router/dashboard/projectV1RouteHelpers";
import { State } from "@/types/proto-es/v1/common_pb";
import { type Issue, IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan, PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  type Rollout,
  Task_Status,
  type TaskRun,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { unknownPlan } from "@/types/v1/issue/plan";
import { unknownProject } from "@/types/v1/project";
import { unknownUser } from "@/types/v1/user";
import { getIssueRoute, setDocumentTitle } from "@/utils";
import { usePlanDetailStoreApi } from "../shared/stores/usePlanDetailStore";
import { fetchPlanSnapshot } from "../shell/hooks/fetchPlanSnapshot";
import { useEditingScopes } from "../shell/hooks/useEditingScopes";
import { useInitialFetch } from "../shell/hooks/useInitialFetch";
import { usePhaseState } from "../shell/hooks/usePhaseState";
import { usePolling } from "../shell/hooks/usePolling";
import { useRouteSelection } from "../shell/hooks/useRouteSelection";
import { useSidebarMode } from "../shell/hooks/useSidebarMode";

export {
  MOBILE_BREAKPOINT_PX,
  POLLER_INTERVAL,
  PROJECT_NAME_PREFIX,
  SIDEBAR_WIDTH_NARROW_PX,
  SIDEBAR_WIDTH_WIDE_PX,
  WIDE_SIDEBAR_BREAKPOINT_PX,
} from "../shell/constants";
export type { PlanDetailSidebarMode } from "../shell/hooks/useSidebarMode";

import type { PlanDetailSidebarMode } from "../shell/hooks/useSidebarMode";

export type PlanDetailPhase = "changes" | "review" | "deploy";

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
  lastRefreshTime: number;
  activePhases: Set<PlanDetailPhase>;
  routeName?: string;
  routePhase?: string;
  routeStageId?: string;
  routeTaskId?: string;
  selectedTaskName?: string;
  pendingLeaveConfirm: boolean;
  sidebarMode: PlanDetailSidebarMode;
  containerWidth: number;
  desktopSidebarWidth: number;
  mobileSidebarOpen: boolean;
  patchState: (patch: Partial<PlanDetailPageSnapshot>) => void;
  refreshState: () => Promise<void>;
  bypassLeaveGuardOnce: () => void;
  setEditing: (scope: string, editing: boolean) => void;
  setMobileSidebarOpen: (open: boolean) => void;
  togglePhase: (phase: PlanDetailPhase) => void;
  expandPhase: (phase: PlanDetailPhase) => void;
  closeTaskPanel: () => void;
  resolveLeaveConfirm: (confirmed: boolean) => void;
}

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

const shouldRedirectToIssueDetail = (plan: Plan, issue?: Issue) => {
  if (!issue?.name) {
    return false;
  }
  if (plan.specs.length === 0) {
    return false;
  }
  return plan.specs.every((spec) => {
    return (
      spec.config?.case === "createDatabaseConfig" ||
      spec.config?.case === "exportDataConfig"
    );
  });
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
  // Target route captured when the guard intercepts a navigation. We cancel
  // that navigation synchronously and re-issue it after the user confirms.
  const pendingLeaveTargetRef = useRef<string | null>(null);
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

  const resolveLeaveConfirm = useCallback(
    (confirmed: boolean) => {
      const target = confirmed ? pendingLeaveTargetRef.current : null;
      pendingLeaveTargetRef.current = null;
      editing.setPendingLeaveConfirm(false);
      if (target) {
        editing.bypassLeaveGuardOnce();
        // Replace (not push) so a confirmed-discard navigation doesn't leave
        // an extra entry that lets Back return to the discarded plan. Works
        // correctly whether the original navigation was push, replace, or
        // browser back/forward.
        void router.replace(target);
      }
    },
    [editing]
  );

  useEffect(() => {
    if (!isEditing) {
      // Editing scope ended (e.g. async save completed) while a leave
      // prompt is open — there's nothing unsaved anymore, so navigate to
      // the captured target without further confirmation.
      if (pendingLeaveTargetRef.current) {
        resolveLeaveConfirm(true);
      }
      return;
    }

    const onBeforeUnload = (event: BeforeUnloadEvent) => {
      event.returnValue = t("common.leave-without-saving");
      event.preventDefault();
    };
    window.addEventListener("beforeunload", onBeforeUnload);
    const removeGuard = router.beforeEach((to, _from, next) => {
      if (storeApi.getState().isLeaveGuardBypassed()) {
        next();
        return;
      }
      // Cancel the navigation synchronously and remember the target so we
      // can re-issue it from resolveLeaveConfirm after the user confirms.
      // Always overwrite the pending target — the latest navigation wins.
      pendingLeaveTargetRef.current = to.fullPath;
      editing.setPendingLeaveConfirm(true);
      next(false);
    });

    return () => {
      window.removeEventListener("beforeunload", onBeforeUnload);
      removeGuard();
    };
  }, [editing, isEditing, resolveLeaveConfirm, storeApi, t]);

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

  useEffect(() => {
    if (
      snapshot.ready &&
      shouldRedirectToIssueDetail(snapshot.plan, snapshot.issue)
    ) {
      void router.replace(getIssueRoute({ name: snapshot.issue?.name ?? "" }));
    }
  }, [snapshot.issue, snapshot.plan, snapshot.ready]);

  useEffect(() => {
    if (
      routeName === PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS ||
      routeName === PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL
    ) {
      phase.expandPhase("changes");
    }
    if (
      routePhase === PLAN_DETAIL_PHASE_DEPLOY ||
      routeStageId ||
      routeTaskId
    ) {
      phase.expandPhase("deploy");
    }
  }, [phase, routeName, routePhase, routeStageId, routeTaskId]);

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

  const selectedTaskName = useMemo(() => {
    if (!routeTaskId || !snapshot.rollout) {
      return undefined;
    }
    for (const stage of snapshot.rollout.stages) {
      const task = stage.tasks.find((item) =>
        item.name.endsWith(`/${routeTaskId}`)
      );
      if (task) {
        return task.name;
      }
    }
    return undefined;
  }, [routeTaskId, snapshot.rollout]);

  return useMemo(
    () => ({
      ...snapshot,
      isEditing,
      isRefreshing,
      isRunningChecks,
      setIsRunningChecks,
      lastRefreshTime,
      activePhases: phase.activePhases,
      routeName,
      routePhase,
      routeStageId,
      routeTaskId,
      selectedTaskName,
      pendingLeaveConfirm: editing.pendingLeaveConfirm,
      sidebarMode: sidebar.sidebarMode,
      containerWidth: sidebar.containerWidth,
      desktopSidebarWidth: sidebar.sidebarWidth,
      mobileSidebarOpen: sidebar.isMobileSidebarOpen,
      bypassLeaveGuardOnce: editing.bypassLeaveGuardOnce,
      patchState,
      refreshState,
      setEditing: editing.setEditing,
      setMobileSidebarOpen: sidebar.setMobileSidebarOpen,
      togglePhase: phase.togglePhase,
      expandPhase: phase.expandPhase,
      closeTaskPanel,
      resolveLeaveConfirm,
    }),
    [
      closeTaskPanel,
      editing,
      isEditing,
      isRefreshing,
      isRunningChecks,
      lastRefreshTime,
      patchState,
      phase,
      refreshState,
      resolveLeaveConfirm,
      routeName,
      routePhase,
      routeStageId,
      routeTaskId,
      selectedTaskName,
      sidebar,
      snapshot,
    ]
  );
};
