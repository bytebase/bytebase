import { create } from "@bufbuild/protobuf";
import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import {
  Issue_Type,
  IssueCommentSchema,
  IssueSchema,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import { PlanSchema } from "@/types/proto-es/v1/plan_service_pb";
import { IssueCommentRow } from "./IssueCommentActivity";

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({
      getUserByIdentifier: () => ({
        email: "alice@example.com",
        title: "Alice",
      }),
    }),
}));

vi.mock("@/react/components/monaco", () => ({
  ReadonlyDiffMonaco: () => null,
}));

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
});
