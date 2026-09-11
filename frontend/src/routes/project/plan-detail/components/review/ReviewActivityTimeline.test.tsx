import { create } from "@bufbuild/protobuf";
import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import {
  IssueComment_ReviewSubmissionSchema,
  IssueComment_ThreadState,
  IssueCommentSchema,
  IssueSchema,
  StatementAnchorSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { PlanSchema } from "@/types/proto-es/v1/plan_service_pb";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string) =>
      key === "plan.review.activity.marked-ready-for-review"
        ? "marked this plan ready for review"
        : key,
  }),
}));

vi.mock("@/components/issue-activity/IssueCommentActivity", () => ({
  ActivityRowFrame: ({
    children,
    icon,
  }: {
    children: ReactNode;
    icon: ReactNode;
  }) => (
    <li data-testid="thread-row">
      {icon}
      {children}
    </li>
  ),
  ActivityUserIcon: () => <span data-testid="user-icon" />,
  ActivityRowShell: ({
    header,
    icon,
  }: {
    header: ReactNode;
    icon: ReactNode;
  }) => (
    <li>
      {icon}
      {header}
    </li>
  ),
  CommentCreator: ({ creator }: { creator: { title: string } }) => (
    <span>{creator.title}</span>
  ),
  CommentIconBadge: ({ icon }: { icon: ReactNode }) => <span>{icon}</span>,
  canEditIssueComment: () => false,
  IssueCommentRow: () => <li data-testid="comment-row" />,
  ReviewSubmissionIcon: () => <span data-testid="review-submission-icon" />,
  ReviewSubmissionSentence: () => (
    <span>marked this plan ready for review</span>
  ),
}));

vi.mock("@/components/HumanizeTs", () => ({
  HumanizeTs: () => null,
}));

vi.mock("@/components/MarkdownEditor", () => ({
  MarkdownEditor: () => null,
}));

vi.mock("@/hooks/useAppState", () => ({
  useCurrentUser: () => ({ email: "me@example.com" }),
  useUserByIdentifier: () => ({
    name: "users/submitter@example.com",
    email: "submitter@example.com",
    title: "Submitter",
  }),
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () => undefined,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: unknown) => unknown) =>
      selector({
        getUserByIdentifier: () => ({
          name: "users/submitter@example.com",
          email: "submitter@example.com",
          title: "Submitter",
        }),
        sheetsByName: {},
      }),
    { getState: () => ({ getOrFetchSheetByName: async () => undefined }) }
  ),
}));

vi.mock("@/app/router", () => ({
  router: { push: vi.fn() },
}));

vi.mock("../../shared/stores/usePlanDetailStore", () => ({
  usePlanDetailStore: (selector: (state: unknown) => unknown) =>
    selector({
      selectedSpecId: undefined,
      placements: new Map(),
      placementTargets: new Map(),
    }),
  usePlanDetailStoreApi: () => ({
    getState: () => ({ requestThreadFocus: vi.fn() }),
  }),
}));

vi.mock("../threads/CommentThreadCard", () => ({
  CommentThreadCard: ({
    context,
    thread,
  }: {
    context: ReactNode;
    thread: { root: { name: string }; replies: { name: string }[] };
  }) => (
    <div
      data-replies={thread.replies.length}
      data-root={thread.root.name}
      data-testid="thread-card"
    >
      {context}
    </div>
  ),
}));

const gateMocks = vi.hoisted(() => ({
  inlineThreadsEnabled: vi.fn(() => true),
}));
vi.mock("@/utils/featureGates", () => gateMocks);

vi.mock("../threads/StatementAnchorContext", () => ({
  StatementAnchorContext: () => <div data-testid="anchor-context" />,
}));

vi.mock("@/stores", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/types", () => ({
  getTimeForPbTimestampProtoEs: () => 0,
  unknownUser: (principal: string) => ({
    name: principal,
    email: principal,
    title: principal,
  }),
}));

vi.mock("@/utils/iam/permission", () => ({
  hasProjectPermissionV2: () => false,
}));

vi.mock("../../hooks/usePlanChangeReferenceData", () => ({
  usePlanChangeReferenceData: () => ({
    databaseGroupsByName: {},
    databasesByName: {},
    environmentList: [],
  }),
}));

vi.mock("../../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => ({ projectId: "p1" }),
}));

vi.mock("../PlanChangeReference", () => ({
  PlanSpecChangeReference: () => <span data-testid="change-reference" />,
}));

