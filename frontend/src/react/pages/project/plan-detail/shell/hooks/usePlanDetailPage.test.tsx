import { create } from "@bufbuild/protobuf";
import { renderHook, waitFor } from "@testing-library/react";
import { act, type ReactNode } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ApprovalStatus, State } from "@/types/proto-es/v1/common_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { PlanSchema } from "@/types/proto-es/v1/plan_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import {
  RolloutSchema,
  Task_Status,
  TaskRunSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import type { PlanDetailPageSnapshot } from "./types";

const mocks = vi.hoisted(() => ({
  fetchPlanSnapshot: vi.fn(),
  getIssueRoute: vi.fn(() => ({ name: "issue-detail" })),
  routerPush: vi.fn(),
  routerReplace: vi.fn(),
  setDocumentTitle: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/react/router")>()),
  router: {
    push: mocks.routerPush,
    replace: mocks.routerReplace,
  },
}));

vi.mock("@/utils", () => ({
  extractTaskNameFromTaskRunName: (taskRunName: string) =>
    taskRunName.split("/taskRuns/")[0],
  getIssueRoute: mocks.getIssueRoute,
  isDev: () => true,
  minmax: (value: number, min: number, max: number) =>
    Math.max(min, Math.min(max, value)),
  setDocumentTitle: mocks.setDocumentTitle,
}));

vi.mock("./fetchPlanSnapshot", () => ({
  fetchPlanSnapshot: mocks.fetchPlanSnapshot,
}));

import { PlanDetailStoreProvider } from "../../shared/stores/PlanDetailStoreProvider";
import { usePlanDetailPage } from "./usePlanDetailPage";

const wrapper = ({ children }: { children: ReactNode }) => (
  <PlanDetailStoreProvider>{children}</PlanDetailStoreProvider>
);

const buildSnapshotPatch = ({
  issue,
  planId = "create",
  rollout,
}: {
  issue?: PlanDetailPageSnapshot["issue"];
  planId?: string;
  rollout?: PlanDetailPageSnapshot["rollout"];
} = {}): Partial<PlanDetailPageSnapshot> =>
  ({
    projectId: "foo",
    planId,
    pageKey: `foo/${planId}`,
    projectTitle: "Foo",
    projectRequireIssueApproval: false,
    projectRequirePlanCheckNoError: false,
    projectCanCreateRollout: true,
    currentUser: { name: "users/me@example.com" },
    project: { name: "projects/foo", title: "Foo" },
    isCreating: planId.toLowerCase() === "create",
    readonly: false,
    plan: {
      name: `projects/foo/plans/${planId}`,
      title: "",
      creator: "users/me@example.com",
      hasRollout: false,
      state: State.ACTIVE,
      specs: [{ id: "spec-1" }, { id: "spec-2" }],
    },
    issue,
    rollout,
    planCheckRuns: [],
    taskRuns: [],
  }) as unknown as Partial<PlanDetailPageSnapshot>;

describe("usePlanDetailPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.fetchPlanSnapshot.mockImplementation(
      async (_projectId: string, planId: string) =>
        buildSnapshotPatch({ planId })
    );
  });

  test("defaults to only the changes phase on the specs route", async () => {
    const { result } = renderHook(
      () =>
        usePlanDetailPage({
          projectId: "foo",
          planId: "create",
          routeName: "workspace.project.plan.detail.specs",
        }),
      { wrapper }
    );

    await waitFor(() => expect(result.current.ready).toBe(true));
    expect(result.current.activePhases.has("changes")).toBe(true);
    expect(result.current.activePhases.has("review")).toBe(false);
    expect(result.current.activePhases.has("deploy")).toBe(false);

    act(() => result.current.togglePhase("changes"));
    await waitFor(() =>
      expect(result.current.activePhases.has("changes")).toBe(false)
    );
  });

  test("defaults to only the deploy phase on the deploy route query", async () => {
    const { result } = renderHook(
      () =>
        usePlanDetailPage({
          projectId: "foo",
          planId: "plan-1",
          routeQuery: { phase: "deploy" },
        }),
      { wrapper }
    );

    await waitFor(() => expect(result.current.ready).toBe(true));
    expect(result.current.activePhases.has("changes")).toBe(false);
    expect(result.current.activePhases.has("review")).toBe(false);
    expect(result.current.activePhases.has("deploy")).toBe(true);

    act(() => result.current.togglePhase("deploy"));
    await waitFor(() =>
      expect(result.current.activePhases.has("deploy")).toBe(false)
    );
  });

  test("resets focused phase when navigating to another plan with the same route phase", async () => {
    const { result, rerender } = renderHook(
      ({ planId }: { planId: string }) =>
        usePlanDetailPage({
          projectId: "foo",
          planId,
          routeQuery: { phase: "deploy" },
        }),
      {
        initialProps: { planId: "plan-1" },
        wrapper,
      }
    );

    await waitFor(() => expect(result.current.ready).toBe(true));
    expect(result.current.activePhases.has("deploy")).toBe(true);

    act(() => result.current.togglePhase("changes"));
    await waitFor(() =>
      expect(result.current.activePhases.has("changes")).toBe(true)
    );

    rerender({ planId: "plan-2" });
    await waitFor(() => expect(result.current.planId).toBe("plan-2"));
    expect(result.current.activePhases.has("changes")).toBe(false);
    expect(result.current.activePhases.has("review")).toBe(false);
    expect(result.current.activePhases.has("deploy")).toBe(true);
  });

  test("defaults to the changes and review phases after an issue is created", async () => {
    mocks.fetchPlanSnapshot.mockResolvedValue(
      buildSnapshotPatch({
        issue: {
          name: "projects/foo/issues/1",
        } as PlanDetailPageSnapshot["issue"],
        planId: "plan-1",
      })
    );

    const { result } = renderHook(
      () =>
        usePlanDetailPage({
          projectId: "foo",
          planId: "plan-1",
        }),
      { wrapper }
    );

    await waitFor(() => expect(result.current.ready).toBe(true));
    expect(result.current.activePhases.has("changes")).toBe(true);
    expect(result.current.activePhases.has("review")).toBe(true);
    expect(result.current.activePhases.has("deploy")).toBe(false);
  });

  test("keeps the review section visible for reviewed plans on the specs route", async () => {
    mocks.fetchPlanSnapshot.mockResolvedValue(
      buildSnapshotPatch({
        issue: {
          name: "projects/foo/issues/1",
        } as PlanDetailPageSnapshot["issue"],
        planId: "plan-1",
      })
    );

    const { result } = renderHook(
      () =>
        usePlanDetailPage({
          projectId: "foo",
          planId: "plan-1",
          routeName: "workspace.project.plan.detail.specs",
        }),
      { wrapper }
    );

    await waitFor(() => expect(result.current.ready).toBe(true));
    expect(result.current.activePhases.has("changes")).toBe(true);
    expect(result.current.activePhases.has("review")).toBe(true);
    expect(result.current.activePhases.has("deploy")).toBe(false);
  });

  test("focuses the deploy phase on the specs route for rollout plans", async () => {
    mocks.fetchPlanSnapshot.mockResolvedValue(
      buildSnapshotPatch({
        planId: "plan-1",
        rollout: {
          name: "projects/foo/rollouts/1",
          stages: [],
        } as unknown as PlanDetailPageSnapshot["rollout"],
      })
    );

    const { result } = renderHook(
      () =>
        usePlanDetailPage({
          projectId: "foo",
          planId: "plan-1",
          routeName: "workspace.project.plan.detail.specs",
        }),
      { wrapper }
    );

    await waitFor(() => expect(result.current.ready).toBe(true));
    expect(result.current.activePhases.has("changes")).toBe(false);
    expect(result.current.activePhases.has("review")).toBe(false);
    expect(result.current.activePhases.has("deploy")).toBe(true);
  });

  test("focuses the deploy phase on the spec-detail route for reviewed rollout plans", async () => {
    mocks.fetchPlanSnapshot.mockResolvedValue(
      buildSnapshotPatch({
        issue: {
          name: "projects/foo/issues/1",
        } as PlanDetailPageSnapshot["issue"],
        planId: "plan-1",
        rollout: {
          name: "projects/foo/rollouts/1",
          stages: [],
        } as unknown as PlanDetailPageSnapshot["rollout"],
      })
    );

    const { result } = renderHook(
      () =>
        usePlanDetailPage({
          projectId: "foo",
          planId: "plan-1",
          routeName: "workspace.project.plan.detail.spec.detail",
          specId: "spec-1",
        }),
      { wrapper }
    );

    await waitFor(() => expect(result.current.ready).toBe(true));
    expect(result.current.activePhases.has("changes")).toBe(false);
    expect(result.current.activePhases.has("review")).toBe(false);
    expect(result.current.activePhases.has("deploy")).toBe(true);
  });

  test("has review default phases on the first ready render", async () => {
    mocks.fetchPlanSnapshot.mockResolvedValue(
      buildSnapshotPatch({
        issue: {
          name: "projects/foo/issues/1",
        } as PlanDetailPageSnapshot["issue"],
        planId: "plan-1",
      })
    );
    const readyRenders: Set<string>[] = [];

    const { result } = renderHook(
      () => {
        const page = usePlanDetailPage({
          projectId: "foo",
          planId: "plan-1",
        });
        if (page.ready) {
          readyRenders.push(new Set(page.activePhases));
        }
        return page;
      },
      { wrapper }
    );

    await waitFor(() => expect(result.current.ready).toBe(true));
    expect(readyRenders[0].has("changes")).toBe(true);
    expect(readyRenders[0].has("review")).toBe(true);
    expect(readyRenders[0].has("deploy")).toBe(false);
  });

  test("has deploy as the only phase on the first ready render for rollout plans", async () => {
    mocks.fetchPlanSnapshot.mockResolvedValue(
      buildSnapshotPatch({
        planId: "plan-1",
        rollout: {
          name: "projects/foo/rollouts/1",
          stages: [],
        } as unknown as PlanDetailPageSnapshot["rollout"],
      })
    );
    const readyRenders: Set<string>[] = [];

    const { result } = renderHook(
      () => {
        const page = usePlanDetailPage({
          projectId: "foo",
          planId: "plan-1",
        });
        if (page.ready) {
          readyRenders.push(new Set(page.activePhases));
        }
        return page;
      },
      { wrapper }
    );

    await waitFor(() => expect(result.current.ready).toBe(true));
    expect(readyRenders[0].has("changes")).toBe(false);
    expect(readyRenders[0].has("review")).toBe(false);
    expect(readyRenders[0].has("deploy")).toBe(true);
  });

  test("does not refetch the page snapshot when only the route spec changes", async () => {
    const { result, rerender } = renderHook(
      ({ specId }: { specId: string }) =>
        usePlanDetailPage({
          projectId: "foo",
          planId: "plan-1",
          routeName: "workspace.project.plan.detail.spec.detail",
          specId,
        }),
      {
        initialProps: { specId: "spec-1" },
        wrapper,
      }
    );

    await waitFor(() => expect(result.current.ready).toBe(true));
    expect(result.current.pageKey).toBe("foo/plan-1");
    expect(result.current.activePhases.has("changes")).toBe(true);
    expect(mocks.fetchPlanSnapshot).toHaveBeenCalledTimes(1);

    rerender({ specId: "spec-2" });

    await waitFor(() =>
      expect(result.current.activePhases.has("changes")).toBe(true)
    );
    expect(result.current.pageKey).toBe("foo/plan-1");
    expect(mocks.fetchPlanSnapshot).toHaveBeenCalledTimes(1);
  });

  test("refetches the page snapshot when the plan changes", async () => {
    const { result, rerender } = renderHook(
      ({ planId }: { planId: string }) =>
        usePlanDetailPage({
          projectId: "foo",
          planId,
          routeName: "workspace.project.plan.detail.spec.detail",
          specId: "spec-1",
        }),
      {
        initialProps: { planId: "plan-1" },
        wrapper,
      }
    );

    await waitFor(() => expect(result.current.ready).toBe(true));
    expect(result.current.pageKey).toBe("foo/plan-1");
    expect(mocks.fetchPlanSnapshot).toHaveBeenCalledTimes(1);

    rerender({ planId: "plan-2" });

    await waitFor(() => expect(result.current.planId).toBe("plan-2"));
    expect(result.current.pageKey).toBe("foo/plan-2");
    expect(mocks.fetchPlanSnapshot).toHaveBeenCalledTimes(2);
  });

  test("preserves a manual phase toggle across a poll refresh", async () => {
    mocks.fetchPlanSnapshot.mockResolvedValue(
      buildSnapshotPatch({
        issue: {
          name: "projects/foo/issues/1",
        } as PlanDetailPageSnapshot["issue"],
        planId: "plan-1",
      })
    );

    const { result } = renderHook(
      () =>
        usePlanDetailPage({
          projectId: "foo",
          planId: "plan-1",
        }),
      { wrapper }
    );

    await waitFor(() => expect(result.current.ready).toBe(true));
    expect(result.current.activePhases.has("review")).toBe(true);

    // User manually collapses the review section.
    act(() => result.current.togglePhase("review"));
    await waitFor(() =>
      expect(result.current.activePhases.has("review")).toBe(false)
    );

    // A background poll re-fetches the same snapshot; the dedup guard must not
    // re-expand the section the user just collapsed.
    await act(async () => {
      await result.current.refreshState();
    });

    expect(result.current.activePhases.has("review")).toBe(false);
    expect(result.current.activePhases.has("changes")).toBe(true);
  });

  test("a poll returning identical content keeps the page state identity", async () => {
    // Real proto messages (not cast plain objects): the identity gate only
    // structurally compares actual messages.
    const buildProtoPatch = (): Partial<PlanDetailPageSnapshot> => ({
      projectId: "foo",
      planId: "plan-1",
      pageKey: "foo/plan-1",
      projectTitle: "Foo",
      projectRequireIssueApproval: false,
      projectRequirePlanCheckNoError: false,
      projectCanCreateRollout: true,
      currentUser: create(UserSchema, { name: "users/me@example.com" }),
      project: create(ProjectSchema, { name: "projects/foo", title: "Foo" }),
      isCreating: false,
      readonly: false,
      plan: create(PlanSchema, {
        name: "projects/foo/plans/plan-1",
        state: State.ACTIVE,
        specs: [{ id: "spec-1" }],
      }),
      issue: undefined,
      rollout: undefined,
      planCheckRuns: [],
      taskRuns: [],
    });
    mocks.fetchPlanSnapshot.mockImplementation(async () => buildProtoPatch());

    const { result } = renderHook(
      () =>
        usePlanDetailPage({
          projectId: "foo",
          planId: "plan-1",
        }),
      { wrapper }
    );
    await waitFor(() => expect(result.current.ready).toBe(true));

    // A quiet poll: fresh objects from the wire, identical content. The
    // snapshot (and thus the whole page object) must keep its identity so
    // nothing under the provider re-renders.
    const before = result.current;
    await act(async () => {
      await result.current.refreshState();
    });
    expect(mocks.fetchPlanSnapshot.mock.calls.length).toBeGreaterThanOrEqual(2);
    expect(result.current).toBe(before);

    // A real change (plan title) produces a new snapshot, but slices that
    // didn't change keep their references.
    mocks.fetchPlanSnapshot.mockImplementation(async () => ({
      ...buildProtoPatch(),
      plan: create(PlanSchema, {
        name: "projects/foo/plans/plan-1",
        title: "renamed",
        state: State.ACTIVE,
        specs: [{ id: "spec-1" }],
      }),
    }));
    await act(async () => {
      await result.current.refreshState();
    });
    expect(result.current).not.toBe(before);
    expect(result.current.plan.title).toBe("renamed");
    expect(result.current.currentUser).toBe(before.currentUser);
    expect(result.current.project).toBe(before.project);
  });

  test("a poll changing one task preserves sibling stage/task/run identities", async () => {
    const stage1 = "projects/foo/rollouts/1/stages/s1";
    const stage2 = "projects/foo/rollouts/1/stages/s2";
    const buildPatch = (
      t2Status: Task_Status,
      runDetail: string
    ): Partial<PlanDetailPageSnapshot> => ({
      ...buildSnapshotPatch({ planId: "plan-1" }),
      rollout: create(RolloutSchema, {
        name: "projects/foo/rollouts/1",
        stages: [
          {
            name: stage1,
            tasks: [
              { name: `${stage1}/tasks/t1`, status: Task_Status.DONE },
              { name: `${stage1}/tasks/t2`, status: t2Status },
            ],
          },
          {
            name: stage2,
            tasks: [
              { name: `${stage2}/tasks/t3`, status: Task_Status.NOT_STARTED },
            ],
          },
        ],
      }),
      taskRuns: [
        create(TaskRunSchema, {
          name: `${stage1}/tasks/t2/taskRuns/r2`,
          detail: runDetail,
        }),
        create(TaskRunSchema, {
          name: `${stage1}/tasks/t1/taskRuns/r1`,
          detail: "finished",
        }),
      ],
    });
    mocks.fetchPlanSnapshot.mockImplementation(async () =>
      buildPatch(Task_Status.RUNNING, "executing step 1")
    );

    const { result } = renderHook(
      () =>
        usePlanDetailPage({
          projectId: "foo",
          planId: "plan-1",
        }),
      { wrapper }
    );
    await waitFor(() => expect(result.current.ready).toBe(true));
    const before = result.current;

    // A poll tick where only task t2 (and its run) changed.
    mocks.fetchPlanSnapshot.mockImplementation(async () =>
      buildPatch(Task_Status.DONE, "executing step 2")
    );
    await act(async () => {
      await result.current.refreshState();
    });
    const after = result.current;

    // The rollout shell and the changed stage/task get new references…
    expect(after.rollout).not.toBe(before.rollout);
    expect(after.rollout?.stages[0]).not.toBe(before.rollout?.stages[0]);
    expect(after.rollout?.stages[0].tasks[1]).not.toBe(
      before.rollout?.stages[0].tasks[1]
    );
    // …while every untouched sibling keeps its previous reference, so
    // memoized cards skip re-rendering.
    expect(after.rollout?.stages[1]).toBe(before.rollout?.stages[1]);
    expect(after.rollout?.stages[0].tasks[0]).toBe(
      before.rollout?.stages[0].tasks[0]
    );
    expect(after.taskRuns).not.toBe(before.taskRuns);
    expect(after.taskRuns[0]).not.toBe(before.taskRuns[0]);
    expect(after.taskRuns[1]).toBe(before.taskRuns[1]);
  });

  test("a partial poll patch keeps last-known-good rollout state", async () => {
    mocks.fetchPlanSnapshot.mockResolvedValue({
      ...buildSnapshotPatch({ planId: "plan-1" }),
      rollout: rolloutWithTask(Task_Status.RUNNING),
      taskRuns: [create(TaskRunSchema, { name: "taskRuns/1" })],
    });

    const { result } = renderHook(
      () => usePlanDetailPage({ projectId: "foo", planId: "plan-1" }),
      { wrapper }
    );
    await waitFor(() => expect(result.current.ready).toBe(true));
    const rollout = result.current.rollout;
    const taskRuns = result.current.taskRuns;

    mocks.fetchPlanSnapshot.mockResolvedValue({ projectTitle: "Refreshed" });
    await act(async () => {
      await result.current.refreshState();
    });

    expect(result.current.projectTitle).toBe("Refreshed");
    expect(result.current.rollout).toBe(rollout);
    expect(result.current.taskRuns).toBe(taskRuns);
  });

  const rolloutWithTask = (status: Task_Status) =>
    create(RolloutSchema, {
      name: "projects/foo/rollouts/1",
      stages: [
        {
          name: "projects/foo/rollouts/1/stages/s1",
          tasks: [
            {
              name: "projects/foo/rollouts/1/stages/s1/tasks/t1",
              status,
            },
          ],
        },
      ],
    });

  test("polls the full snapshot on the active cadence while a task transitions", async () => {
    vi.useFakeTimers();
    try {
      mocks.fetchPlanSnapshot.mockResolvedValue({
        ...buildSnapshotPatch({ planId: "plan-1" }),
        rollout: rolloutWithTask(Task_Status.RUNNING),
      });

      const { result } = renderHook(
        () => usePlanDetailPage({ projectId: "foo", planId: "plan-1" }),
        { wrapper }
      );
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0);
      });
      expect(result.current.ready).toBe(true);
      mocks.fetchPlanSnapshot.mockClear();

      // A RUNNING task → the poll re-fetches the full snapshot every ~1s.
      await act(async () => {
        await vi.advanceTimersByTimeAsync(1000);
      });
      expect(mocks.fetchPlanSnapshot).toHaveBeenCalledTimes(1);
      await act(async () => {
        await vi.advanceTimersByTimeAsync(1000);
      });
      expect(mocks.fetchPlanSnapshot).toHaveBeenCalledTimes(2);
    } finally {
      vi.useRealTimers();
    }
  });

  test("uses the idle cadence when no task is transitioning", async () => {
    vi.useFakeTimers();
    try {
      mocks.fetchPlanSnapshot.mockResolvedValue({
        ...buildSnapshotPatch({ planId: "plan-1" }),
        rollout: rolloutWithTask(Task_Status.NOT_STARTED),
      });

      const { result } = renderHook(
        () => usePlanDetailPage({ projectId: "foo", planId: "plan-1" }),
        { wrapper }
      );
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0);
      });
      expect(result.current.ready).toBe(true);
      mocks.fetchPlanSnapshot.mockClear();

      // No active task → the fast (1s) cadence must not fire; the idle (15s) one
      // eventually does.
      await act(async () => {
        await vi.advanceTimersByTimeAsync(2000);
      });
      expect(mocks.fetchPlanSnapshot).not.toHaveBeenCalled();
      await act(async () => {
        await vi.advanceTimersByTimeAsync(14000);
      });
      expect(mocks.fetchPlanSnapshot).toHaveBeenCalled();
    } finally {
      vi.useRealTimers();
    }
  });

  test("polls on the active cadence while a done issue waits for its rollout", async () => {
    vi.useFakeTimers();
    try {
      mocks.fetchPlanSnapshot.mockResolvedValue(
        buildSnapshotPatch({
          planId: "plan-1",
          issue: {
            name: "projects/foo/issues/1",
            status: IssueStatus.DONE,
            approvalStatus: ApprovalStatus.APPROVED,
          } as PlanDetailPageSnapshot["issue"],
        })
      );

      const { result } = renderHook(
        () => usePlanDetailPage({ projectId: "foo", planId: "plan-1" }),
        { wrapper }
      );
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0);
      });
      expect(result.current.ready).toBe(true);
      mocks.fetchPlanSnapshot.mockClear();

      await act(async () => {
        await vi.advanceTimersByTimeAsync(1000);
      });
      expect(mocks.fetchPlanSnapshot).toHaveBeenCalledTimes(1);
    } finally {
      vi.useRealTimers();
    }
  });

  test("polls on the active cadence when the plan expects a missing rollout", async () => {
    vi.useFakeTimers();
    try {
      const patch = buildSnapshotPatch({ planId: "plan-1" });
      mocks.fetchPlanSnapshot.mockResolvedValue({
        ...patch,
        plan: { ...patch.plan, hasRollout: true },
      });

      const { result } = renderHook(
        () => usePlanDetailPage({ projectId: "foo", planId: "plan-1" }),
        { wrapper }
      );
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0);
      });
      expect(result.current.ready).toBe(true);
      mocks.fetchPlanSnapshot.mockClear();

      await act(async () => {
        await vi.advanceTimersByTimeAsync(1000);
      });
      expect(mocks.fetchPlanSnapshot).toHaveBeenCalledTimes(1);
    } finally {
      vi.useRealTimers();
    }
  });

  test("a stale poll from the previous plan cannot overwrite the new plan", async () => {
    vi.useFakeTimers();
    try {
      const patchFor = (
        planId: string,
        title: string,
        status: Task_Status
      ) => ({
        ...buildSnapshotPatch({ planId }),
        projectTitle: title,
        rollout: rolloutWithTask(status),
      });
      mocks.fetchPlanSnapshot.mockImplementation(
        async (_projectId: string, planId: string) =>
          planId === "plan-a"
            ? patchFor("plan-a", "A", Task_Status.RUNNING)
            : patchFor("plan-b", "B", Task_Status.NOT_STARTED)
      );

      const { result, rerender } = renderHook(
        ({ planId }: { planId: string }) =>
          usePlanDetailPage({ projectId: "foo", planId }),
        { initialProps: { planId: "plan-a" }, wrapper }
      );
      // Initial plan-A load: its RUNNING task arms the 1s active cadence.
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0);
      });
      expect(result.current.projectTitle).toBe("A");

      // Hold the next fetch — the first plan-A poll tick — in flight.
      let releaseStalePoll: () => void = () => {};
      mocks.fetchPlanSnapshot.mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            releaseStalePoll = () =>
              resolve(patchFor("plan-a", "A-late", Task_Status.RUNNING));
          })
      );
      await act(async () => {
        await vi.advanceTimersByTimeAsync(1000);
      });

      // Navigate to plan B; the page is not remounted, so the plan-A poll stays
      // in flight. B's initial load resolves and wins.
      rerender({ planId: "plan-b" });
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0);
      });
      expect(result.current.projectTitle).toBe("B");

      // The held plan-A poll resolves late — the page-identity guard must drop
      // it so it can't merge plan A's data onto plan B's snapshot.
      await act(async () => {
        releaseStalePoll();
        await Promise.resolve();
      });
      expect(result.current.projectTitle).toBe("B");
      expect(result.current.planId).toBe("plan-b");
    } finally {
      vi.useRealTimers();
    }
  });

  test("a stale in-flight fetch cannot overwrite a newer refresh", async () => {
    const { result } = renderHook(
      () =>
        usePlanDetailPage({
          projectId: "foo",
          planId: "plan-1",
        }),
      { wrapper }
    );
    await waitFor(() => expect(result.current.ready).toBe(true));

    // A poll tick's fetch is in flight when a user action (e.g. task rerun)
    // triggers an immediate refresh. The older fetch carries pre-action data
    // and resolves late — it must be dropped, not applied over the newer one.
    let resolveStale: (patch: Partial<PlanDetailPageSnapshot>) => void = () =>
      undefined;
    let resolveFresh: (patch: Partial<PlanDetailPageSnapshot>) => void = () =>
      undefined;
    mocks.fetchPlanSnapshot
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveStale = resolve;
          })
      )
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveFresh = resolve;
          })
      );

    let stalePending: Promise<void> = Promise.resolve();
    let freshPending: Promise<void> = Promise.resolve();
    act(() => {
      stalePending = result.current.refreshState();
      freshPending = result.current.refreshState();
    });

    await act(async () => {
      resolveFresh({
        ...buildSnapshotPatch({ planId: "plan-1" }),
        projectTitle: "fresh",
      });
      await freshPending;
    });
    expect(result.current.projectTitle).toBe("fresh");

    await act(async () => {
      resolveStale({
        ...buildSnapshotPatch({ planId: "plan-1" }),
        projectTitle: "stale",
      });
      await stalePending;
    });
    expect(result.current.projectTitle).toBe("fresh");
  });
});
