import { create } from "@bufbuild/protobuf";
import { renderHook, waitFor } from "@testing-library/react";
import { act, type ReactNode } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";
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
  fetchRolloutState: vi.fn(),
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
  fetchRolloutState: mocks.fetchRolloutState,
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
    mocks.fetchRolloutState.mockResolvedValue({});
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

  test("poll ticks use the slim status lane while a task is active", async () => {
    vi.useFakeTimers();
    try {
      const activePatch = {
        ...buildSnapshotPatch({ planId: "plan-1" }),
        rollout: create(RolloutSchema, {
          name: "projects/foo/rollouts/1",
          stages: [
            {
              name: "projects/foo/rollouts/1/stages/s1",
              tasks: [
                {
                  name: "projects/foo/rollouts/1/stages/s1/tasks/t1",
                  status: Task_Status.RUNNING,
                },
              ],
            },
          ],
        }),
      };
      mocks.fetchPlanSnapshot.mockResolvedValue(activePatch);
      mocks.fetchRolloutState.mockResolvedValue({
        rollout: activePatch.rollout,
        taskRuns: [],
      });

      const { result } = renderHook(
        () => usePlanDetailPage({ projectId: "foo", planId: "plan-1" }),
        { wrapper }
      );

      // Flush the initial full fetch so the RUNNING rollout is in the snapshot.
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0);
      });
      expect(result.current.ready).toBe(true);

      mocks.fetchPlanSnapshot.mockClear();
      mocks.fetchRolloutState.mockClear();

      // Cross the first poll tick (scheduled at the 1000ms base, jittered). With
      // a task RUNNING the tick must take the slim status lane, not re-fetch the
      // full page.
      await act(async () => {
        await vi.advanceTimersByTimeAsync(1300);
      });
      expect(mocks.fetchRolloutState).toHaveBeenCalled();
      expect(mocks.fetchPlanSnapshot).not.toHaveBeenCalled();
    } finally {
      vi.useRealTimers();
    }
  });

  test("wakes up to the fast status lane when a scheduled task comes due", async () => {
    vi.useFakeTimers();
    try {
      const BASE = 1_700_000_000_000;
      vi.setSystemTime(BASE);
      // A PENDING task scheduled to run 10s from now — not active yet.
      const scheduled = {
        ...buildSnapshotPatch({ planId: "plan-1" }),
        rollout: create(RolloutSchema, {
          name: "projects/foo/rollouts/1",
          stages: [
            {
              name: "projects/foo/rollouts/1/stages/s1",
              tasks: [
                {
                  name: "projects/foo/rollouts/1/stages/s1/tasks/t1",
                  status: Task_Status.PENDING,
                  runTime: { seconds: BigInt(Math.floor(BASE / 1000) + 10) },
                },
              ],
            },
          ],
        }),
      };
      mocks.fetchPlanSnapshot.mockResolvedValue(scheduled);
      mocks.fetchRolloutState.mockResolvedValue({
        rollout: scheduled.rollout,
        taskRuns: [],
      });

      const { result } = renderHook(
        () => usePlanDetailPage({ projectId: "foo", planId: "plan-1" }),
        { wrapper }
      );
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0);
      });
      expect(result.current.ready).toBe(true);
      mocks.fetchRolloutState.mockClear();

      // Before the run_time the task is not active, so no slim-lane poll runs.
      await act(async () => {
        await vi.advanceTimersByTimeAsync(2000);
      });
      expect(mocks.fetchRolloutState).not.toHaveBeenCalled();

      // Crossing the run_time fires the wake-up, which flips the activity flag.
      await act(async () => {
        await vi.advanceTimersByTimeAsync(11000);
      });
      // Now that the task is active, a following poll tick takes the fast status
      // lane (a separate advance so the flag's re-render has committed first).
      await act(async () => {
        await vi.advanceTimersByTimeAsync(35000);
      });
      expect(mocks.fetchRolloutState).toHaveBeenCalled();
    } finally {
      vi.useRealTimers();
    }
  });

  test("a slim poll tick does not supersede an in-flight full refresh", async () => {
    vi.useFakeTimers();
    try {
      const activePatch = {
        ...buildSnapshotPatch({ planId: "plan-1" }),
        projectTitle: "orig",
        rollout: create(RolloutSchema, {
          name: "projects/foo/rollouts/1",
          stages: [
            {
              name: "projects/foo/rollouts/1/stages/s1",
              tasks: [
                {
                  name: "projects/foo/rollouts/1/stages/s1/tasks/t1",
                  status: Task_Status.RUNNING,
                },
              ],
            },
          ],
        }),
      };
      mocks.fetchPlanSnapshot.mockResolvedValue(activePatch);
      mocks.fetchRolloutState.mockResolvedValue({
        rollout: activePatch.rollout,
        taskRuns: [],
      });

      const { result } = renderHook(
        () => usePlanDetailPage({ projectId: "foo", planId: "plan-1" }),
        { wrapper }
      );
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0);
      });
      expect(result.current.ready).toBe(true);
      mocks.fetchRolloutState.mockClear();

      // A user action triggers a full refresh whose fetch stays in flight. The
      // restart also arms the fast poll (the task is RUNNING).
      let resolveFull: (patch: Partial<PlanDetailPageSnapshot>) => void = () =>
        undefined;
      mocks.fetchPlanSnapshot.mockReturnValueOnce(
        new Promise((resolve) => {
          resolveFull = resolve;
        })
      );
      let refreshDone: Promise<void> = Promise.resolve();
      act(() => {
        refreshDone = result.current.refreshState();
      });

      // The restarted fast poll fires while the full fetch is in flight; the
      // slim lane must be skipped so it can't drop the richer full result.
      await act(async () => {
        await vi.advanceTimersByTimeAsync(2000);
      });
      expect(mocks.fetchRolloutState).not.toHaveBeenCalled();

      // The full refresh resolves and its plan-wide update (projectTitle) lands.
      await act(async () => {
        resolveFull({ ...activePatch, projectTitle: "updated-by-action" });
        await refreshDone;
      });
      expect(result.current.projectTitle).toBe("updated-by-action");
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
