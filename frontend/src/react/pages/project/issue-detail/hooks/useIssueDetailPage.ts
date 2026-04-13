import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/connect";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import {
  WORKSPACE_ROUTE_403,
  WORKSPACE_ROUTE_404,
} from "@/router/dashboard/workspaceRoutes";
import { projectNamePrefix, useProjectV1Store } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  GetIssueRequestSchema,
  type Issue,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  GetPlanCheckRunRequestSchema,
  GetPlanRequestSchema,
  type Plan,
  type PlanCheckRun,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  GetRolloutRequestSchema,
  ListTaskRunsRequestSchema,
  type Rollout,
  Task_Status,
  type TaskRun,
} from "@/types/proto-es/v1/rollout_service_pb";
import { unknownPlan } from "@/types/v1/issue/plan";
import { getRolloutFromPlan, minmax, setDocumentTitle } from "@/utils";
import type { ProjectIssueDetailPageProps } from "../types";
import {
  type IssueDetailType,
  resolveIssueDetailType,
} from "./useIssueDetailType";

export type IssueDetailSidebarMode = "NONE" | "DESKTOP" | "MOBILE";

export interface IssueDetailPageSnapshot {
  projectId: string;
  issueId: string;
  pageKey: string;
  projectTitle: string;
  isCreating: boolean;
  isInitializing: boolean;
  ready: boolean;
  readonly: boolean;
  issue?: Issue;
  plan?: Plan;
  rollout?: Rollout;
  taskRuns: TaskRun[];
  planCheckRuns: PlanCheckRun[];
  issueType?: IssueDetailType;
  sidebarMode: IssueDetailSidebarMode;
  desktopSidebarWidth: number;
  mobileSidebarOpen: boolean;
}

export interface IssueDetailPageState extends IssueDetailPageSnapshot {
  isEditing: boolean;
  patchState: (patch: Partial<IssueDetailPageSnapshot>) => void;
  refreshState: () => Promise<void>;
  setEditing: (scope: string, editing: boolean) => void;
  setMobileSidebarOpen: (open: boolean) => void;
}

const POLLER_INTERVAL = {
  min: 1000,
  max: 30000,
  growth: 2,
  jitter: 250,
} as const;

const buildDefaultSnapshot = (
  projectId: string,
  issueId: string
): IssueDetailPageSnapshot => ({
  projectId,
  issueId,
  pageKey: `${projectId}/${issueId}`,
  projectTitle: "",
  isCreating: issueId.toLowerCase() === "create",
  isInitializing: true,
  ready: false,
  readonly: true,
  issue: undefined,
  plan: undefined,
  rollout: undefined,
  taskRuns: [],
  planCheckRuns: [],
  issueType: undefined,
  sidebarMode: "NONE",
  desktopSidebarWidth: 0,
  mobileSidebarOpen: false,
});

const applyDerivedState = (
  snapshot: IssueDetailPageSnapshot
): IssueDetailPageSnapshot => {
  const readonly =
    snapshot.plan?.state === State.DELETED ||
    (snapshot.issue ? snapshot.issue.status !== IssueStatus.OPEN : false);
  return {
    ...snapshot,
    issueType: resolveIssueDetailType(
      snapshot.issue,
      snapshot.plan ?? unknownPlan()
    ),
    readonly,
    ready:
      (Boolean(snapshot.issue) || Boolean(snapshot.plan)) &&
      !snapshot.isInitializing,
  };
};

const loadIssueDetailSnapshot = async (
  projectId: string,
  issueId: string
): Promise<Partial<IssueDetailPageSnapshot>> => {
  const issue = await issueServiceClientConnect.getIssue(
    create(GetIssueRequestSchema, {
      name: `${projectNamePrefix}${projectId}/issues/${issueId}`,
    })
  );

  if (!issue.plan) {
    return {
      issue,
      plan: unknownPlan(),
      planCheckRuns: [],
      rollout: undefined,
      taskRuns: [],
    };
  }

  const plan = await planServiceClientConnect.getPlan(
    create(GetPlanRequestSchema, {
      name: issue.plan,
    })
  );

  const [planCheckRuns, rollout] = await Promise.all([
    planServiceClientConnect
      .getPlanCheckRun(
        create(GetPlanCheckRunRequestSchema, {
          name: `${plan.name}/planCheckRun`,
        })
      )
      .then((run) => [run] as PlanCheckRun[])
      .catch(() => []),
    plan.hasRollout
      ? rolloutServiceClientConnect
          .getRollout(
            create(GetRolloutRequestSchema, {
              name: getRolloutFromPlan(plan.name),
            })
          )
          .catch(() => undefined)
      : Promise.resolve(undefined),
  ]);

  const taskRuns =
    rollout !== undefined
      ? await rolloutServiceClientConnect
          .listTaskRuns(
            create(ListTaskRunsRequestSchema, {
              parent: `${rollout.name}/stages/-/tasks/-`,
            })
          )
          .then((response) => response.taskRuns)
          .catch(() => [])
      : [];

  return {
    issue,
    plan,
    planCheckRuns,
    rollout,
    taskRuns,
  };
};

