import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_Approver_Status } from "@/types/proto-es/v1/issue_service_pb";
import type { PlanDetailPageState } from "../shell/hooks/types";
import { PlanDetailProvider } from "../shell/PlanDetailContext";
import { PlanDetailReviewApprovalFlow } from "./PlanDetailApprovalFlow";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  comments: [] as Array<{
    comment?: string;
    creator: string;
    event?: { case?: string };
  }>,
  getIssueComments: vi.fn(() => mocks.comments),
  listIssueComments: vi.fn(async () => ({ issueComments: mocks.comments })),
  requestIssue: vi.fn(async () => ({})),
  routerPush: vi.fn(),
  useTranslation: vi.fn(() => ({
    t: (key: string) => key,
  })),
  getOrFetchProjectByName: vi.fn(async () => ({})),
  getOrFetchProjectIamPolicy: vi.fn(async () => ({})),
  getProjectByName: vi.fn(() => ({ allowSelfApproval: false })),
  getProjectIamPolicy: vi.fn(() => ({ bindings: [] })),
  batchGetOrFetchGroups: vi.fn(async () => []),
  batchGetOrFetchUsers: vi.fn(async () => []),
  getOrFetchUserByIdentifier: vi.fn(async () => undefined),
  pushNotification: vi.fn(),
  currentUserStore: {
    value: { email: "me@example.com", name: "users/me" },
  },
  groupStore: {
    batchGetOrFetchGroups: vi.fn(async () => []),
  },
  projectIamPolicyStore: {
    getOrFetchProjectIamPolicy: vi.fn(async () => ({})),
    getProjectIamPolicy: vi.fn(() => ({ bindings: [] })),
  },
  projectStore: {
    getOrFetchProjectByName: vi.fn(async () => ({})),
    getProjectByName: vi.fn(() => ({ allowSelfApproval: false })),
  },
  roleStore: {
    roleList: [
      {
        name: "roles/PROJECT_OWNER",
        title: "Project Owner",
      },
    ],
  },
  workspaceStore: {
    roleMapToUsers: new Map(),
  },
  userStore: {
    batchGetOrFetchUsers: vi.fn(async () => []),
    getOrFetchUserByIdentifier: vi.fn(async () => undefined),
  },
  issueCommentStore: {
    getIssueComments: vi.fn(() => mocks.comments),
    listIssueComments: vi.fn(async () => ({ issueComments: mocks.comments })),
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/router", () => ({
  router: {
    push: mocks.routerPush,
  },
}));

vi.mock("@/react/components/FeatureBadge", () => ({
  FeatureBadge: () => <span data-testid="feature-badge">feature</span>,
}));

vi.mock("@/connect", () => ({
  issueServiceClientConnect: {
    requestIssue: mocks.requestIssue,
  },
}));

vi.mock("@/store", () => ({
  pushNotification: mocks.pushNotification,
  useCurrentUserV1: () => mocks.currentUserStore,
  useGroupStore: () => mocks.groupStore,
  useProjectIamPolicyStore: () => mocks.projectIamPolicyStore,
  useProjectV1Store: () => mocks.projectStore,
  useRoleStore: () => mocks.roleStore,
  useWorkspaceV1Store: () => mocks.workspaceStore,
  useUserStore: () => mocks.userStore,
}));

vi.mock("@/store/modules/v1/common", () => ({
  projectNamePrefix: "projects/",
  roleNamePrefix: "roles/",
  userNamePrefix: "users/",
}));

vi.mock("@/store/modules/v1/issueComment", () => ({
  IssueCommentType: {
    APPROVAL: "APPROVAL",
  },
  getIssueCommentType: (comment: { event?: { case?: string } }) =>
    comment.event?.case === "approval" ? "APPROVAL" : "USER_COMMENT",
  useIssueCommentStore: () => mocks.issueCommentStore,
}));

vi.mock("@/utils", () => ({
  displayRoleTitle: (role: { title?: string; name?: string }) =>
    role.title ?? role.name ?? "",
  ensureUserFullName: (user: { title?: string; name?: string }) =>
    user.title ?? user.name ?? "",
  isBindingPolicyExpired: () => false,
  memberMapToRolesInProjectIAM: () => new Map(),
}));

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

const makePageState = ({
  issue,
  readonly = false,
}: {
  issue?: Issue;
  readonly?: boolean;
}): PlanDetailPageState =>
  ({
    activePhases: new Set(["review"]),
    closeTaskPanel: vi.fn(),
    containerWidth: 0,
    expandPhase: vi.fn(),
    isCreating: false,
    isEditing: false,
    isInitializing: false,
    isRefreshing: false,
    issue,
    pageKey: "projects/p1/plans/1",
    patchState: vi.fn(),
    plan: { creator: "users/me", name: "projects/p1/plans/1" },
    planCheckRuns: [],
    planId: "1",
    projectId: "p1",
    projectTitle: "Project 1",
    readonly,
    ready: true,
    refreshState: vi.fn(async () => undefined),
    rollout: undefined,
    routeName: undefined,
    routePhase: undefined,
    routeStageId: undefined,
    routeTaskId: undefined,
    selectedTaskName: undefined,
    setEditing: vi.fn(),
    layoutMode: "NONE",
    specId: undefined,
    taskRuns: [],
    togglePhase: vi.fn(),
  }) as unknown as PlanDetailPageState;

