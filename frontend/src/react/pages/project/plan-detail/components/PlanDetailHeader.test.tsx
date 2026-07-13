import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import {
  type ButtonHTMLAttributes,
  cloneElement,
  isValidElement,
  type ReactElement,
  type ReactNode,
  type TextareaHTMLAttributes,
} from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { PlanDetailPageState } from "../shell/hooks/types";
import { PlanDetailHeader } from "./PlanDetailHeader";
import { PlanDetailHeaderDetails } from "./PlanDetailHeaderDetails";

const mocks = vi.hoisted(() => ({
  batchUpdateIssuesStatus: vi.fn(),
  createIssue: vi.fn(),
  createPlan: vi.fn(),
  lifecycle: { kind: "none" } as { kind: string },
  page: undefined as unknown as PlanDetailPageState,
  patchState: vi.fn(),
  permissions: new Set<string>(),
  replaceRoute: vi.fn(),
  updateIssue: vi.fn(),
  updatePlan: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@bufbuild/protobuf", () => ({
  clone: (_schema: unknown, message: Record<string, unknown>) => ({
    ...message,
  }),
  create: (_schema: unknown, init?: Record<string, unknown>) => ({ ...init }),
}));

vi.mock("@/connect", () => ({
  issueServiceClientConnect: {
    batchUpdateIssuesStatus: mocks.batchUpdateIssuesStatus,
    createIssue: mocks.createIssue,
    updateIssue: mocks.updateIssue,
  },
  planServiceClientConnect: {
    createPlan: mocks.createPlan,
    updatePlan: mocks.updatePlan,
  },
}));

vi.mock("@/react/components/IssueLabelSelect", () => ({
  IssueLabelSelect: ({
    onChange,
    selected,
  }: {
    onChange: (labels: string[]) => void;
    selected: string[];
  }) => (
    <div>
      <output data-testid="selected-labels">{selected.join(",")}</output>
      <button onClick={() => onChange(["replacement"])} type="button">
        select replacement label
      </button>
    </div>
  ),
}));

vi.mock("@/react/components/MarkdownEditor", () => ({
  MarkdownEditor: ({ content }: { content: string }) => <span>{content}</span>,
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    appearance: _appearance,
    children,
    size: _size,
    variant: _variant,
    ...props
  }: ButtonHTMLAttributes<HTMLButtonElement> & {
    appearance?: string;
    size?: string;
    variant?: string;
  }) => <button {...props}>{children}</button>,
}));

vi.mock("@/react/components/ui/checkbox", () => ({
  Checkbox: ({
    checked,
    onCheckedChange,
  }: {
    checked: boolean;
    onCheckedChange?: (checked: boolean) => void;
  }) => (
    <input
      checked={checked}
      onChange={(event) => onCheckedChange?.(event.target.checked)}
      type="checkbox"
    />
  ),
}));

vi.mock("@/react/components/ui/textarea", () => ({
  Textarea: (props: TextareaHTMLAttributes<HTMLTextAreaElement>) => (
    <textarea {...props} />
  ),
}));

vi.mock("@/react/components/ui/popover", () => ({
  Popover: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  PopoverContent: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  PopoverTrigger: ({
    children,
    render,
  }: {
    children: ReactNode;
    render?: ReactElement;
  }) =>
    isValidElement(render) ? (
      cloneElement(render, {}, children)
    ) : (
      <>{children}</>
    ),
}));

vi.mock("@/react/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  DropdownMenuContent: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  DropdownMenuItem: ({
    children,
    onClick,
  }: {
    children: ReactNode;
    onClick?: () => void;
  }) => <button onClick={onClick}>{children}</button>,
  DropdownMenuTrigger: ({
    children,
    render,
  }: {
    children: ReactNode;
    render?: ReactElement;
  }) =>
    isValidElement(render) ? (
      cloneElement(render, {}, children)
    ) : (
      <>{children}</>
    ),
}));

