import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue, IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import {
  IssueComment_Approval_Status,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { PlanDetailPageState } from "../../shell/hooks/types";
import { PlanDetailProvider } from "../../shell/PlanDetailContext";
import { PlanReviewSection } from "./PlanReviewSection";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

// ---------------------------------------------------------------------------
// Hoisted mocks (vi.hoisted runs before imports so vi.mock factories can
// close over these values).
// ---------------------------------------------------------------------------
const mocks = vi.hoisted(() => ({
  comments: [] as IssueComment[],
  getIssueComments: vi.fn((_issueName: string) => [] as IssueComment[]),
  listIssueComments: vi.fn(async () => ({ issueComments: [] })),
  getOrFetchProjectByName: vi.fn(async () => ({})),
  loadProjectIamPolicy: vi.fn(async () => ({})),
  getUserByIdentifier: vi.fn(() => undefined),
  roleList: [] as unknown[],
  projectsByName: {} as Record<string, unknown>,
  projectPoliciesByName: {} as Record<string, unknown>,
  groupsByName: {} as Record<string, unknown>,
  batchGetOrFetchUsers: vi.fn(async () => []),
  batchGetOrFetchGroups: vi.fn(async () => []),
  createIssueComment: vi.fn(async () => ({})),
  updateIssueComment: vi.fn(async () => ({})),
  getProjectByName: vi.fn(() => ({
    name: "projects/p1",
    allowSelfApproval: true,
  })),
  requestIssue: vi.fn(async () => ({})),
  approveIssue: vi.fn(async () => ({})),
  rejectIssue: vi.fn(async () => ({})),
  createRollout: vi.fn(async () => ({ name: "rollouts/r1", stages: [] })),
  routerPush: vi.fn(),
  routerResolve: vi.fn(() => ({
    fullPath: "/projects/p1/issues/123",
    href: "/projects/p1/issues/123",
  })),
  currentUser: {
    email: "me@example.com",
    name: "users/me@example.com",
    title: "Me",
  },
}));

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (params && Object.keys(params).length > 0) {
        return `${key}:${JSON.stringify(params)}`;
      }
      return key;
    },
  }),
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: unknown) => unknown) =>
      selector({
        getIssueComments: (issueName: string) =>
          mocks.getIssueComments(issueName),
        getUserByIdentifier: mocks.getUserByIdentifier,
        roleList: mocks.roleList,
        projectsByName: mocks.projectsByName,
        projectPoliciesByName: mocks.projectPoliciesByName,
        groupsByName: mocks.groupsByName,
        batchGetOrFetchUsers: mocks.batchGetOrFetchUsers,
        batchGetOrFetchGroups: mocks.batchGetOrFetchGroups,
        loadProjectIamPolicy: mocks.loadProjectIamPolicy,
      }),
    {
      getState: () => ({
        listIssueComments: mocks.listIssueComments,
        getOrFetchProjectByName: mocks.getOrFetchProjectByName,
        loadProjectIamPolicy: mocks.loadProjectIamPolicy,
        getProjectByName: mocks.getProjectByName,
        createIssueComment: mocks.createIssueComment,
        updateIssueComment: mocks.updateIssueComment,
      }),
    }
  ),
}));

vi.mock("@/connect", () => ({
  issueServiceClientConnect: {
    requestIssue: mocks.requestIssue,
    approveIssue: mocks.approveIssue,
    rejectIssue: mocks.rejectIssue,
  },
  rolloutServiceClientConnect: {
    createRollout: mocks.createRollout,
  },
}));

vi.mock("@/react/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/react/router")>()),
  router: {
    push: mocks.routerPush,
    resolve: mocks.routerResolve,
  },
}));

