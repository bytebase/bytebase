import type { ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";
import type { PlanDetailPageState } from "./shell/hooks/types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  localStorage: {
    clear: vi.fn(),
    getItem: vi.fn(() => null),
    removeItem: vi.fn(),
    setItem: vi.fn(),
  },
  usePlanDetailPage: vi.fn(),
}));

vi.stubGlobal("localStorage", mocks.localStorage);

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/components/ui/alert-dialog", () => ({
  AlertDialog: ({ children }: { children: ReactNode }) => <>{children}</>,
  AlertDialogContent: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  AlertDialogFooter: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  AlertDialogTitle: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
}));

vi.mock("@/react/components/ui/badge", () => ({
  Badge: ({ children }: { children: ReactNode }) => <span>{children}</span>,
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
  }: {
    children: ReactNode;
    onClick?: () => void;
  }) => <button onClick={onClick}>{children}</button>,
}));

vi.mock("@/react/lib/utils", () => ({
  cn: (...classes: Array<string | false | null | undefined>) =>
    classes.filter(Boolean).join(" "),
}));

vi.mock("./components/deploy/DeployBranch", () => ({
  DeployBranch: () => <div data-testid="deploy-branch">deploy branch</div>,
}));

vi.mock("./components/review/PlanReviewSection", () => ({
  PlanReviewSection: () => null,
}));

vi.mock("./components/PlanDetailChangesBranch", () => ({
  PlanDetailChangesBranch: ({
    onSelectedSpecIdChange,
    selectedSpecId,
  }: {
    onSelectedSpecIdChange: (specId: string) => void;
    selectedSpecId: string;
  }) => (
    <div>
      <div data-testid="selected-spec-id">{selectedSpecId}</div>
      <button onClick={() => onSelectedSpecIdChange("spec-2")}>
        select second spec
      </button>
    </div>
  ),
}));

vi.mock("./components/PlanDetailDeployFuture", () => ({
  PlanDetailDeployFuture: () => (
    <button data-testid="deploy-future-control">deploy future control</button>
  ),
}));

vi.mock("./components/PlanDetailHeader", () => ({
  PlanDetailHeader: () => null,
}));

vi.mock("./components/PlanDetailHeaderDetails", () => ({
  PlanDetailHeaderDetails: () => null,
}));

vi.mock("./shell/hooks/usePlanDetailPage", () => ({
  usePlanDetailPage: mocks.usePlanDetailPage,
}));

vi.mock("./utils/phaseSummary", () => ({
  buildChangesSummary: () => "",
  buildDeploySummary: () => "",
  buildReviewSummary: () => "",
}));

import { ProjectPlanDetailPage } from "./ProjectPlanDetailPage";

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  vi.clearAllMocks();
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => {
    root.unmount();
  });
  document.body.removeChild(container);
});

const buildPage = (): PlanDetailPageState =>
  ({
    activePhases: new Set(["changes"]),
    bypassLeaveGuardOnce: vi.fn(),
    currentUser: { name: "users/me@example.com" },
    expandPhase: vi.fn(),
    isCreating: true,
    isEditing: false,
    isInitializing: false,
    pageKey: "foo/create/spec-1",
    patchState: vi.fn(),
    pendingLeaveConfirm: false,
    plan: {
      name: "projects/foo/plans/create",
      creator: "users/me@example.com",
      hasRollout: false,
      state: State.ACTIVE,
      specs: [{ id: "spec-1" }, { id: "spec-2" }],
    },
    planCheckRuns: [],
    planId: "create",
    project: { name: "projects/foo", title: "Foo" },
    projectCanCreateRollout: true,
    projectId: "foo",
    projectRequireIssueApproval: false,
    projectRequirePlanCheckNoError: false,
    projectTitle: "Foo",
    readonly: false,
    ready: true,
    refreshState: vi.fn(async () => {}),
    resolveLeaveConfirm: vi.fn(),
    routeName: undefined,
    routePhase: "changes",
    selectedTaskName: undefined,
    setEditing: vi.fn(),
    taskRuns: [],
    togglePhase: vi.fn(),
  }) as unknown as PlanDetailPageState;

const selectedSpecIdText = () =>
  container.querySelector('[data-testid="selected-spec-id"]')?.textContent;