vi.mock("@/react/lib/utils", () => ({
  cn: (...classes: Array<string | false | null | undefined>) =>
    classes.filter(Boolean).join(" "),
}));

vi.mock("@/react/router", () => ({
  router: { replace: mocks.replaceRoute },
}));

vi.mock("@/react/stores/app", () => {
  const getState = () => ({
    createSheet: vi.fn(),
    listIssueComments: vi.fn(),
  });
  return { useAppStore: { getState } };
});

vi.mock("@/store", () => ({ pushNotification: vi.fn() }));

vi.mock("@/utils", () => ({
  extractPlanUID: (name: string) => name.split("/").at(-1) ?? "",
  extractProjectResourceName: () => "p1",
  extractSheetUID: (name: string) => name,
  hasProjectPermissionV2: (_project: unknown, permission: string) =>
    mocks.permissions.has(permission),
}));

vi.mock("../hooks/usePlanDetailSpecValidation", () => ({
  usePlanDetailSpecValidation: () => ({ emptySpecIdSet: new Set<string>() }),
}));

vi.mock("../shell/focusPhase", () => ({ focusPlanPhase: vi.fn() }));
vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => mocks.page,
}));
vi.mock("../utils/localSheet", () => ({
  getLocalSheetByName: vi.fn(),
  removeLocalSheet: vi.fn(),
}));
vi.mock("./PlanDetailMeta", () => ({ PlanDetailMeta: () => null }));
vi.mock("./lifecycle/PlanLifecycleSlot", () => ({
  PlanLifecycleSlot: () => null,
}));
vi.mock("./lifecycle/PlanLifecycleStamp", () => ({
  PlanLifecycleStamp: () => null,
}));
vi.mock("./lifecycle/planLifecycleHeaderState", () => ({
  slotHasPrimaryControl: () => false,
}));
vi.mock("./lifecycle/usePlanLifecycleHeader", () => ({
  usePlanLifecycleHeader: () => mocks.lifecycle,
}));

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((next) => {
    resolve = next;
  });
  return { promise, resolve };
}

const makePage = ({
  draft = true,
  issueStatus = IssueStatus.OPEN,
  planState = State.ACTIVE,
  creating = false,
}: {
  draft?: boolean;
  issueStatus?: IssueStatus;
  planState?: State;
  creating?: boolean;
} = {}): PlanDetailPageState =>
  ({
    activePhases: new Set(),
    bypassLeaveGuardOnce: vi.fn(),
    currentUser: { email: "reviewer@example.com", name: "users/reviewer" },
    expandPhase: vi.fn(),
    isCreating: creating,
    isEditing: false,
    isInitializing: false,
    isRunningChecks: false,
    issue: creating
      ? undefined
      : {
          description: "Stale issue description",
          draft,
          labels: [],
          name: "projects/p1/issues/456",
          plan: "projects/p1/plans/123",
          status: issueStatus,
          title: "Stale issue title",
        },
    pageKey: "plan-123",
    patchState: mocks.patchState,
    pendingLeaveConfirm: false,
    plan: {
      creator: "users/owner",
      description: "Plan description",
      hasRollout: false,
      issue: creating ? "" : "projects/p1/issues/456",
      name: creating ? "" : "projects/p1/plans/123",
      specs: [],
      state: planState,
      title: "Plan title",
    },
    planCheckRuns: [],
    planId: creating ? "create" : "123",
    project: {
      enforceSqlReview: false,
      forceIssueLabels: false,
      issueLabels: [],
      name: "projects/p1",
    },
    projectCanCreateRollout: false,
    projectId: "p1",
    projectRequireIssueApproval: false,
    projectRequirePlanCheckNoError: false,
    projectTitle: "Project One",
    readonly: false,
    ready: true,
    refreshState: vi.fn(async () => undefined),
    resolveLeaveConfirm: vi.fn(),
    rollout: undefined,
    setEditing: vi.fn(),
    setIsRunningChecks: vi.fn(),
    taskRuns: [],
    taskRunsByTaskName: new Map(),
    togglePhase: vi.fn(),
  }) as unknown as PlanDetailPageState;