const makeIssue = ({
  approvalStatus,
  approvers = [],
  roles = ["roles/PROJECT_OWNER"],
  creator = "users/me",
}: {
  approvalStatus: ApprovalStatus;
  approvers?: Array<Record<string, unknown>>;
  roles?: string[];
  creator?: string;
}): Issue =>
  ({
    approvalStatus,
    approvalTemplate: {
      description: "Sensitive databases",
      flow: {
        roles,
      },
      title: "Prod policy",
    },
    approvers,
    creator,
    name: "projects/p1/issues/123",
    riskLevel: 2,
  }) as unknown as Issue;

beforeEach(() => {
  mocks.comments = [];
  mocks.issueCommentStore.getIssueComments.mockClear();
  mocks.issueCommentStore.listIssueComments.mockClear();
  mocks.requestIssue.mockClear();
  mocks.routerPush.mockClear();
  mocks.pushNotification.mockClear();
  mocks.groupStore.batchGetOrFetchGroups.mockClear();
  mocks.projectIamPolicyStore.getOrFetchProjectIamPolicy.mockClear();
  mocks.projectIamPolicyStore.getProjectIamPolicy.mockClear();
  mocks.projectStore.getOrFetchProjectByName.mockClear();
  mocks.projectStore.getProjectByName.mockClear();
  mocks.userStore.batchGetOrFetchUsers.mockClear();
  mocks.userStore.getOrFetchUserByIdentifier.mockClear();
});

describe("PlanDetailApprovalFlow", () => {
  test("renders generating row for checking approval flow", () => {
    const issue = makeIssue({
      approvalStatus: ApprovalStatus.CHECKING,
    });
    const { container, render, unmount } = renderIntoContainer(
      <PlanDetailProvider value={makePageState({ issue })}>
        <PlanDetailReviewApprovalFlow />
      </PlanDetailProvider>
    );

    render();

    expect(container.textContent).toContain(
      "custom-approval.issue-review.generating-approval-flow"
    );
    expect(container.textContent).not.toContain("issue.approval-flow.self");
    expect(container.textContent).not.toContain("plan.view-discussion");

    unmount();
  });

  test("renders skip placeholder when approval flow is skipped", () => {
    const issue = makeIssue({
      approvalStatus: ApprovalStatus.SKIPPED,
      roles: [],
    });
    const { container, render, unmount } = renderIntoContainer(
      <PlanDetailProvider value={makePageState({ issue })}>
        <PlanDetailReviewApprovalFlow />
      </PlanDetailProvider>
    );

    render();

    expect(container.textContent).toContain(
      "custom-approval.approval-flow.skip"
    );
    expect(container.textContent).not.toContain("issue.approval-flow.self");

    unmount();
  });

  test("renders timeline header and footer link for pending approval", () => {
    const issue = makeIssue({
      approvalStatus: ApprovalStatus.PENDING,
      approvers: [
        {
          principal: "users/approver",
          status: Issue_Approver_Status.PENDING,
        },
      ],
    });
    const { container, render, unmount } = renderIntoContainer(
      <PlanDetailProvider value={makePageState({ issue })}>
        <PlanDetailReviewApprovalFlow />
      </PlanDetailProvider>
    );

    render();

    expect(container.textContent).toContain("issue.approval-flow.self");
    expect(container.textContent).toContain("common.under-review");
    expect(container.textContent).toContain(
      "common.issue #123 · plan.view-discussion"
    );

    const footerButton = [...container.querySelectorAll("button")].find(
      (button) =>
        button.textContent?.includes("common.issue #123 · plan.view-discussion")
    );
    expect(footerButton).toBeTruthy();

    act(() => {
      footerButton?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });

    expect(mocks.routerPush).toHaveBeenCalledWith(
      expect.objectContaining({
        params: {
          issueId: "123",
          projectId: "p1",
        },
      })
    );

    unmount();
  });

  test("renders rejection banner and re-request action when allowed", () => {
    mocks.comments = [
      {
        comment: "Needs more work",
        creator: "reviewer@example.com",
        event: { case: "approval" },
      },
    ];
    const issue = makeIssue({
      approvalStatus: ApprovalStatus.REJECTED,
      approvers: [
        {
          principal: "users/approver",
          status: Issue_Approver_Status.REJECTED,
        },
      ],
      creator: "users/me@example.com",
    });
    const { container, render, unmount } = renderIntoContainer(
      <PlanDetailProvider value={makePageState({ issue })}>
        <PlanDetailReviewApprovalFlow />
      </PlanDetailProvider>
    );

    render();

    expect(container.textContent).toContain(
      "custom-approval.issue-review.rejected-by reviewer@example.com"
    );
    expect(container.textContent).toContain("Needs more work");
    expect(container.textContent).toContain(
      "custom-approval.issue-review.re-request-review"
    );

    unmount();
  });
});