const refreshIssueDetailSnapshot = async (
  snapshot: Pick<IssueDetailPageSnapshot, "issue" | "plan">
): Promise<Partial<IssueDetailPageSnapshot>> => {
  const issue = snapshot.issue?.name
    ? await issueServiceClientConnect
        .getIssue(
          create(GetIssueRequestSchema, {
            name: snapshot.issue.name,
          })
        )
        .catch(() => undefined)
    : undefined;

  const planName = snapshot.plan?.name || issue?.plan;
  if (!planName) {
    return {
      issue,
      plan: unknownPlan(),
      planCheckRuns: [],
      rollout: undefined,
      taskRuns: [],
    };
  }

  const plan = await planServiceClientConnect.getPlan(
    create(GetPlanRequestSchema, {
      name: planName,
    })
  );

  const [planCheckRuns, rollout] = await Promise.all([
    planServiceClientConnect
      .getPlanCheckRun(
        create(GetPlanCheckRunRequestSchema, {
          name: `${plan.name}/planCheckRun`,
        })
      )
      .then((run) => [run] as PlanCheckRun[])
      .catch(() => []),
    plan.hasRollout
      ? rolloutServiceClientConnect
          .getRollout(
            create(GetRolloutRequestSchema, {
              name: getRolloutFromPlan(plan.name),
            })
          )
          .catch(() => undefined)
      : Promise.resolve(undefined),
  ]);

  const taskRuns =
    rollout !== undefined
      ? await rolloutServiceClientConnect
          .listTaskRuns(
            create(ListTaskRunsRequestSchema, {
              parent: `${rollout.name}/stages/-/tasks/-`,
            })
          )
          .then((response) => response.taskRuns)
          .catch(() => [])
      : [];

  return {
    issue,
    plan,
    planCheckRuns,
    rollout,
    taskRuns,
  };
};

