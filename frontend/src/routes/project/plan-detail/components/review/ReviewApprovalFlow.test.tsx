import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import {
  type Issue,
  Issue_Approver_Status,
} from "@/types/proto-es/v1/issue_service_pb";

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
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

const NO_CANDIDATES = {
  canCurrentUserApprove: false,
  candidates: [],
  isCurrentUserReviewCandidate: false,
  isCurrentUserRoleMember: false,
} as ApprovalCandidates;

const mocks: {
  approvalCandidates: ApprovalCandidates;
  projectPoliciesByName: Record<string, unknown>;
  // What usePrincipal resolves, by `users/{email}`.
  principals: Record<string, { email: string; title: string }>;
} = {
  approvalCandidates: NO_CANDIDATES,
  projectPoliciesByName: {},
  principals: {},
};

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({
      roleList: [],
      projectPoliciesByName: mocks.projectPoliciesByName,
    }),
}));

vi.mock("@/hooks/usePrincipal", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/hooks/usePrincipal")>()),
  usePrincipal: (identifier?: string) =>
    identifier ? mocks.principals[identifier] : undefined,
}));

vi.mock("../../../useApprovalCandidates", () => ({
  useApprovalCandidates: () => mocks.approvalCandidates,
}));

vi.mock("../../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => ({ projectId: "p1" }),
}));

vi.mock("@/components/UserAvatar", () => ({
  getAvatarColor: () => "#000",
  getInitials: (name: string) => name.slice(0, 2),
}));

vi.mock("@/lib/role", () => ({
  displayRoleTitleFromList: (role: string) => role,
}));

import { ReviewApprovalFlow } from "./ReviewApprovalFlow";
import type { ApprovalCandidates } from "../../../useApprovalCandidates";

const makeIssue = ({
  approvalStatus = ApprovalStatus.PENDING,
  approvers = [],
  roles = ["roles/PROJECT_OWNER"],
}: {
  approvalStatus?: ApprovalStatus;
  approvers?: { principal: string; status: Issue_Approver_Status }[];
  roles?: string[];
} = {}): Issue =>
  ({
    name: "projects/p1/issues/123",
    approvalStatus,
    approvers,
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

afterEach(() => {
  mocks.approvalCandidates = NO_CANDIDATES;
  mocks.projectPoliciesByName = {};
  mocks.principals = {};
});

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
      "plan.review.candidate-last-changed-plan"
    );

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

  test("a step whose role only a service account holds lists it as the reviewer", () => {
    mocks.projectPoliciesByName = { "projects/p1": {} };
    mocks.approvalCandidates = {
      ...NO_CANDIDATES,
      candidates: [
        {
          canApprove: true,
          canReview: true,
          ineligibilities: [],
          user: {
            email: "deploy@service.bytebase.com",
            name: "users/deploy@service.bytebase.com",
            title: "Deploy bot",
          },
        },
      ],
    } as unknown as ApprovalCandidates;
    const { container, render, unmount } = renderIntoContainer(
      <ReviewApprovalFlow issue={makeIssue()} />
    );

    render();

    expect(container.textContent).toContain(
      'plan.review.approval-flow.n-reviewers:{"count":1}'
    );
    expect(container.textContent).not.toContain(
      "plan.review.approval-flow.no-reviewers"
    );

    unmount();
  });

  test("a step nobody holds the role for says so once the policy has loaded", () => {
    mocks.projectPoliciesByName = { "projects/p1": {} };
    const { container, render, unmount } = renderIntoContainer(
      <ReviewApprovalFlow issue={makeIssue()} />
    );

    render();

    expect(container.textContent).toContain(
      "plan.review.approval-flow.no-reviewers"
    );

    unmount();
  });

  test.each([
    [Issue_Approver_Status.APPROVED, "plan.review.approval-flow.approved-by"],
    [Issue_Approver_Status.REJECTED, "plan.review.approval-flow.rejected-by"],
  ])("a step decided by a service account names it and marks its account type (%s)", (status, sentence) => {
    mocks.principals["users/deploy@service.bytebase.com"] = {
      email: "deploy@service.bytebase.com",
      title: "Deploy bot",
    };
    const { container, render, unmount } = renderIntoContainer(
      <ReviewApprovalFlow
        issue={makeIssue({
          approvers: [
            { principal: "users/deploy@service.bytebase.com", status },
          ],
        })}
      />
    );

    render();

    expect(container.textContent).toContain(`${sentence}:{"user":"Deploy bot"}`);
    expect(container.textContent).toContain("settings.members.service-account");

    unmount();
  });

  test("a step approved by a person whose record has not arrived falls back to the email name, unmarked", () => {
    mocks.principals["users/alex@example.com"] = {
      email: "alex@example.com",
      title: "",
    };
    const { container, render, unmount } = renderIntoContainer(
      <ReviewApprovalFlow
        issue={makeIssue({
          approvers: [
            {
              principal: "users/alex@example.com",
              status: Issue_Approver_Status.APPROVED,
            },
          ],
        })}
      />
    );

    render();

    expect(container.textContent).toContain(
      'plan.review.approval-flow.approved-by:{"user":"alex"}'
    );
    expect(container.textContent).not.toContain("settings.members.");

    unmount();
  });
});