describe("ProjectPlanDetailPage", () => {
  it("keeps local spec selection on creating plans with a stale route spec id", async () => {
    mocks.usePlanDetailPage.mockReturnValue(buildPage());

    await act(async () => {
      root.render(
        <ProjectPlanDetailPage
          planId="create"
          projectId="foo"
          specId="spec-1"
        />
      );
      await Promise.resolve();
    });

    expect(selectedSpecIdText()).toBe("spec-1");

    await act(async () => {
      (
        [...container.querySelectorAll("button")].find(
          (button) => button.textContent === "select second spec"
        ) as HTMLButtonElement
      ).click();
      await Promise.resolve();
    });

    expect(selectedSpecIdText()).toBe("spec-2");
  });

  it("resets spec selection when navigating to another plan", async () => {
    const firstPage = buildPage();
    firstPage.isCreating = false;
    firstPage.pageKey = "foo/plan-1";
    firstPage.planId = "plan-1";
    mocks.usePlanDetailPage.mockReturnValue(firstPage);

    await act(async () => {
      root.render(
        <ProjectPlanDetailPage
          planId="plan-1"
          projectId="foo"
          specId="spec-1"
        />
      );
      await Promise.resolve();
    });

    await act(async () => {
      (
        [...container.querySelectorAll("button")].find(
          (button) => button.textContent === "select second spec"
        ) as HTMLButtonElement
      ).click();
      await Promise.resolve();
    });
    expect(selectedSpecIdText()).toBe("spec-2");

    const secondPage = buildPage();
    secondPage.isCreating = false;
    secondPage.pageKey = "foo/plan-2";
    secondPage.planId = "plan-2";
    secondPage.plan.specs = [
      { id: "new-spec" },
    ] as typeof secondPage.plan.specs;
    mocks.usePlanDetailPage.mockReturnValue(secondPage);
    await act(async () => {
      root.render(<ProjectPlanDetailPage planId="plan-2" projectId="foo" />);
      await Promise.resolve();
    });

    expect(selectedSpecIdText()).toBe("new-spec");
  });

  it("renders the review phase for sheet-backed plans with an issue", async () => {
    mocks.usePlanDetailPage.mockReturnValue({
      ...buildPage(),
      issue: { name: "projects/foo/issues/1" },
    } as unknown as PlanDetailPageState);

    await act(async () => {
      root.render(
        <ProjectPlanDetailPage
          planId="create"
          projectId="foo"
          specId="spec-1"
        />
      );
      await Promise.resolve();
    });

    expect(container.textContent).toContain("plan.navigator.review");
  });

  it("shows the review phase for sheet-backed plans without an issue", async () => {
    mocks.usePlanDetailPage.mockReturnValue(buildPage());

    await act(async () => {
      root.render(
        <ProjectPlanDetailPage
          planId="create"
          projectId="foo"
          specId="spec-1"
        />
      );
      await Promise.resolve();
    });

    // CI/CD UI (sheet-backed) plans always surface the review phase, even
    // before an issue exists — it renders as an upcoming "future" step.
    expect(container.textContent).toContain("plan.navigator.review");
    expect(container.textContent).toContain("plan.phase.review-description");
  });

  it("recognizes a linked Draft Review Issue as the active draft changes phase", async () => {
    const page = buildPage();
    page.isCreating = false;
    page.planId = "1";
    page.plan.name = "projects/foo/plans/1";
    page.plan.issue = "projects/foo/issues/1";
    page.issue = {
      name: "projects/foo/issues/1",
      draft: true,
    } as PlanDetailPageState["issue"];
    mocks.usePlanDetailPage.mockReturnValue(page);

    await act(async () => {
      root.render(
        <ProjectPlanDetailPage planId="1" projectId="foo" specId="spec-1" />
      );
      await Promise.resolve();
    });

    expect(container.textContent).toContain("common.draft");
    expect(container.textContent).not.toContain("common.under-review");
  });

  it("does not surface rollout controls for a draft issue with stale rollout data", async () => {
    const page = buildPage();
    page.isCreating = false;
    page.planId = "1";
    page.pageKey = "foo/1";
    page.plan.name = "projects/foo/plans/1";
    page.plan.issue = "projects/foo/issues/1";
    page.issue = {
      name: "projects/foo/issues/1",
      draft: true,
    } as PlanDetailPageState["issue"];
    page.rollout = {
      name: "projects/foo/plans/1/rollout",
      stages: [
        {
          name: "projects/foo/plans/1/rollout/stages/prod",
          tasks: [{ status: 1 }],
        },
      ],
    } as PlanDetailPageState["rollout"];
    mocks.usePlanDetailPage.mockReturnValue(page);

    await act(async () => {
      root.render(
        <ProjectPlanDetailPage planId="1" projectId="foo" specId="spec-1" />
      );
      await Promise.resolve();
    });

    expect(container.textContent).toContain("common.draft");
    expect(container.querySelector("[data-testid='deploy-branch']")).toBeNull();
    expect(
      container.querySelector("[data-testid='deploy-future-control']")
    ).toBeNull();
    expect(container.textContent).not.toContain("common.not-started");
  });

  it("shows an incomplete state for a persisted plan without its Draft Review Issue", async () => {
    const page = buildPage();
    page.isCreating = false;
    page.planId = "1";
    page.plan.name = "projects/foo/plans/1";
    page.plan.issue = "";
    page.issue = undefined;
    mocks.usePlanDetailPage.mockReturnValue(page);

    await act(async () => {
      root.render(
        <ProjectPlanDetailPage planId="1" projectId="foo" specId="spec-1" />
      );
      await Promise.resolve();
    });

    expect(container.textContent).toContain("plan.lifecycle.incomplete");
  });

  it("hides the review phase for GitOps plans with release-backed specs", async () => {
    const page = buildPage();
    page.plan.specs = [
      {
        id: "spec-1",
        config: {
          case: "changeDatabaseConfig",
          value: {
            release: "projects/foo/releases/abc",
            sheet: "",
            targets: [],
            enablePriorBackup: false,
          },
        },
      },
    ] as unknown as PlanDetailPageState["plan"]["specs"];
    mocks.usePlanDetailPage.mockReturnValue(page);

    await act(async () => {
      root.render(
        <ProjectPlanDetailPage
          planId="create"
          projectId="foo"
          specId="spec-1"
        />
      );
      await Promise.resolve();
    });

    expect(container.textContent).not.toContain("plan.navigator.review");
  });
});
