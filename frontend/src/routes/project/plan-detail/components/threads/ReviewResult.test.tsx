import { create } from "@bufbuild/protobuf";
import { cleanup, fireEvent, render } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import {
  IssueComment_ReviewMetadata_Priority,
  IssueComment_ReviewMetadataSchema,
  ReviewRun_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
} from "@/types/proto-es/v1/plan_service_pb";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, options?: { n?: number }) =>
      options?.n !== undefined ? `${key}:${options.n}` : key,
  }),
}));

vi.mock("@/hooks/useAppState", () => ({
  useUserByIdentifier: () => undefined,
}));

vi.mock("../PlanTargetDisplay", () => ({
  PlanTargetDisplay: ({ target }: { target: string }) => (
    <span data-testid="target">{target}</span>
  ),
}));

import {
  ReviewHeading,
  ReviewPriorityBadge,
  ReviewRule,
  ReviewTargets,
} from "./ReviewResult";

afterEach(cleanup);

const targetsOf = (container: HTMLElement) =>
  Array.from(container.querySelectorAll("[data-testid='target']")).map(
    (node) => node.textContent
  );

const databases = (count: number) =>
  Array.from({ length: count }, (_, i) => `instances/i/databases/db${i}`);

const changeSpec = (
  targets: string[],
  source: { sheet: string } | { release: string } = {
    sheet: "projects/p/sheets/abc",
  }
) =>
  create(Plan_SpecSchema, {
    id: "spec-1",
    config: {
      case: "changeDatabaseConfig",
      value: create(Plan_ChangeDatabaseConfigSchema, { targets, ...source }),
    },
  });

describe("ReviewPriorityBadge", () => {
  test.each([
    [IssueComment_ReviewMetadata_Priority.P0, "P0", null],
    [IssueComment_ReviewMetadata_Priority.P1, "P1", "plan.review.thread.priority.p1"],
    [IssueComment_ReviewMetadata_Priority.P2, "P2", "plan.review.thread.priority.p2"],
  ])("labels priority %s as %s, explained by %s", (priority, label, description) => {
    const { getByTestId } = render(<ReviewPriorityBadge priority={priority} />);
    expect(getByTestId("review-priority").textContent).toBe(label);
    expect(getByTestId("review-priority").getAttribute("title")).toBe(description);
  });

  test("shows nothing for an unspecified priority", () => {
    const { container } = render(
      <ReviewPriorityBadge
        priority={IssueComment_ReviewMetadata_Priority.PRIORITY_UNSPECIFIED}
      />
    );
    expect(container.innerHTML).toBe("");
  });
});

describe("ReviewRule", () => {
  test("names the standard rule in the reader's language", () => {
    const { getByTestId } = render(
      <ReviewRule ruleType={ReviewRuleType.REQUIRE_WHERE} />
    );
    expect(getByTestId("review-rule").textContent).toBe(
      "sql-review.standard-rules.rule.require-where.title"
    );
  });

  test("shows nothing for an unspecified rule", () => {
    const { container } = render(
      <ReviewRule ruleType={ReviewRuleType.REVIEW_RULE_TYPE_UNSPECIFIED} />
    );
    expect(container).toBeEmptyDOMElement();
  });
});

describe("ReviewHeading", () => {
  const codesOf = (container: HTMLElement) =>
    Array.from(container.querySelectorAll("[data-testid='review-code']")).map(
      (node) => node.textContent
    );

  test("puts a rule result's priority, rule, and codes on one line", () => {
    const { getByTestId } = render(
      <ReviewHeading
        review={create(IssueComment_ReviewMetadataSchema, {
          runType: ReviewRun_Type.RULE,
          ruleType: ReviewRuleType.PRIOR_BACKUP,
          priority: IssueComment_ReviewMetadata_Priority.P0,
          codes: ["PB103", "PB102"],
        })}
      />
    );
    const heading = getByTestId("review-heading");
    expect(
      Array.from(heading.children).map((node) => node.getAttribute("data-testid"))
    ).toEqual(["review-priority", "review-rule", "review-code", "review-code"]);
    expect(codesOf(heading)).toEqual(["PB103", "PB102"]);
  });

  test("shows a rule result without codes by its priority and rule", () => {
    const { container, getByTestId } = render(
      <ReviewHeading
        review={create(IssueComment_ReviewMetadataSchema, {
          runType: ReviewRun_Type.RULE,
          ruleType: ReviewRuleType.REQUIRE_WHERE,
          priority: IssueComment_ReviewMetadata_Priority.P1,
        })}
      />
    );
    expect(getByTestId("review-rule")).toBeInTheDocument();
    expect(codesOf(container)).toEqual([]);
  });

  test("shows an AI result by its priority and title", () => {
    const { container, getByTestId } = render(
      <ReviewHeading
        review={create(IssueComment_ReviewMetadataSchema, {
          runType: ReviewRun_Type.AI,
          priority: IssueComment_ReviewMetadata_Priority.P2,
          codes: ["WT101"],
          title: "Create `idx_rental` *concurrently*",
        })}
      />
    );
    const heading = getByTestId("review-heading");
    expect(
      Array.from(heading.children).map((node) => node.getAttribute("data-testid"))
    ).toEqual(["review-priority", "review-title"]);
    expect(getByTestId("review-priority").textContent).toBe("P2");
    expect(getByTestId("review-title").textContent).toBe(
      "Create idx_rental concurrently"
    );
    expect(container.querySelector("[data-testid='review-rule']")).toBeNull();
    expect(codesOf(container)).toEqual([]);
  });

  test("shows a rule result without a title", () => {
    const { container } = render(
      <ReviewHeading
        review={create(IssueComment_ReviewMetadataSchema, {
          runType: ReviewRun_Type.RULE,
          ruleType: ReviewRuleType.REQUIRE_WHERE,
          priority: IssueComment_ReviewMetadata_Priority.P1,
          title: "ignored",
        })}
      />
    );
    expect(container.querySelector("[data-testid='review-title']")).toBeNull();
  });
});

