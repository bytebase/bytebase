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
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/router", () => ({
  router: {
    push: mocks.routerPush,
    replace: mocks.routerReplace,
  },
}));

vi.mock("@/router/dashboard/projectV1", () => ({
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL: "project.plan.detail.spec.detail",
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS: "project.plan.detail.specs",
}));

vi.mock("@/router/dashboard/projectV1RouteHelpers", () => ({
  getRouteQueryString: (value: string | string[] | undefined) =>
    Array.isArray(value) ? value[0] : value,
  PLAN_DETAIL_PHASE_CHANGES: "changes",
  PLAN_DETAIL_PHASE_REVIEW: "review",
  PLAN_DETAIL_PHASE_DEPLOY: "deploy",
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
    pageKey: `foo/${planId}/`,
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
      specs: [{ id: "spec-1" }],
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
          routeName: "project.plan.detail.specs",
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

  test("defaults to only the review phase after an issue is created", async () => {
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
    expect(result.current.activePhases.has("changes")).toBe(false);
    expect(result.current.activePhases.has("review")).toBe(true);
    expect(result.current.activePhases.has("deploy")).toBe(false);
  });
});
