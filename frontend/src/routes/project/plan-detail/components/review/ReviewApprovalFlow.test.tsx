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

// jsdom has no layout, so the real virtualizer renders no rows.
vi.mock("@tanstack/react-virtual", () => ({
  useVirtualizer: (options: { count: number }) => ({
    getVirtualItems: () =>
      Array.from({ length: options.count }, (_, index) => ({
        index,
        key: index,
        start: index * 40,
      })),
    getTotalSize: () => options.count * 40,
    measureElement: () => {},
  }),
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
  document.body.innerHTML = "";
});

const withCandidates = (count: number) => {
  mocks.projectPoliciesByName = { "projects/p1": {} };
  mocks.approvalCandidates = {
    ...NO_CANDIDATES,
    candidates: Array.from({ length: count }, (_, i) => ({
      canApprove: true,
      canReview: true,
      ineligibilities: [],
      user: {
        email: `person${i + 1}@example.com`,
        name: `users/person${i + 1}@example.com`,
        title: `Person ${i + 1}`,
      },
    })),
  } as unknown as ApprovalCandidates;
};

const reviewerTrigger = (container: HTMLElement) => {
  const trigger = [...container.querySelectorAll("button")].find((button) =>
    button.textContent?.includes("plan.review.approval-flow.n-reviewers")
  );
  if (!trigger) throw new Error("no reviewer trigger");
  return trigger;
};

describe("ReviewApprovalFlow reviewer list", () => {
  test("opens on click, not on hover, stays open when the pointer leaves, and lists every reviewer", async () => {
    withCandidates(120);
    const { container, render, unmount } = renderIntoContainer(
      <ReviewApprovalFlow issue={makeIssue()} />
    );
    document.body.appendChild(container);
    render();
    const trigger = reviewerTrigger(container);
    // Screen readers name the button by its count, not by the avatar initials.
    expect(trigger.querySelector('[aria-hidden="true"]')?.textContent).toBe(
      "PePePe+117"
    );

    await act(async () => {
      for (const type of ["pointerover", "pointerenter", "pointermove"]) {
        trigger.dispatchEvent(
          new PointerEvent(type, { bubbles: true, pointerType: "mouse" })
        );
      }
      for (const type of ["mouseover", "mouseenter", "mousemove"]) {
        trigger.dispatchEvent(new MouseEvent(type, { bubbles: true }));
      }
      await new Promise((resolve) => setTimeout(resolve, 300));
    });
    expect(document.body.querySelector('[role="list"]')).toBeNull();

    await act(async () => {
      trigger.click();
    });
    expect(trigger.getAttribute("aria-expanded")).toBe("true");
    expect(
      document.body.querySelector(
        'input[aria-label="plan.review.approval-flow.search-placeholder"]'
      )
    ).not.toBeNull();
    expect(document.body.querySelectorAll('[role="listitem"]')).toHaveLength(120);
    expect(document.body.textContent).not.toContain("and-n-more");

    await act(async () => {
      for (const type of ["pointerout", "pointerleave"]) {
        trigger.dispatchEvent(
          new PointerEvent(type, { bubbles: true, pointerType: "mouse" })
        );
      }
      for (const type of ["mouseout", "mouseleave"]) {
        trigger.dispatchEvent(new MouseEvent(type, { bubbles: true }));
      }
      await new Promise((resolve) => setTimeout(resolve, 300));
    });
    expect(trigger.getAttribute("aria-expanded")).toBe("true");

    unmount();
  });

  test.each([
    [12, "the search box"],
    [3, "the popup"],
  ])("with %i reviewers, opening moves focus to %s", async (count, target) => {
    withCandidates(count);
    const { container, render, unmount } = renderIntoContainer(
      <ReviewApprovalFlow issue={makeIssue()} />
    );
    document.body.appendChild(container);
    render();

    await act(async () => {
      reviewerTrigger(container).click();
      await new Promise((resolve) => setTimeout(resolve, 100));
    });

    const focused = document.activeElement as HTMLElement;
    if (target === "the search box") {
      expect(focused.getAttribute("aria-label")).toBe(
        "plan.review.approval-flow.search-placeholder"
      );
    } else {
      expect(focused.getAttribute("role")).toBe("dialog");
      expect(focused.querySelector('[role="list"]')).not.toBeNull();
    }

    unmount();
  });

  test("the overflow pill counts every reviewer past the third and widens to fit", () => {
    withCandidates(1200);
    const { container, render, unmount } = renderIntoContainer(
      <ReviewApprovalFlow issue={makeIssue()} />
    );
    render();

    const pill = [...container.querySelectorAll("span")].find(
      (span) => span.textContent === "+1197"
    );
    expect(pill).toBeDefined();
    expect(pill?.className).toContain("min-w-5");
    expect(pill?.className).not.toContain("size-5");

    unmount();
  });
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
