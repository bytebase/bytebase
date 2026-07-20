import { create } from "@bufbuild/protobuf";
import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import {
  IssueComment_ReviewSubmissionSchema,
  IssueCommentSchema,
  IssueSchema,
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

vi.mock("@/react/components/issue-activity/IssueCommentActivity", () => ({
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

vi.mock("@/react/components/HumanizeTs", () => ({
  HumanizeTs: () => null,
}));

vi.mock("@/react/components/MarkdownEditor", () => ({
  MarkdownEditor: () => null,
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useCurrentUser: () => ({ email: "me@example.com" }),
}));

vi.mock("@/react/hooks/useProjectByName", () => ({
  useProjectByName: () => undefined,
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: unknown) => unknown) =>
      selector({
        getUserByIdentifier: () => ({
          name: "users/submitter@example.com",
          email: "submitter@example.com",
          title: "Submitter",
        }),
      }),
    { getState: () => ({}) }
  ),
}));

vi.mock("@/store", () => ({
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

vi.mock("../../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => ({ projectId: "p1" }),
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
});