vi.mock("./ReviewCommentComposer", () => ({
  ReviewCommentComposer: () => null,
}));

import { ReviewActivityTimeline } from "./ReviewActivityTimeline";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const reviewSubmission = (name: string) =>
  create(IssueCommentSchema, {
    name,
    creator: "users/submitter@example.com",
    event: {
      case: "reviewSubmission",
      value: create(IssueComment_ReviewSubmissionSchema),
    },
  });

describe("ReviewActivityTimeline", () => {
  test("renders one persisted Review Submission instead of a fallback duplicate", () => {
    const issue = create(IssueSchema, {
      name: "projects/p1/issues/1",
      creator: "users/issue-creator@example.com",
      draft: false,
    });
    const plan = create(PlanSchema, { name: "projects/p1/plans/1" });
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(
        <ReviewActivityTimeline
          comments={[
            reviewSubmission("comments/submission"),
            reviewSubmission("comments/duplicate"),
          ]}
          issue={issue}
          plan={plan}
        />
      );
    });

    expect(
      container.textContent?.match(/marked this plan ready for review/g)
    ).toHaveLength(1);
    expect(
      container.querySelectorAll("[data-testid='review-submission-icon']")
    ).toHaveLength(1);
    expect(container.querySelectorAll("li")).toHaveLength(1);
    expect(container.querySelector("[data-testid='comment-row']")).toBeNull();

    act(() => root.unmount());
  });

  test("a thread root renders one card with its replies and never a reply row", () => {
    const issue = create(IssueSchema, {
      name: "projects/p1/issues/1",
      creator: "users/issue-creator@example.com",
      draft: false,
    });
    const plan = create(PlanSchema, { name: "projects/p1/plans/1" });
    const rootName = "projects/p1/issues/1/issueComments/root";
    const comments = [
      reviewSubmission("comments/submission"),
      create(IssueCommentSchema, {
        name: rootName,
        comment: "Root",
        creator: "users/submitter@example.com",
        threadState: IssueComment_ThreadState.OPEN,
        statementAnchor: create(StatementAnchorSchema, { spec: "spec-1" }),
      }),
      create(IssueCommentSchema, {
        name: "projects/p1/issues/1/issueComments/reply",
        comment: "Reply",
        creator: "users/submitter@example.com",
        root: rootName,
      }),
      create(IssueCommentSchema, {
        name: "projects/p1/issues/1/issueComments/general",
        comment: "General",
        creator: "users/submitter@example.com",
      }),
    ];
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(
        <ReviewActivityTimeline comments={comments} issue={issue} plan={plan} />
      );
    });

    const card = container.querySelector("[data-testid='thread-card']");
    expect(card?.getAttribute("data-root")).toBe(rootName);
    expect(card?.getAttribute("data-replies")).toBe("1");
    expect(card?.querySelector("[data-testid='anchor-context']")).not.toBeNull();
    // submission row + thread row + general comment row
    expect(container.querySelectorAll("li")).toHaveLength(3);
    expect(
      container.querySelectorAll("[data-testid='comment-row']")
    ).toHaveLength(1);

    act(() => root.unmount());
  });

  test("a release build renders a thread root as an ordinary comment", () => {
    gateMocks.inlineThreadsEnabled.mockReturnValue(false);
    const issue = create(IssueSchema, {
      name: "projects/p1/issues/1",
      creator: "users/issue-creator@example.com",
      draft: false,
    });
    const plan = create(PlanSchema, { name: "projects/p1/plans/1" });
    const rootName = "projects/p1/issues/1/issueComments/root";
    const comments = [
      create(IssueCommentSchema, {
        name: rootName,
        comment: "Root",
        creator: "users/submitter@example.com",
        threadState: IssueComment_ThreadState.OPEN,
        statementAnchor: create(StatementAnchorSchema, { spec: "spec-1" }),
      }),
      create(IssueCommentSchema, {
        name: "projects/p1/issues/1/issueComments/general",
        comment: "General",
        creator: "users/submitter@example.com",
      }),
    ];
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(
        <ReviewActivityTimeline comments={comments} issue={issue} plan={plan} />
      );
    });

    // The root keeps its stored thread state; with the gate off it must still
    // reach the reader, as a plain comment rather than a card.
    expect(container.querySelector("[data-testid='thread-card']")).toBeNull();
    expect(
      container.querySelectorAll("[data-testid='comment-row']")
    ).toHaveLength(2);

    act(() => root.unmount());
    gateMocks.inlineThreadsEnabled.mockReturnValue(true);
  });
});
