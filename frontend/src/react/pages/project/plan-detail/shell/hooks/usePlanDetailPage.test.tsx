import { renderHook, waitFor } from "@testing-library/react";
import { act, type ReactNode } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";
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
          pageHost: null,
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
          pageHost: null,
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
          pageHost: null,
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
          pageHost: null,
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
          pageHost: null,
        }),
      { wrapper }
    );

    await waitFor(() => expect(result.current.ready).toBe(true));
    expect(result.current.activePhases.has("changes")).toBe(true);
    expect(result.current.activePhases.has("review")).toBe(true);
    expect(result.current.activePhases.has("deploy")).toBe(false);
  });

  test("defaults to only the changes phase on the specs route for rollout plans", async () => {
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
          pageHost: null,
        }),
      { wrapper }
    );

    await waitFor(() => expect(result.current.ready).toBe(true));
    expect(result.current.activePhases.has("changes")).toBe(true);
    expect(result.current.activePhases.has("review")).toBe(false);
    expect(result.current.activePhases.has("deploy")).toBe(false);
  });

  test("keeps changes and review on the spec-detail route for reviewed rollout plans", async () => {
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
          pageHost: null,
        }),
      { wrapper }
    );

    await waitFor(() => expect(result.current.ready).toBe(true));
    expect(result.current.activePhases.has("changes")).toBe(true);
    expect(result.current.activePhases.has("review")).toBe(true);
    expect(result.current.activePhases.has("deploy")).toBe(false);
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
          pageHost: null,
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
          pageHost: null,
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
          pageHost: null,
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
          pageHost: null,
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
          pageHost: null,
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
});