export const useIssueDetailPage = ({
  issueId,
  pageHost,
  projectId,
}: ProjectIssueDetailPageProps & {
  pageHost: HTMLDivElement | null;
}): IssueDetailPageState => {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();
  const project = useVueState(() =>
    projectStore.getProjectByName(`${projectNamePrefix}${projectId}`)
  );
  const [snapshot, setSnapshot] = useState<IssueDetailPageSnapshot>(() =>
    buildDefaultSnapshot(projectId, issueId)
  );
  const [editingScopes, setEditingScopes] = useState<Record<string, true>>({});
  const latestSnapshotRef = useRef(snapshot);
  const pollTimerRef = useRef<number | undefined>(undefined);
  const isEditing = Object.keys(editingScopes).length > 0;

  useEffect(() => {
    latestSnapshotRef.current = snapshot;
  }, [snapshot]);

  const stopPolling = useCallback(() => {
    if (!pollTimerRef.current) {
      return;
    }
    window.clearTimeout(pollTimerRef.current);
    pollTimerRef.current = undefined;
  }, []);

  const patchState = useCallback(
    (patch: Partial<IssueDetailPageSnapshot>) => {
      setSnapshot((prev) => applyDerivedState({ ...prev, ...patch }));
    },
    [setSnapshot]
  );

  const setMobileSidebarOpen = useCallback(
    (open: boolean) => {
      patchState({ mobileSidebarOpen: open });
    },
    [patchState]
  );

  const refreshState = useCallback(async () => {
    const current = latestSnapshotRef.current;
    const patch = await refreshIssueDetailSnapshot(current);
    patchState(patch);
  }, [patchState]);

  const setEditing = useCallback((scope: string, editing: boolean) => {
    setEditingScopes((prev) => {
      if (editing) {
        if (prev[scope]) {
          return prev;
        }
        return {
          ...prev,
          [scope]: true,
        };
      }
      if (!prev[scope]) {
        return prev;
      }
      const next = { ...prev };
      delete next[scope];
      return next;
    });
  }, []);

  useEffect(() => {
    setEditingScopes({});
    patchState({
      issueId,
      isCreating: issueId.toLowerCase() === "create",
      isInitializing: true,
      issue: undefined,
      pageKey: `${projectId}/${issueId}`,
      plan: undefined,
      planCheckRuns: [],
      projectId,
      rollout: undefined,
      taskRuns: [],
    });

    let canceled = false;

    const load = async () => {
      try {
        const patch = await loadIssueDetailSnapshot(projectId, issueId);
        if (canceled) {
          return;
        }
        patchState({
          ...patch,
          isInitializing: false,
        });
      } catch (error) {
        if (canceled) {
          return;
        }
        if (error instanceof ConnectError) {
          if (error.code === Code.NotFound) {
            void router.push({ name: WORKSPACE_ROUTE_404 });
          } else if (error.code === Code.PermissionDenied) {
            void router.push({ name: WORKSPACE_ROUTE_403 });
          }
          patchState({ isInitializing: false });
          return;
        }

        patchState({ isInitializing: false });
        throw error;
      }
    };

    void load();

    return () => {
      canceled = true;
    };
  }, [issueId, patchState, projectId]);

  useEffect(() => {
    patchState({
      projectTitle: project.title,
    });
  }, [patchState, project.title]);

  useEffect(() => {
    const entityTitle = snapshot.issue?.title || snapshot.plan?.title;
    if (snapshot.ready && entityTitle) {
      setDocumentTitle(entityTitle, snapshot.projectTitle);
    }
  }, [
    snapshot.issue?.title,
    snapshot.plan?.title,
    snapshot.projectTitle,
    snapshot.ready,
  ]);

  useEffect(() => {
    if (!pageHost) {
      patchState({
        desktopSidebarWidth: 0,
        mobileSidebarOpen: false,
        sidebarMode: "NONE",
      });
      return;
    }

    const updateSidebar = () => {
      const containerWidth = pageHost.getBoundingClientRect().width;
      const sidebarMode: IssueDetailSidebarMode =
        containerWidth === 0
          ? "NONE"
          : containerWidth < 780
            ? "MOBILE"
            : "DESKTOP";
      const desktopSidebarWidth = containerWidth < 1280 ? 240 : 336;
      patchState({
        desktopSidebarWidth,
        mobileSidebarOpen:
          sidebarMode === "MOBILE"
            ? latestSnapshotRef.current.mobileSidebarOpen
            : false,
        sidebarMode,
      });
    };

    updateSidebar();
    const observer = new ResizeObserver(() => updateSidebar());
    observer.observe(pageHost);

    return () => {
      observer.disconnect();
    };
  }, [pageHost, patchState]);

  useEffect(() => {
    if (!isEditing) {
      return;
    }

    const onBeforeUnload = (event: BeforeUnloadEvent) => {
      event.returnValue = t("common.leave-without-saving");
      event.preventDefault();
    };
    window.addEventListener("beforeunload", onBeforeUnload);
    const removeGuard = router.beforeEach((_to, _from, next) => {
      if (window.confirm(t("common.leave-without-saving"))) {
        next();
      } else {
        next(false);
      }
    });

    return () => {
      window.removeEventListener("beforeunload", onBeforeUnload);
      removeGuard();
    };
  }, [isEditing, t]);

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

  useEffect(() => {
    if (!snapshot.ready || snapshot.isCreating || isPlanDone) {
      stopPolling();
      return;
    }

    let canceled = false;

    const poll = (interval: number) => {
      stopPolling();
      const nextInterval = minmax(
        interval +
          Math.floor(Math.random() * (POLLER_INTERVAL.jitter * 2 + 1)) -
          POLLER_INTERVAL.jitter,
        POLLER_INTERVAL.min,
        POLLER_INTERVAL.max
      );

      pollTimerRef.current = window.setTimeout(async () => {
        if (canceled) {
          return;
        }
        await refreshState().catch(() => undefined);
        if (canceled) {
          return;
        }
        poll(
          Math.min(nextInterval * POLLER_INTERVAL.growth, POLLER_INTERVAL.max)
        );
      }, nextInterval);
    };

    poll(POLLER_INTERVAL.min);

    return () => {
      canceled = true;
      stopPolling();
    };
  }, [
    isPlanDone,
    refreshState,
    snapshot.isCreating,
    snapshot.ready,
    stopPolling,
  ]);

  return useMemo(
    () => ({
      ...snapshot,
      isEditing,
      patchState,
      refreshState,
      setEditing,
      setMobileSidebarOpen,
    }),
    [
      isEditing,
      patchState,
      refreshState,
      setEditing,
      setMobileSidebarOpen,
      snapshot,
    ]
  );
};
