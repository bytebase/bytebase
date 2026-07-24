import { create } from "@bufbuild/protobuf";
import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import {
  IssueComment_PlanUpdateSchema,
  Issue_Type,
  IssueComment_ReviewSubmissionSchema,
  IssueCommentSchema,
  IssueSchema,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  getIssueCommentType,
  IssueCommentType,
} from "@/stores/app/issueComment";
import { IssueCommentRow } from "./IssueCommentActivity";

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({
    t: (key: string) => {
      if (key === "plan.review.activity.marked-ready-for-review") {
        return "marked this plan ready for review";
      }
      if (key === "plan.spec.change") {
        return "Change";
      }
      return key;
    },
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

const changeSpec = ({
  id = "spec-1",
  sheet,
  target,
}: {
  id?: string;
  sheet: string;
  target: string;
}) =>
  create(Plan_SpecSchema, {
    id,
    config: {
      case: "changeDatabaseConfig",
      value: create(Plan_ChangeDatabaseConfigSchema, {
        sheet,
        targets: [`instances/instance-1/databases/${target}`],
      }),
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

  test("renders a computed change reference for a SQL update", () => {
    const fromSpec = changeSpec({
      sheet: "sheets/1",
      target: "orders",
    });
    const toSpec = changeSpec({
      sheet: "sheets/2",
      target: "orders",
    });
    const comment = create(IssueCommentSchema, {
      creator: "users/alice@example.com",
      event: {
        case: "planUpdate",
        value: create(IssueComment_PlanUpdateSchema, {
          fromSpecs: [fromSpec],
          toSpecs: [toSpec],
        }),
      },
    });

    render(
      <IssueCommentRow
        comment={comment}
        isLast
        plan={create(PlanSchema, { specs: [toSpec] })}
        renderPlanChangeReference={({ siblings, spec }) => (
          <span data-testid="change-reference">
            {siblings.indexOf(spec) + 1}{" "}
            {spec.config.case === "changeDatabaseConfig"
              ? spec.config.value.targets[0].split("/").at(-1)
              : ""}
          </span>
        )}
      />
    );

    expect(
      screen.getByText("activity.sentence.modified-sql-of")
    ).toBeInTheDocument();
    expect(screen.getByTestId("change-reference")).toHaveTextContent(
      "1 orders"
    );
    expect(screen.getByText("Change")).toHaveAttribute("aria-hidden", "true");
  });

  test("uses the new target in a changed-targets reference", () => {
    const fromSpec = changeSpec({
      sheet: "sheets/1",
      target: "orders",
    });
    const toSpec = changeSpec({
      sheet: "sheets/1",
      target: "employees",
    });
    const comment = create(IssueCommentSchema, {
      creator: "users/alice@example.com",
      event: {
        case: "planUpdate",
        value: create(IssueComment_PlanUpdateSchema, {
          fromSpecs: [fromSpec],
          toSpecs: [toSpec],
        }),
      },
    });

    render(
      <IssueCommentRow
        comment={comment}
        isLast
        plan={create(PlanSchema, { specs: [toSpec] })}
        renderPlanChangeReference={({ spec }) => (
          <span data-testid="change-reference">
            {spec.config.case === "changeDatabaseConfig"
              ? spec.config.value.targets[0].split("/").at(-1)
              : ""}
          </span>
        )}
      />
    );

    expect(
      screen.getByText("activity.sentence.changed-targets-of")
    ).toBeInTheDocument();
    expect(screen.getByTestId("change-reference")).toHaveTextContent(
      "employees"
    );
  });

  test("uses the historical index and target for a removed change", () => {
    const remainingSpec = changeSpec({
      sheet: "sheets/1",
      target: "orders",
    });
    const removedSpec = changeSpec({
      id: "spec-2",
      sheet: "sheets/2",
      target: "employees",
    });
    const comment = create(IssueCommentSchema, {
      creator: "users/alice@example.com",
      event: {
        case: "planUpdate",
        value: create(IssueComment_PlanUpdateSchema, {
          fromSpecs: [remainingSpec, removedSpec],
          toSpecs: [remainingSpec],
        }),
      },
    });

    render(
      <IssueCommentRow
        comment={comment}
        isLast
        plan={create(PlanSchema, { specs: [remainingSpec] })}
        renderPlanChangeReference={({ siblings, spec }) => (
          <span data-testid="change-reference">
            {siblings.indexOf(spec) + 1}{" "}
            {spec.config.case === "changeDatabaseConfig"
              ? spec.config.value.targets[0].split("/").at(-1)
              : ""}
          </span>
        )}
      />
    );

    expect(screen.getByText("activity.sentence.removed-spec")).toBeVisible();
    expect(screen.getByTestId("change-reference")).toHaveTextContent(
      "2 employees"
    );
  });
});
