import type { TFunction } from "i18next";
import { act, createElement, type ReactNode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { ApprovalStatus, State } from "@/types/proto-es/v1/common_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import {
  createPlanStagesColumn,
  type PlanColumn,
  PlanLifecycleBadges,
  PlanTable,
} from "./PlanTable";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  routerPush: vi.fn(),
  getEnvironmentByName: vi.fn(),
}));

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/app/router", () => ({
  router: {
    push: mocks.routerPush,
    resolve: ({ params }: { params: Record<string, string> }) => ({
      fullPath: `/projects/${params.projectId}/plans/${params.planId}`,
    }),
  },
}));

vi.mock("@/components/TaskStatusIcon", () => ({
  TaskStatusIcon: () => null,
}));

vi.mock("@/components/ui/table", () => ({
  Table: ({ children }: { children: ReactNode }) =>
    createElement("table", {}, children),
  TableBody: ({ children }: { children: ReactNode }) =>
    createElement("tbody", {}, children),
  TableCell: ({ children }: { children?: ReactNode }) =>
    createElement("td", {}, children),
  TableHead: ({ children }: { children?: ReactNode }) =>
    createElement("th", {}, children),
  TableHeader: ({ children }: { children: ReactNode }) =>
    createElement("thead", {}, children),
  TableRow: ({ children, ...props }: { children?: ReactNode }) =>
    createElement("tr", props, children),
}));

vi.mock("@/hooks/useColumnWidths", () => ({
  useColumnWidths: (columns: { defaultWidth: number }[]) => ({
    widths: columns.map((column) => column.defaultWidth),
    totalWidth: columns.reduce((sum, column) => sum + column.defaultWidth, 0),
    onResizeStart: vi.fn(),
    setWidths: vi.fn(),
  }),
}));

vi.mock("@/stores/app", () => {
  const state = {
    environmentList: [],
    getEnvironmentByName: mocks.getEnvironmentByName,
    getUserByIdentifier: () => undefined,
  };
  return {
    useAppStore: (selector: (s: typeof state) => unknown) => selector(state),
  };
});

const basePlan = {
  name: "projects/foo/plans/7",
  title: "Plan 7",
  state: State.ACTIVE,
  creator: "users/creator@example.com",
  issue: "",
  hasRollout: true,
  approvalStatus: ApprovalStatus.APPROVAL_STATUS_UNSPECIFIED,
  specs: [],
  rolloutStageSummaries: [],
} as unknown as Plan;

const titleColumn: PlanColumn = {
  key: "title",
  title: "Title",
  defaultWidth: 200,
  render: (plan, ctx) => (
    <span>
      {plan.title}
      <PlanLifecycleBadges ctx={ctx} />
    </span>
  ),
};

describe("PlanTable", () => {
  let container: HTMLDivElement;
  let root: Root;

  const render = (plans: Plan[], props: Partial<Parameters<typeof PlanTable>[0]> = {}) => {
    act(() => {
      root.render(
        <PlanTable
          plans={plans}
          projectId="foo"
          columns={[titleColumn, createPlanStagesColumn(((key: string) => key) as unknown as TFunction)]}
          isMobile={false}
          onOpenPlan={vi.fn()}
          {...props}
        />
      );
    });
  };

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.clearAllMocks();
  });

  test("renders the given columns", () => {
    render([basePlan]);

    expect(
      Array.from(container.querySelectorAll("th")).map((th) => th.textContent)
    ).toEqual(["Title", "rollout.stage.self"]);
  });

  test("renders a stage chain with environment titles", () => {
    mocks.getEnvironmentByName.mockImplementation((name: string) =>
      name === "environments/prod" ? { title: "Production" } : undefined
    );
    render([
      {
        ...basePlan,
        rolloutStageSummaries: [
          { stage: "projects/foo/plans/7/rollout/stages/test", taskStatusCounts: [] },
          { stage: "projects/foo/plans/7/rollout/stages/prod", taskStatusCounts: [] },
        ],
      } as unknown as Plan,
    ]);

    const stages = container.querySelectorAll("td")[1].textContent;
    expect(stages).toContain("environments/test");
    expect(stages).toContain("Production");
  });

  test("shows - for a Plan without rollout stages", () => {
    render([basePlan]);

    expect(container.querySelectorAll("td")[1].textContent).toBe("-");
  });

  test("opens the URL the caller resolves, in a new tab on ctrl-click", () => {
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    const onOpenPlan = vi.fn();
    render([basePlan], {
      onOpenPlan,
      resolvePlanUrl: (plan) => `/${plan.name}/rollout`,
    });
    const row = container.querySelector<HTMLElement>(
      '[data-testid="plan-list-item"]'
    );

    act(() => row?.click());
    expect(onOpenPlan).toHaveBeenCalledOnce();
    expect(mocks.routerPush).toHaveBeenCalledWith("/projects/foo/plans/7/rollout");

    act(() => {
      row?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, ctrlKey: true })
      );
    });
    expect(open).toHaveBeenCalledWith("/projects/foo/plans/7/rollout", "_blank");
    expect(mocks.routerPush).toHaveBeenCalledOnce();
    open.mockRestore();
  });

  test("defaults to Plan Detail", () => {
    render([basePlan]);

    act(() =>
      container
        .querySelector<HTMLElement>('[data-testid="plan-list-item"]')
        ?.click()
    );

    expect(mocks.routerPush).toHaveBeenCalledWith("/projects/foo/plans/7");
  });

  test.each([
    {
      name: "a closed Plan as Closed",
      plan: { state: State.DELETED },
      expected: "common.closed",
    },
    {
      name: "a ClickOps Plan without an issue or rollout as Incomplete",
      plan: { hasRollout: false },
      expected: "plan.lifecycle.incomplete",
    },
  ])("badges $name", ({ plan, expected }) => {
    render([{ ...basePlan, ...plan } as Plan]);

    expect(container.querySelectorAll("td")[0].textContent).toBe(
      `Plan 7${expected}`
    );
  });

  test("never badges a GitOps Plan as Incomplete", () => {
    render([
      {
        ...basePlan,
        hasRollout: false,
        specs: [
          {
            config: {
              case: "changeDatabaseConfig",
              value: { release: "projects/foo/releases/r1" },
            },
          },
        ],
      } as unknown as Plan,
    ]);

    expect(container.querySelectorAll("td")[0].textContent).toBe("Plan 7");
  });
});
