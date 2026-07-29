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
import { IssueStatus, State } from "@/types/proto-es/v1/common_pb";
import type { PlanDetailPageState } from "../shell/hooks/types";
import { PlanDetailHeader } from "./PlanDetailHeader";
import { PlanDetailHeaderDetails } from "./PlanDetailHeaderDetails";

const mocks = vi.hoisted(() => ({
  batchUpdateIssuesStatus: vi.fn(),
  creationIssueLabels: [] as string[],
  createIssue: vi.fn(),
  createPlan: vi.fn(),
  invalidateProjectPlansPagedDataCache: vi.fn(),
  lifecycle: { kind: "none" } as { kind: string },
  page: undefined as unknown as PlanDetailPageState,
  patchState: vi.fn(),
  permissions: new Set<string>(),
  pushNotification: vi.fn(),
  replaceRoute: vi.fn(),
  setCreationIssueLabels: vi.fn(),
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

vi.mock("@/api", () => ({
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

vi.mock("@/components/MarkdownEditor", () => ({
  MarkdownEditor: ({ content }: { content: string }) => <span>{content}</span>,
}));

vi.mock("@/components/ui/button", () => ({
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

vi.mock("@/components/ui/textarea", () => ({
  Textarea: (props: TextareaHTMLAttributes<HTMLTextAreaElement>) => (
    <textarea {...props} />
  ),
}));

vi.mock("@/components/ui/popover", () => ({
  Popover: ({ children, open }: { children: ReactNode; open?: boolean }) =>
    open ? <div>{children}</div> : null,
  PopoverContent: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock("@/components/ui/dropdown-menu", () => ({
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

vi.mock("@/lib/projectPagedDataCache", () => ({
  invalidateProjectPlansPagedDataCache:
    mocks.invalidateProjectPlansPagedDataCache,
}));

vi.mock("@/lib/utils", () => ({
  cn: (...classes: Array<string | false | null | undefined>) =>
    classes.filter(Boolean).join(" "),
}));

vi.mock("@/app/router", () => ({
  router: { replace: mocks.replaceRoute },
}));

vi.mock("@/stores/app", () => {
  const getState = () => ({
    createSheet: vi.fn(),
    listIssueComments: vi.fn(),
  });
  return { useAppStore: { getState } };
});

vi.mock("@/stores", () => ({ pushNotification: mocks.pushNotification }));

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
    creationIssueLabels: mocks.creationIssueLabels,
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
    setCreationIssueLabels: mocks.setCreationIssueLabels,
    setEditing: vi.fn(),
    setIsRunningChecks: vi.fn(),
    taskRuns: [],
    taskRunsByTaskName: new Map(),
    togglePhase: vi.fn(),
  }) as unknown as PlanDetailPageState;

beforeEach(() => {
  vi.clearAllMocks();
  mocks.creationIssueLabels = [];
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

  test("does not carry a revealed notice to the next plan", () => {
    mocks.permissions = new Set(["bb.plans.update", "bb.issues.update"]);
    mocks.lifecycle = { kind: "ready-for-review" };
    mocks.page = {
      ...makePage(),
      isEditing: true,
    } as unknown as PlanDetailPageState;
    const { rerender } = render(<PlanDetailHeader />);

    fireEvent.click(
      screen.getByRole("button", { name: "plan.ready-for-review" })
    );
    expect(screen.getByRole("alert")).toHaveTextContent(
      "plan.editor.save-changes-before-continuing"
    );

    mocks.page = { ...mocks.page, pageKey: "plan-456" };
    rerender(<PlanDetailHeader />);

    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  test("submits the draft in one press and surfaces a single failure", async () => {
    const failure = new Error("approval setup failed");
    mocks.permissions = new Set(["bb.plans.update", "bb.issues.update"]);
    mocks.lifecycle = { kind: "ready-for-review" };
    mocks.page = {
      ...makePage(),
      issue: { ...makePage().issue!, labels: ["old"] },
    };
    mocks.updateIssue.mockRejectedValueOnce(failure);
    render(<PlanDetailHeader />);

    fireEvent.click(
      screen.getByRole("button", { name: "plan.ready-for-review" })
    );

    // No form stands between the press and the submission.
    expect(
      screen.queryByRole("button", { name: "common.confirm" })
    ).not.toBeInTheDocument();

    await waitFor(() => expect(mocks.updateIssue).toHaveBeenCalledOnce());
    expect(mocks.updateIssue).toHaveBeenCalledWith(
      expect.objectContaining({
        issue: expect.objectContaining({ draft: false }),
        updateMask: { paths: ["draft"] },
      })
    );
    await waitFor(() =>
      expect(mocks.pushNotification).toHaveBeenCalledWith({
        module: "bytebase",
        style: "CRITICAL",
        title: "common.failed",
        description: String(failure),
      })
    );
    expect(mocks.updateIssue).toHaveBeenCalledOnce();
    expect(mocks.patchState).not.toHaveBeenCalled();
  });

  test("leaves label edits to the metadata row", async () => {
    mocks.permissions = new Set(["bb.plans.update", "bb.issues.update"]);
    mocks.lifecycle = { kind: "ready-for-review" };
    mocks.page = {
      ...makePage(),
      issue: { ...makePage().issue!, labels: ["keep-me"] },
    };
    render(<PlanDetailHeader />);

    fireEvent.click(
      screen.getByRole("button", { name: "plan.ready-for-review" })
    );

    await waitFor(() => expect(mocks.updateIssue).toHaveBeenCalledOnce());
    expect(
      mocks.updateIssue.mock.calls[0][0].updateMask.paths
    ).not.toContain("labels");
  });

  test("lists a missing update permission instead of disabling the action", () => {
    mocks.permissions = new Set(["bb.plans.update"]);
    mocks.lifecycle = { kind: "ready-for-review" };
    mocks.page = makePage();
    render(<PlanDetailHeader />);

    const submit = screen.getByRole("button", {
      name: "plan.ready-for-review",
    });
    expect(submit).toBeEnabled();
    fireEvent.click(submit);

    expect(screen.getByRole("alert")).toHaveTextContent(
      "plan.draft-update-permission-required"
    );
    expect(mocks.updateIssue).not.toHaveBeenCalled();
  });

  test("confirms a named override before submitting past failed checks", async () => {
    mocks.permissions = new Set(["bb.plans.update", "bb.issues.update"]);
    mocks.lifecycle = { kind: "ready-for-review" };
    const page = makePage();
    mocks.page = {
      ...page,
      plan: {
        ...page.plan,
        planCheckRunStatusCount: { ERROR: 1 },
        specs: [{ config: { case: "changeDatabaseConfig", value: {} } }],
      },
    } as unknown as PlanDetailPageState;
    render(<PlanDetailHeader />);

    fireEvent.click(
      screen.getByRole("button", { name: "plan.ready-for-review" })
    );

    expect(mocks.updateIssue).not.toHaveBeenCalled();
    expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "plan.submit-review-anyway" })
    );

    await waitFor(() => expect(mocks.updateIssue).toHaveBeenCalledOnce());
  });

  test("blocks rather than confirms failed checks where SQL review is enforced", () => {
    mocks.permissions = new Set(["bb.plans.update", "bb.issues.update"]);
    mocks.lifecycle = { kind: "ready-for-review" };
    const page = makePage();
    mocks.page = {
      ...page,
      plan: {
        ...page.plan,
        planCheckRunStatusCount: { ERROR: 1 },
        specs: [{ config: { case: "changeDatabaseConfig", value: {} } }],
      },
      project: { ...page.project, enforceSqlReview: true },
    } as unknown as PlanDetailPageState;
    render(<PlanDetailHeader />);

    fireEvent.click(
      screen.getByRole("button", { name: "plan.ready-for-review" })
    );

    expect(
      screen.queryByRole("button", { name: "plan.submit-review-anyway" })
    ).not.toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent(
      "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
    );
    expect(mocks.updateIssue).not.toHaveBeenCalled();
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

  test("creates directly without a confirmation panel", async () => {
    mocks.permissions = new Set(["bb.plans.create", "bb.issues.create"]);
    mocks.lifecycle = { kind: "create" };
    mocks.page = makePage({ creating: true });
    mocks.createPlan.mockResolvedValue({
      ...mocks.page.plan,
      name: "projects/p1/plans/123",
    });
    mocks.createIssue.mockResolvedValue({
      draft: true,
      labels: [],
      name: "projects/p1/issues/456",
      plan: "projects/p1/plans/123",
    });

    render(<PlanDetailHeader />);

    expect(screen.getByRole("button", { name: "common.create" })).toBeEnabled();
    fireEvent.click(screen.getByRole("button", { name: "common.create" }));

    expect(
      screen.queryByRole("button", { name: "common.confirm" })
    ).not.toBeInTheDocument();
    await waitFor(() => expect(mocks.createPlan).toHaveBeenCalledOnce());
  });

  test("allows draft creation when review labels are still missing", () => {
    mocks.permissions = new Set(["bb.plans.create", "bb.issues.create"]);
    mocks.lifecycle = { kind: "create" };
    const page = makePage({ creating: true });
    mocks.page = {
      ...page,
      project: {
        ...page.project,
        forceIssueLabels: true,
      },
    };

    render(<PlanDetailHeader />);

    expect(screen.getByRole("button", { name: "common.create" })).toBeEnabled();
  });

  test("lists every blocker instead of disabling Create", () => {
    mocks.permissions = new Set(["bb.plans.create", "bb.issues.create"]);
    mocks.lifecycle = { kind: "create" };
    const page = makePage({ creating: true });
    mocks.page = {
      ...page,
      plan: { ...page.plan, title: "" },
    };

    render(<PlanDetailHeader />);

    const create = screen.getByRole("button", { name: "common.create" });
    expect(create).toBeEnabled();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();

    fireEvent.click(create);

    expect(mocks.createPlan).not.toHaveBeenCalled();
    const notice = screen.getByRole("alert");
    expect(notice).toHaveTextContent("plan.cannot-create");
    expect(notice).toHaveTextContent("plan.title-required");
  });

  test("puts the cursor in the empty title when that is the blocker", () => {
    mocks.permissions = new Set(["bb.plans.create", "bb.issues.create"]);
    mocks.lifecycle = { kind: "create" };
    const page = makePage({ creating: true });
    mocks.page = { ...page, plan: { ...page.plan, title: "" } };

    render(<PlanDetailHeader />);
    // Create mode focuses the title on mount; move focus away so the assertion
    // is about the blocked press and not about that.
    const createButton = screen.getByRole("button", { name: "common.create" });
    createButton.focus();
    expect(document.activeElement).toBe(createButton);

    fireEvent.click(createButton);

    expect(document.activeElement).toBe(
      screen.getByPlaceholderText("common.untitled")
    );
  });

  test("clears the create notice as the blocker resolves", () => {
    mocks.permissions = new Set(["bb.plans.create", "bb.issues.create"]);
    mocks.lifecycle = { kind: "create" };
    const page = makePage({ creating: true });
    mocks.page = { ...page, plan: { ...page.plan, title: "" } };
    const { rerender } = render(<PlanDetailHeader />);
    fireEvent.click(screen.getByRole("button", { name: "common.create" }));
    expect(screen.getByRole("alert")).toBeInTheDocument();

    mocks.page = { ...mocks.page, plan: { ...mocks.page.plan, title: "Named" } };
    rerender(<PlanDetailHeader />);

    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  test("lists a missing create permission instead of disabling Create", () => {
    mocks.permissions = new Set(["bb.plans.create"]);
    mocks.lifecycle = { kind: "create" };
    mocks.page = makePage({ creating: true });

    render(<PlanDetailHeader />);

    const create = screen.getByRole("button", { name: "common.create" });
    expect(create).toBeEnabled();
    fireEvent.click(create);

    expect(mocks.createPlan).not.toHaveBeenCalled();
    expect(screen.getByRole("alert")).toHaveTextContent(
      "common.missing-required-permission"
    );
  });

  test("creates the draft Issue with labels selected on the preview page", async () => {
    mocks.creationIssueLabels = ["preview-label"];
    mocks.permissions = new Set([
      "bb.plans.create",
      "bb.issues.create",
      "bb.issues.update",
    ]);
    mocks.lifecycle = { kind: "create" };
    mocks.page = makePage({ creating: true });
    mocks.createPlan.mockResolvedValue({
      ...mocks.page.plan,
      name: "projects/p1/plans/123",
    });
    mocks.createIssue.mockResolvedValue({
      draft: true,
      labels: ["preview-label"],
      name: "projects/p1/issues/456",
      plan: "projects/p1/plans/123",
    });

    render(<PlanDetailHeader />);

    fireEvent.click(screen.getByRole("button", { name: "common.create" }));

    await waitFor(() => expect(mocks.createIssue).toHaveBeenCalledOnce());
    expect(mocks.createIssue).toHaveBeenCalledWith(
      expect.objectContaining({
        issue: expect.objectContaining({ labels: ["preview-label"] }),
      })
    );
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
    expect(mocks.invalidateProjectPlansPagedDataCache).toHaveBeenCalledWith(
      "p1"
    );
    expect(
      mocks.updateIssue.mock.calls.map(([request]) => request.updateMask)
    ).toEqual(
      expect.arrayContaining([{ paths: ["title"] }, { paths: ["description"] }])
    );
    expect(mocks.updatePlan).not.toHaveBeenCalled();
  });
});