describe("ReviewTargets", () => {
  test("shows a short list whole", () => {
    const { container, queryByRole } = render(
      <ReviewTargets spec={undefined} targets={databases(4)} />
    );
    expect(targetsOf(container)).toEqual(databases(4));
    expect(queryByRole("button")).toBeNull();
  });

  test("folds a long list behind a count until expanded", () => {
    const { container, getByRole } = render(
      <ReviewTargets spec={undefined} targets={databases(6)} />
    );
    expect(targetsOf(container)).toEqual(databases(3));
    fireEvent.click(getByRole("button", { name: "common.n-more:3" }));
    expect(targetsOf(container)).toEqual(databases(6));
  });

  test("shows nothing without targets", () => {
    const { container } = render(
      <ReviewTargets spec={undefined} targets={[]} />
    );
    expect(container.innerHTML).toBe("");
  });

  test("hides the row when the targets equal the spec's databases in another order", () => {
    const { container } = render(
      <ReviewTargets
        spec={changeSpec([...databases(3)].reverse())}
        targets={databases(3)}
      />
    );
    expect(container.innerHTML).toBe("");
  });

  test("shows the row when the targets are a strict subset of the spec's databases", () => {
    const { container } = render(
      <ReviewTargets spec={changeSpec(databases(3))} targets={databases(2)} />
    );
    expect(targetsOf(container)).toEqual(databases(2));
  });

  test("shows the row when the spec is unknown", () => {
    const { container } = render(
      <ReviewTargets spec={undefined} targets={databases(2)} />
    );
    expect(targetsOf(container)).toEqual(databases(2));
  });

  test("shows the row when the spec targets a database group", () => {
    const { container } = render(
      <ReviewTargets
        spec={changeSpec(["projects/p/databaseGroups/all"])}
        targets={databases(2)}
      />
    );
    expect(targetsOf(container)).toEqual(databases(2));
  });

  test("hides the row when project-instance targets equal the spec's as a set", () => {
    const projectDatabases = [
      "projects/p/instances/i/databases/app",
      "projects/p/instances/i/databases/billing",
    ];
    const { container } = render(
      <ReviewTargets
        spec={changeSpec([...projectDatabases].reverse())}
        targets={projectDatabases}
      />
    );
    expect(container.innerHTML).toBe("");
  });

  test("shows the row when the spec names a project-instance database and the result a workspace-instance one", () => {
    const { container } = render(
      <ReviewTargets
        spec={changeSpec(["projects/p/instances/i/databases/d"])}
        targets={["instances/i/databases/d"]}
      />
    );
    expect(targetsOf(container)).toEqual(["instances/i/databases/d"]);
  });

  test("shows the row when a target carries a trailing segment", () => {
    const withSchema = ["instances/i/databases/d/schemas/public"];
    const { container } = render(
      <ReviewTargets spec={changeSpec(withSchema)} targets={withSchema} />
    );
    expect(targetsOf(container)).toEqual(withSchema);
  });

  test("shows the row when the spec is release-backed", () => {
    const { container } = render(
      <ReviewTargets
        spec={changeSpec(databases(2), { release: "projects/p/releases/r1" })}
        targets={databases(2)}
      />
    );
    expect(targetsOf(container)).toEqual(databases(2));
  });
});