vi.mock("@/store", () => ({
  pushNotification: vi.fn(),
  extractUserEmail: (identifier: string) => identifier.replace(/^users\//, ""),
}));

vi.mock("@/store/modules/v1/common", () => ({
  projectNamePrefix: "projects/",
  roleNamePrefix: "roles/",
  userNamePrefix: "users/",
  issueNamePrefix: "issues/",
  getProjectIdIssueIdIssueCommentId: (name: string) => {
    const parts = name.split("/");
    return { projectId: parts[1], issueId: parts[3], issueCommentId: parts[5] };
  },
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useCurrentUser: () => mocks.currentUser,
}));

vi.mock("@/react/hooks/useProjectByName", () => ({
  useProjectByName: () => ({
    name: "projects/p1",
    allowSelfApproval: true,
  }),
}));

vi.mock("@/react/components/MarkdownEditor", () => ({
  MarkdownEditor: ({ content }: { content?: string }) => (
    <div data-testid="markdown">{content}</div>
  ),
}));

vi.mock("@/react/components/HumanizeTs", () => ({
  HumanizeTs: ({ ts }: { ts: number }) => (
    <span data-testid="humanize-ts">{ts}</span>
  ),
}));

vi.mock("@/react/components/UserAvatar", () => ({
  UserAvatar: ({ title }: { title?: string }) => (
    <span data-testid="user-avatar">{title}</span>
  ),
  getAvatarColor: () => "#000",
  getInitials: (name: string) => name.slice(0, 2),
}));

vi.mock("@/react/components/RouterLink", () => ({
  RouterLink: ({
    children,
    to,
  }: {
    children: React.ReactNode;
    to: unknown;
  }) => <a href={JSON.stringify(to)}>{children}</a>,
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock("@/react/components/ui/badge", () => ({
  Badge: ({ children }: { children: React.ReactNode }) => (
    <span>{children}</span>
  ),
}));

vi.mock("@/react/components/ui/popover", () => ({
  Popover: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  PopoverTrigger: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  PopoverContent: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}));

vi.mock("@/react/components/ui/alert", () => ({
  Alert: ({
    children,
    title,
  }: {
    children: React.ReactNode;
    title?: React.ReactNode;
  }) => (
    <div data-testid="alert">
      {title && <div data-testid="alert-title">{title}</div>}
      <div data-testid="alert-body">{children}</div>
    </div>
  ),
  alertVariants: {},
}));

vi.mock("@/react/lib/utils", () => ({
  cn: (...classes: Array<string | false | null | undefined>) =>
    classes.filter(Boolean).join(" "),
}));

vi.mock("@/react/lib/role", () => ({
  displayRoleTitleFromList: (role: string) => role,
}));

vi.mock("@/react/lib/plan/diffPlanSpecs", () => ({
  diffPlanSpecsForEvent: () => [],
  diffEntryKey: (entry: { kind: string }) => entry.kind,
}));

vi.mock("@/utils", () => ({
  displayRoleTitle: (role: { title?: string; name?: string }) =>
    role.title ?? role.name ?? "",
  ensureUserFullName: (name: string) => name,
  isBindingPolicyExpired: () => false,
  memberMapToRolesInProjectIAM: () => new Map(),
}));

vi.mock("@/utils/iam/permission", () => ({
  hasProjectPermissionV2: () => false,
}));

vi.mock("@/utils/v1/issue/issue", () => ({
  extractIssueUID: (name: string) => name.split("/").pop() ?? "",
}));

vi.mock("@/utils/v1/issue/plan", () => ({
  enablePriorBackupOfSpec: () => false,
}));

vi.mock("@/types", () => ({
  unknownUser: (principal: string) => ({
    name: principal,
    email: principal.replace(/^users\//, ""),
    title: principal.replace(/^users\//, ""),
  }),
  getTimeForPbTimestampProtoEs: () => 0,
}));

vi.mock("@/react/stores/app/issueComment", () => ({
  IssueCommentType: {
    APPROVAL: "APPROVAL",
    USER_COMMENT: "USER_COMMENT",
    ISSUE_UPDATE: "ISSUE_UPDATE",
    PLAN_UPDATE: "PLAN_UPDATE",
  },
  getIssueCommentType: (comment: { event?: { case?: string } }) => {
    if (comment.event?.case === "approval") return "APPROVAL";
    if (comment.event?.case === "issueUpdate") return "ISSUE_UPDATE";
    if (comment.event?.case === "planUpdate") return "PLAN_UPDATE";
    return "USER_COMMENT";
  },
}));

vi.mock("@/react/stores/app/group", () => ({
  ensureGroupIdentifier: (id: string) => id,
}));

vi.mock("@/types/v1/user", () => ({
  AccountType: { USER: "USER", SERVICE_ACCOUNT: "SERVICE_ACCOUNT" },
  getAccountTypeByEmail: () => "USER",
  groupBindingPrefix: "groups/",
}));

// Mock the heavy sub-components that have intractable fan-out:
// - ReviewApprovalFlow: uses ResizeObserver, many store subscriptions, Tooltip
// - ReviewActivityTimeline: uses many comment utilities, comment composer
// Both are mocked to null. ReviewRejectionBanner and ReviewReadinessFooter
// are kept real because the assertions target them directly.
vi.mock("./ReviewApprovalFlow", () => ({
  ReviewApprovalFlow: () => null,
  deriveSteps: (issue: Issue) => {
    const roles = issue.approvalTemplate?.flow?.roles ?? [];
    return roles.map((role: string, index: number) => ({
      index,
      role,
      status: "pending" as const,
      approver: undefined,
    }));
  },
}));

vi.mock("./ReviewActivityTimeline", () => ({
  ReviewActivityTimeline: () => null,
}));

// Import React for JSX — must come after vi.mock calls.
import React from "react";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const makeIssueComment = (status: IssueComment_Approval_Status): IssueComment =>
  ({
    name: "projects/p1/issues/123/issueComments/1",
    comment: "Needs rework",
    creator: "users/reviewer@example.com",
    createTime: undefined,
    updateTime: undefined,
    event: {
      case: "approval" as const,
      value: { status },
    },
  }) as unknown as IssueComment;

const makeIssue = ({
  approvalStatus,
  status = IssueStatus.OPEN,
  roles = ["roles/PROJECT_OWNER"],
}: {
  approvalStatus: ApprovalStatus;
  status?: IssueStatus;
  roles?: string[];
}): Issue =>
  ({
    name: "projects/p1/issues/123",
    creator: "users/creator@example.com",
    approvalStatus,
    status,
    approvers: [],
    approvalTemplate: {
      flow: { roles },
      title: "Test policy",
    },
    riskLevel: 0,
    updateTime: undefined,
  }) as unknown as Issue;

const makePlan = ({
  hasRollout = false,
  planCheckRunStatusCount = {} as Record<string, number>,
}: {
  hasRollout?: boolean;
  planCheckRunStatusCount?: Record<string, number>;
} = {}): Plan =>
  ({
    name: "projects/p1/plans/1",
    creator: "users/me@example.com",
    specs: [],
    hasRollout,
    planCheckRunStatusCount,
    createTime: undefined,
  }) as unknown as Plan;

const makePageState = (
  issue: Issue,
  plan: Plan,
  overrides: Partial<PlanDetailPageState> = {}
): PlanDetailPageState =>
  ({
    projectId: "p1",
    planId: "1",
    pageKey: "projects/p1/plans/1",
    projectTitle: "Project 1",
    plan,
    issue,
    readonly: false,
    projectCanCreateRollout: true,
    projectRequireIssueApproval: false,
    projectRequirePlanCheckNoError: false,
    isCreating: false,
    isInitializing: false,
    isEditing: false,
    isRefreshing: false,
    isRunningChecks: false,
    setIsRunningChecks: vi.fn(),
    ready: true,
    patchState: vi.fn(),
    refreshState: vi.fn(async () => undefined),
    activePhases: new Set(["review"]),
    routeName: undefined,
    routePhase: undefined,
    routeStageId: undefined,
    routeTaskId: undefined,
    selectedTaskName: undefined,
    layoutMode: "NONE",
    containerWidth: 0,
    pendingLeaveConfirm: false,
    planCheckRuns: [],
    taskRuns: [],
    rollout: undefined,
    currentUser: mocks.currentUser,
    project: { name: "projects/p1" },
    bypassLeaveGuardOnce: vi.fn(),
    setEditing: vi.fn(),
    togglePhase: vi.fn(),
    expandPhase: vi.fn(),
    closeTaskPanel: vi.fn(),
    resolveLeaveConfirm: vi.fn(),
    ...overrides,
  }) as unknown as PlanDetailPageState;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  mocks.comments = [];
  mocks.getIssueComments.mockImplementation(() => mocks.comments);
  vi.clearAllMocks();
  mocks.getIssueComments.mockImplementation(() => mocks.comments);
});

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("PlanReviewSection — five review states", () => {
  test("in-progress (PENDING, checks pass): footer shows waiting-on-review + muted underline bypass link", () => {
    const issue = makeIssue({ approvalStatus: ApprovalStatus.PENDING });
    const plan = makePlan();
    const { container, render, unmount } = renderIntoContainer(
      <PlanDetailProvider value={makePageState(issue, plan)}>
        <PlanReviewSection />
      </PlanDetailProvider>
    );

    render();

    expect(container.textContent).toContain(
      "plan.review.footer.waiting-on-review"
    );
    // Muted underline link — not a primary Button
    const underlineBtn = container.querySelector("button.underline");
    expect(underlineBtn).not.toBeNull();
    expect(underlineBtn?.textContent).toContain(
      "plan.review.footer.bypass-and-deploy"
    );
    // Should NOT be a primary Button element (no class "underline" on shared Button)
    expect(container.textContent).not.toContain(
      "plan.review.footer.blocked-by-rejection"
    );
    expect(container.textContent).not.toContain(
      "plan.review.footer.all-gates-passed"
    );

    unmount();
  });

  // BYT-9709: with failed checks the rollout will NOT be created automatically
  // after approval, so the waiting-review line must not promise auto-creation.
  test("in-progress (PENDING) + checks failed: footer drops the auto-rollout promise", () => {
    const issue = makeIssue({ approvalStatus: ApprovalStatus.PENDING });
    const plan = makePlan({
      planCheckRunStatusCount: { ERROR: 1, SUCCESS: 2 },
    });
    const { container, render, unmount } = renderIntoContainer(
      <PlanDetailProvider value={makePageState(issue, plan)}>
        <PlanReviewSection />
      </PlanDetailProvider>
    );

    render();

    expect(container.textContent).toContain(
      "plan.review.footer.waiting-on-review"
    );
    expect(container.textContent).not.toContain(
      "plan.review.footer.auto-rollout-after-approval"
    );
    expect(container.textContent).toContain(
      "plan.review.footer.rollout-blocked-by-failed-checks"
    );

    unmount();
  });

  test("rejected (REJECTED + rejection comment): rejection banner renders + footer shows blocked-by-rejection + bypass action (explicit override)", () => {
    const issue = makeIssue({ approvalStatus: ApprovalStatus.REJECTED });
    const plan = makePlan();
    const rejectionComment = makeIssueComment(
      IssueComment_Approval_Status.REJECTED
    );
    mocks.comments = [rejectionComment];
    mocks.getIssueComments.mockImplementation(() => mocks.comments);

    const { container, render, unmount } = renderIntoContainer(
      <PlanDetailProvider value={makePageState(issue, plan)}>
        <PlanReviewSection />
      </PlanDetailProvider>
    );

    render();

    // Rejection banner should be visible
    const alertTitle = container.querySelector("[data-testid='alert-title']");
    expect(alertTitle).not.toBeNull();
    expect(alertTitle?.textContent).toContain("plan.review.rejection.title");

    // Footer shows blocked-by-rejection
    expect(container.textContent).toContain(
      "plan.review.footer.blocked-by-rejection"
    );

    // A rejected review still offers an explicit bypass-and-deploy override
    // (the page state grants bb.rollouts.create and is not readonly).
    expect(container.textContent).toContain(
      "plan.review.footer.bypass-and-deploy"
    );

    unmount();
  });

  test("approved + checks failed (APPROVED, ERROR>0): footer shows approved-but-checks-failed + primary Button bypass", () => {
    const issue = makeIssue({ approvalStatus: ApprovalStatus.APPROVED });
    // Simulate error checks via planCheckRunStatusCount using Advice_Level.ERROR key
    const plan = makePlan({
      planCheckRunStatusCount: { ERROR: 3, SUCCESS: 2 },
    });

    const { container, render, unmount } = renderIntoContainer(
      <PlanDetailProvider value={makePageState(issue, plan)}>
        <PlanReviewSection />
      </PlanDetailProvider>
    );

    render();

    expect(container.textContent).toContain(
      "plan.review.footer.approved-but-checks-failed"
    );

    // Primary Button — the shared Button component renders without the underline
    // class; verify bypass-and-deploy text is present.
    expect(container.textContent).toContain(
      "plan.review.footer.bypass-and-deploy"
    );
    // Must NOT be the muted underline link
    const underlineBtn = container.querySelector("button.underline");
    expect(underlineBtn).toBeNull();

    unmount();
  });

  test("approved + checks passed (APPROVED, checks pass): footer shows all-gates-passed + muted underline link", () => {
    const issue = makeIssue({ approvalStatus: ApprovalStatus.APPROVED });
    const plan = makePlan({ planCheckRunStatusCount: { SUCCESS: 5 } });

    const { container, render, unmount } = renderIntoContainer(
      <PlanDetailProvider value={makePageState(issue, plan)}>
        <PlanReviewSection />
      </PlanDetailProvider>
    );

    render();

    expect(container.textContent).toContain(
      "plan.review.footer.all-gates-passed"
    );
    // Muted underline link present
    const underlineBtn = container.querySelector("button.underline");
    expect(underlineBtn).not.toBeNull();
    expect(underlineBtn?.textContent).toContain(
      "plan.review.footer.bypass-and-deploy"
    );

    unmount();
  });

  test("skipped (SKIPPED): footer shows all-gates-passed", () => {
    const issue = makeIssue({
      approvalStatus: ApprovalStatus.SKIPPED,
      roles: [],
    });
    const plan = makePlan({ planCheckRunStatusCount: { SUCCESS: 2 } });

    const { container, render, unmount } = renderIntoContainer(
      <PlanDetailProvider value={makePageState(issue, plan)}>
        <PlanReviewSection />
      </PlanDetailProvider>
    );

    render();

    // The "no approval required" note is rendered by ReviewApprovalFlow itself
    // (mocked out here; covered in ReviewApprovalFlow.test.tsx). The footer
    // treats a skipped approval as satisfied.
    expect(container.textContent).toContain(
      "plan.review.footer.all-gates-passed"
    );

    unmount();
  });

  test("footer disappears once a rollout exists (hasRollout true → no bypass action)", () => {
    const issue = makeIssue({ approvalStatus: ApprovalStatus.PENDING });
    const plan = makePlan({ hasRollout: true });

    const { container, render, unmount } = renderIntoContainer(
      <PlanDetailProvider value={makePageState(issue, plan)}>
        <PlanReviewSection />
      </PlanDetailProvider>
    );

    render();

    // Footer is entirely hidden — no footer keys, no bypass button
    expect(container.textContent).not.toContain(
      "plan.review.footer.waiting-on-review"
    );
    expect(container.textContent).not.toContain(
      "plan.review.footer.bypass-and-deploy"
    );
    expect(container.textContent).not.toContain(
      "plan.review.footer.all-gates-passed"
    );

    unmount();
  });
});