beforeEach(() => {
  vi.clearAllMocks();
  mocks.permissions = new Set(["bb.plans.update"]);
  mocks.lifecycle = { kind: "none" };
  mocks.page = makePage();
  mocks.updatePlan.mockImplementation(async (request) => request.plan);
  mocks.updateIssue.mockImplementation(async (request) => request.issue);
  mocks.batchUpdateIssuesStatus.mockResolvedValue({});
  vi.spyOn(window, "confirm").mockReturnValue(true);
});

describe("PlanDetailHeader draft ownership", () => {
  test("edits draft title and description through UpdatePlan with plan permission only", async () => {
    render(
      <>
        <PlanDetailHeader />
        <PlanDetailHeaderDetails />
      </>
    );

    const title = screen.getByDisplayValue("Plan title");
    expect(title).toBeEnabled();
    fireEvent.focus(title);
    fireEvent.change(title, { target: { value: "Updated plan title" } });
    fireEvent.blur(title);

    fireEvent.click(screen.getByText("Plan description"));
    const description = screen.getByDisplayValue("Plan description");
    fireEvent.change(description, {
      target: { value: "Updated plan description" },
    });
    fireEvent.click(screen.getByRole("button", { name: "common.save" }));

    await waitFor(() => expect(mocks.updatePlan).toHaveBeenCalledTimes(2));
    expect(mocks.updatePlan.mock.calls.map(([request]) => request)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          plan: expect.objectContaining({ title: "Updated plan title" }),
          updateMask: { paths: ["title"] },
        }),
        expect.objectContaining({
          plan: expect.objectContaining({
            description: "Updated plan description",
          }),
          updateMask: { paths: ["description"] },
        }),
      ])
    );
    expect(mocks.updateIssue).not.toHaveBeenCalled();
    expect(mocks.patchState).toHaveBeenCalledWith({
      plan: expect.objectContaining({ title: "Updated plan title" }),
    });
    expect(mocks.patchState).toHaveBeenCalledWith({
      plan: expect.objectContaining({
        description: "Updated plan description",
      }),
    });
  });

  test("preserves dirty title and description edits across polling updates", () => {
    const { rerender } = render(
      <>
        <PlanDetailHeader />
        <PlanDetailHeaderDetails />
      </>
    );
    const title = screen.getByDisplayValue("Plan title");
    fireEvent.focus(title);
    fireEvent.change(title, { target: { value: "Local draft title" } });
    fireEvent.click(screen.getByText("Plan description"));
    fireEvent.change(screen.getByDisplayValue("Plan description"), {
      target: { value: "Local draft description" },
    });

    mocks.page = {
      ...mocks.page,
      plan: {
        ...mocks.page.plan,
        description: "Polled plan description",
        title: "Polled plan title",
      },
    };
    rerender(
      <>
        <PlanDetailHeader />
        <PlanDetailHeaderDetails />
      </>
    );

    expect(screen.getByDisplayValue("Local draft title")).toBe(title);
    expect(screen.getByDisplayValue("Local draft description")).toBeVisible();
  });

  test("resets dirty metadata when navigating to another plan", () => {
    const { rerender } = render(
      <>
        <PlanDetailHeader />
        <PlanDetailHeaderDetails />
      </>
    );
    const title = screen.getByDisplayValue("Plan title");
    fireEvent.focus(title);
    fireEvent.change(title, { target: { value: "Old local title" } });
    fireEvent.click(screen.getByText("Plan description"));
    fireEvent.change(screen.getByDisplayValue("Plan description"), {
      target: { value: "Old local description" },
    });

    mocks.page = {
      ...makePage(),
      pageKey: "plan-456",
      plan: {
        ...makePage().plan,
        description: "New plan description",
        name: "projects/p1/plans/456",
        title: "New plan title",
      },
    };
    rerender(
      <>
        <PlanDetailHeader />
        <PlanDetailHeaderDetails />
      </>
    );

    expect(screen.getByDisplayValue("New plan title")).toBeVisible();
    expect(screen.queryByDisplayValue("Old local title")).toBeNull();
    expect(screen.getByText("New plan description")).toBeVisible();
    expect(screen.queryByDisplayValue("Old local description")).toBeNull();
  });

  test("collapses an expanded description after navigation", () => {
    const longDescription = "A".repeat(180);
    mocks.page = {
      ...makePage(),
      plan: { ...makePage().plan, description: longDescription },
    };
    const { rerender } = render(<PlanDetailHeaderDetails />);

    fireEvent.click(screen.getByRole("button", { name: "common.show-more" }));
    expect(
      screen.getByRole("button", { name: "common.show-less" })
    ).toBeVisible();

    mocks.page = {
      ...makePage(),
      pageKey: "plan-456",
      plan: {
        ...makePage().plan,
        description: "B".repeat(180),
        name: "projects/p1/plans/456",
      },
    };
    rerender(<PlanDetailHeaderDetails />);

    expect(
      screen.getByRole("button", { name: "common.show-more" })
    ).toBeVisible();
  });

  test("resets review labels and warning acknowledgment after navigation", () => {
    mocks.permissions = new Set(["bb.plans.update", "bb.issues.update"]);
    mocks.lifecycle = { kind: "ready-for-review" };
    mocks.page = {
      ...makePage(),
      issue: { ...makePage().issue!, labels: ["old"] },
      plan: {
        ...makePage().plan,
        planCheckRunStatusCount: { ERROR: 1 },
        specs: [{ config: { case: "changeDatabaseConfig", value: {} } }],
      },
    } as unknown as PlanDetailPageState;
    const { rerender } = render(<PlanDetailHeader />);

    fireEvent.click(
      screen.getByRole("button", { name: "select replacement label" })
    );
    fireEvent.click(screen.getByRole("checkbox"));
    expect(screen.getByTestId("selected-labels")).toHaveTextContent(
      "replacement"
    );
    expect(screen.getByRole("checkbox")).toBeChecked();

    mocks.page = {
      ...mocks.page,
      pageKey: "plan-456",
      issue: { ...mocks.page.issue!, labels: ["new"] },
      plan: {
        ...mocks.page.plan,
        name: "projects/p1/plans/456",
      },
    };
    rerender(<PlanDetailHeader />);

    expect(screen.getByTestId("selected-labels")).toHaveTextContent("new");
    expect(screen.getByRole("checkbox")).not.toBeChecked();
  });

  test("ignores a title update response from the previous plan", async () => {
    const pending = deferred<Record<string, unknown>>();
    mocks.updatePlan.mockReturnValueOnce(pending.promise);
    const { rerender } = render(<PlanDetailHeader />);
    const title = screen.getByDisplayValue("Plan title");
    fireEvent.focus(title);
    fireEvent.change(title, { target: { value: "Old saved title" } });
    fireEvent.blur(title);

    mocks.page = {
      ...makePage(),
      pageKey: "plan-456",
      plan: {
        ...makePage().plan,
        name: "projects/p1/plans/456",
        title: "New plan title",
      },
    };
    rerender(<PlanDetailHeader />);
    pending.resolve({
      name: "projects/p1/plans/123",
      title: "Old saved title",
    });

    await waitFor(() =>
      expect(screen.getByDisplayValue("New plan title")).toBeVisible()
    );
    expect(mocks.patchState).not.toHaveBeenCalled();
  });

  test("ignores a description update response from the previous plan", async () => {
    const pending = deferred<Record<string, unknown>>();
    mocks.updatePlan.mockReturnValueOnce(pending.promise);
    const { rerender } = render(<PlanDetailHeaderDetails />);
    fireEvent.click(screen.getByText("Plan description"));
    fireEvent.change(screen.getByDisplayValue("Plan description"), {
      target: { value: "Old saved description" },
    });
    fireEvent.click(screen.getByRole("button", { name: "common.save" }));

    mocks.page = {
      ...makePage(),
      pageKey: "plan-456",
      plan: {
        ...makePage().plan,
        description: "New plan description",
        name: "projects/p1/plans/456",
      },
    };
    rerender(<PlanDetailHeaderDetails />);
    pending.resolve({
      description: "Old saved description",
      name: "projects/p1/plans/123",
    });

    await waitFor(() =>
      expect(screen.getByText("New plan description")).toBeVisible()
    );
    expect(mocks.patchState).not.toHaveBeenCalled();
  });

  test.each([
    ["common.close", State.ACTIVE, State.DELETED],
    ["common.reopen", State.DELETED, State.ACTIVE],
  ])("uses UpdatePlan for draft %s", async (label, initialState, nextState) => {
    mocks.page = makePage({ planState: initialState });
    render(<PlanDetailHeader />);

    fireEvent.click(screen.getByRole("button", { name: label }));

    await waitFor(() => expect(mocks.updatePlan).toHaveBeenCalledOnce());
    expect(mocks.updatePlan).toHaveBeenCalledWith(
      expect.objectContaining({
        plan: expect.objectContaining({ state: nextState }),
        updateMask: { paths: ["state"] },
      })
    );
    expect(mocks.batchUpdateIssuesStatus).not.toHaveBeenCalled();
    await waitFor(() =>
      expect(mocks.patchState).toHaveBeenCalledWith({
        plan: expect.objectContaining({ state: nextState }),
        issue: expect.objectContaining({
          status:
            nextState === State.DELETED
              ? IssueStatus.CANCELED
              : IssueStatus.OPEN,
        }),
      })
    );
  });

  test("keeps draft creation enabled and warns when issue update permission is missing", () => {
    mocks.permissions = new Set(["bb.plans.create", "bb.issues.create"]);
    mocks.lifecycle = { kind: "create" };
    mocks.page = makePage({ creating: true });

    render(<PlanDetailHeader />);

    expect(screen.getByRole("button", { name: "common.create" })).toBeEnabled();
    expect(screen.getByRole("alert")).toHaveTextContent(
      "plan.draft-update-permission-required"
    );
    expect(
      screen.getByRole("button", { name: "common.confirm" })
    ).toBeEnabled();
  });
});

