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

vi.mock("@/react/components/ui/sheet", () => ({
  Sheet: ({ children, open }: { children: ReactNode; open: boolean }) =>
    open ? <div>{children}</div> : null,
  SheetBody: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetContent: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  SheetHeader: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetTitle: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));

vi.mock("@/react/lib/utils", () => ({
  cn: (...classes: Array<string | false | null | undefined>) =>
    classes.filter(Boolean).join(" "),
}));

vi.mock("./components/deploy/DeployBranch", () => ({
  DeployBranch: () => null,
}));

vi.mock("./components/deploy/DeployTaskDetailPanel", () => ({
  DeployTaskDetailPanel: () => null,
}));

vi.mock("./components/PlanDetailApprovalFlow", () => ({
  PlanDetailReviewApprovalFlow: () => null,
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
  PlanDetailDeployFuture: () => null,
}));

vi.mock("./components/PlanDetailHeader", () => ({
  PlanDetailHeader: () => null,
}));

vi.mock("./shell/constants", () => ({
  INLINE_TASK_PANEL_BREAKPOINT_PX: 1024,
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
    closeTaskPanel: vi.fn(),
    containerWidth: 1200,
    currentUser: { name: "users/me@example.com" },
    expandPhase: vi.fn(),
    isCreating: true,
    isEditing: false,
    isInitializing: false,
    isRefreshing: false,
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
    layoutMode: "NONE",
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

  it("renders the review phase for sheet-backed plans", async () => {
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

    expect(container.textContent).toContain("plan.navigator.review");
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
