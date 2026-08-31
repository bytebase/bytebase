import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

// ---------------------------------------------------------------------------
// Module mocks. The skipped/empty path under test only needs `useTranslation`;
// the heavy store/context/candidate hooks fire solely when flow nodes render
// (the non-skipped test below), so stub them to keep the graph isolated.
// ---------------------------------------------------------------------------

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params && Object.keys(params).length > 0
        ? `${key}:${JSON.stringify(params)}`
        : key,
  }),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({
      roleList: [],
      getUserByIdentifier: () => undefined,
      projectPoliciesByName: {},
    }),
}));

const mocks: { approvalCandidates: ApprovalCandidates } = {
  approvalCandidates: {
    canCurrentUserApprove: false,
    candidates: [],
    isCurrentUserReviewCandidate: false,
    isCurrentUserRoleMember: false,
  } as ApprovalCandidates,
};

vi.mock("./useApprovalCandidates", () => ({
  useApprovalCandidates: () => mocks.approvalCandidates,
}));

vi.mock("../../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => ({ projectId: "p1" }),
}));

vi.mock("@/components/UserAvatar", () => ({
  getAvatarColor: () => "#000",
  getInitials: (name: string) => name.slice(0, 2),
}));

vi.mock("@/stores/modules/v1/common", () => ({
  projectNamePrefix: "projects/",
}));

vi.mock("@/types", () => ({
  unknownUser: (principal: string) => ({
    name: principal,
    email: principal,
    title: principal,
  }),
}));

vi.mock("@/lib/role", () => ({
  displayRoleTitleFromList: (role: string) => role,
}));

import { ReviewApprovalFlow } from "./ReviewApprovalFlow";
import type { ApprovalCandidates } from "./useApprovalCandidates";

const makeIssue = ({
  approvalStatus = ApprovalStatus.PENDING,
  roles = ["roles/PROJECT_OWNER"],
}: {
  approvalStatus?: ApprovalStatus;
  roles?: string[];
} = {}): Issue =>
  ({
    name: "projects/p1/issues/123",
    approvalStatus,
    approvers: [],
    approvalTemplate: { flow: { roles }, title: "Test policy" },
    riskLevel: 0,
  }) as unknown as Issue;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  return {
    container,
    render: () => act(() => root.render(element)),
    unmount: () => act(() => root.unmount()),
  };
};

describe("ReviewApprovalFlow", () => {
  test("keeps the last plan editor visible with an ineligible reason", () => {
    mocks.approvalCandidates = {
      canCurrentUserApprove: false,
      candidates: [
        {
          canApprove: false,
          canReview: true,
          ineligibilities: ["last-plan-editor"],
          user: {
            email: "editor@example.com",
            name: "users/editor@example.com",
            title: "Editor",
          },
        },
      ],
      isCurrentUserReviewCandidate: true,
      isCurrentUserRoleMember: true,
    } as unknown as ApprovalCandidates;
    const { container, render, unmount } = renderIntoContainer(
      <ReviewApprovalFlow issue={makeIssue()} />
    );

    render();

    expect(container.textContent).toContain("Editor");
    expect(container.textContent).toContain(
      "plan.review.last-plan-editor-cannot-approve"
    );

    mocks.approvalCandidates = {
      canCurrentUserApprove: false,
      candidates: [],
      isCurrentUserReviewCandidate: false,
      isCurrentUserRoleMember: false,
    } as ApprovalCandidates;
    unmount();
  });

  // BYT-9745: a skipped approval has no flow to render. The component must show
  // the "no approval required" note instead of an empty box — otherwise callers
  // that forget to guard the skipped case (e.g. the bypass confirm sheet) show
  // an empty bordered box.
  test("SKIPPED approval renders the no-approval-required note", () => {
    const issue = makeIssue({
      approvalStatus: ApprovalStatus.SKIPPED,
      roles: [],
    });
    const { container, render, unmount } = renderIntoContainer(
      <ReviewApprovalFlow issue={issue} />
    );

    render();

    expect(container.textContent).toContain(
      "custom-approval.approval-flow.skip"
    );

    unmount();
  });

  test("template with no approver roles renders the note (not an empty box)", () => {
    const issue = makeIssue({
      approvalStatus: ApprovalStatus.APPROVED,
      roles: [],
    });
    const { container, render, unmount } = renderIntoContainer(
      <ReviewApprovalFlow issue={issue} />
    );

    render();

    expect(container.textContent).toContain(
      "custom-approval.approval-flow.skip"
    );

    unmount();
  });

  test("a pending flow with roles renders the approval node, not the skip note", () => {
    const issue = makeIssue({
      approvalStatus: ApprovalStatus.PENDING,
      roles: ["roles/PROJECT_OWNER"],
    });
    const { container, render, unmount } = renderIntoContainer(
      <ReviewApprovalFlow issue={issue} />
    );

    render();

    expect(container.textContent).toContain("roles/PROJECT_OWNER");
    expect(container.textContent).not.toContain(
      "custom-approval.approval-flow.skip"
    );

    unmount();
  });
});