describe("PlanDetailHeader submitted issue behavior", () => {
  test("keeps submitted metadata and close mutations on the Issue service", async () => {
    mocks.permissions = new Set(["bb.issues.update"]);
    mocks.page = makePage({ draft: false });
    render(
      <>
        <PlanDetailHeader />
        <PlanDetailHeaderDetails />
      </>
    );

    const title = screen.getByDisplayValue("Stale issue title");
    fireEvent.focus(title);
    fireEvent.change(title, { target: { value: "Submitted issue title" } });
    fireEvent.blur(title);

    fireEvent.click(screen.getByText("Stale issue description"));
    fireEvent.change(screen.getByDisplayValue("Stale issue description"), {
      target: { value: "Submitted issue description" },
    });
    fireEvent.click(screen.getByRole("button", { name: "common.save" }));

    fireEvent.click(
      screen.getByRole("button", { name: "issue.batch-transition.close" })
    );

    await waitFor(() => expect(mocks.updateIssue).toHaveBeenCalledTimes(2));
    await waitFor(() =>
      expect(mocks.batchUpdateIssuesStatus).toHaveBeenCalledOnce()
    );
    expect(
      mocks.updateIssue.mock.calls.map(([request]) => request.updateMask)
    ).toEqual(
      expect.arrayContaining([{ paths: ["title"] }, { paths: ["description"] }])
    );
    expect(mocks.updatePlan).not.toHaveBeenCalled();
  });
});
