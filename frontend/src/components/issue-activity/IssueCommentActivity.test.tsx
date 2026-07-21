import { create } from "@bufbuild/protobuf";
import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import {
  Issue_Type,
  IssueComment_ReviewSubmissionSchema,
  IssueCommentSchema,
  IssueSchema,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import { PlanSchema } from "@/types/proto-es/v1/plan_service_pb";
import {
  getIssueCommentType,
  IssueCommentType,
} from "@/stores/app/issueComment";
import { IssueCommentRow } from "./IssueCommentActivity";

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({
    t: (key: string) =>
      key === "plan.review.activity.marked-ready-for-review"
        ? "marked this plan ready for review"
        : key,
  }),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({
      getUserByIdentifier: (identifier: string) =>
        identifier.includes("alice")
          ? {
              email: "alice@example.com",
              title: "Alice",
            }
          : {
              name: "users/submitter@example.com",
              email: "submitter@example.com",
              title: "Submitter",
            },
    }),
}));

vi.mock("@/components/HumanizeTs", () => ({
  HumanizeTs: () => null,
}));

vi.mock("@/components/monaco", () => ({
  ReadonlyDiffMonaco: () => null,
}));

vi.mock("@/components/UserAvatar", () => ({
  UserAvatar: () => <span data-testid="user-avatar" />,
}));

const reviewSubmission = create(IssueCommentSchema, {
  name: "projects/p1/issues/1/issueComments/1",
  creator: "users/submitter@example.com",
  event: {
    case: "reviewSubmission",
    value: create(IssueComment_ReviewSubmissionSchema),
  },
});

describe("IssueCommentRow", () => {
  test("shows who bypassed review and created the rollout", () => {
    const issue = create(IssueSchema, {
      type: Issue_Type.DATABASE_CHANGE,
    });
    const plan = create(PlanSchema, { hasRollout: true });
    const comment = create(IssueCommentSchema, {
      creator: "users/alice@example.com",
      event: {
        case: "issueUpdate",
        value: {
          fromStatus: IssueStatus.OPEN,
          toStatus: IssueStatus.DONE,
        },
      },
    });

    render(
      <IssueCommentRow
        comment={comment}
        isLast
        issue={issue}
        linkless
        plan={plan}
      />
    );

    expect(screen.getByText("Alice")).toBeInTheDocument();
    expect(
      screen.getByText("activity.sentence.review-skipped-rollout-created")
    ).toBeInTheDocument();
  });

  test("renders Review Submission as a meaningful Issue Detail activity", () => {
    expect(getIssueCommentType(reviewSubmission)).toBe(
      IssueCommentType.REVIEW_SUBMISSION
    );

    const { container } = render(
      <IssueCommentRow comment={reviewSubmission} isLast={true} />
    );

    expect(screen.getByText("Submitter")).toBeInTheDocument();
    expect(
      screen.getByText("marked this plan ready for review")
    ).toBeInTheDocument();
    expect(container.querySelectorAll(".lucide-send")).toHaveLength(1);
    expect(container.querySelector("[data-testid='user-avatar']")).toBeNull();
  });
});
