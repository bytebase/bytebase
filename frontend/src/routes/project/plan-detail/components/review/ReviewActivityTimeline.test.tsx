import { create } from "@bufbuild/protobuf";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { shownTimestampModes } from "@/test-utils/humanizeTs";
import { PositionSchema } from "@/types/proto-es/v1/common_pb";
import {
  IssueComment_ReviewMetadataSchema,
  IssueComment_ReviewRunCompletionSchema,
  IssueComment_ReviewSubmissionSchema,
  IssueComment_ThreadState,
  IssueCommentSchema,
  IssueSchema,
  StatementAnchorSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";

const mocks = vi.hoisted(() => ({
  activityFocus: undefined as { commentName: string; nonce: number } | undefined,
  clearActivityFocus: vi.fn(),
  requestThreadFocus: vi.fn(),
  expandPhase: vi.fn(),
  placements: new Map<string, unknown>(),
  placementTargets: new Map<string, string>(),
}));

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
    id,
  }: {
    children: ReactNode;
    icon: ReactNode;
    id?: string;
  }) => (
    <li data-testid="thread-row" id={id}>
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

vi.mock("@/components/HumanizeTs", async () => ({
  ...(await import("@/test-utils/humanizeTs")).humanizeTsStub(),
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

vi.mock("@/hooks/usePrincipal", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/hooks/usePrincipal")>()),
  usePrincipal: (identifier?: string) =>
    identifier
      ? { email: "submitter@example.com", title: "Submitter" }
      : undefined,
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
      placements: mocks.placements,
      placementTargets: mocks.placementTargets,
      activityFocus: mocks.activityFocus,
      clearActivityFocus: mocks.clearActivityFocus,
    }),
  usePlanDetailStoreApi: () => ({
    getState: () => ({
      requestThreadFocus: mocks.requestThreadFocus,
    }),
  }),
}));

vi.mock("../threads/CommentThreadCard", () => ({
  CommentThreadCard: ({
    bar,
    excerpt,
    spec,
    thread,
  }: {
    bar: { foldLabel: string };
    excerpt?: ReactNode;
    spec?: { id: string };
    thread: { root: { name: string }; replies: { name: string }[] };
  }) => (
    <div
      data-fold-label={bar.foldLabel}
      data-replies={thread.replies.length}
      data-root={thread.root.name}
      data-spec={spec?.id}
      data-testid="thread-card"
    >
      {excerpt}
    </div>
  ),
}));

vi.mock("../threads/StatementAnchorContext", () => ({
  StatementAnchorContext: ({
    anchor,
    children,
    onViewInStatement,
  }: {
    anchor?: unknown;
    children: (anchored: { bar: { foldLabel: string }; excerpt?: ReactNode }) => ReactNode;
    onViewInStatement: () => void;
  }) =>
    children({
      bar: { foldLabel: anchor ? "on lines" : "on the plan" },
      excerpt: anchor ? (
        <button data-testid="anchor-context" onClick={onViewInStatement}>
          View in Statement
        </button>
      ) : undefined,
    }),
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
  usePlanDetailContext: () => ({
    projectId: "p1",
    planId: "1",
    plan: { specs: [{ id: "spec-1" }] },
    expandPhase: mocks.expandPhase,
  }),
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
  afterEach(() => {
    vi.clearAllMocks();
    mocks.activityFocus = undefined;
    mocks.placements.clear();
    mocks.placementTargets.clear();
  });

  test.each([
    { mapped: false, expectedLine: 12 },
    { mapped: true, expectedLine: 20 },
  ])("passes the $expectedLine saved statement line to the editor", ({ mapped, expectedLine }) => {
    const currentSha = "c".repeat(64);
    const commentName = "projects/p1/issues/1/issueComments/root";
    const plan = create(PlanSchema, {
      name: "projects/p1/plans/1",
      specs: [
        create(Plan_SpecSchema, {
          id: "spec-1",
          config: {
            case: "changeDatabaseConfig",
            value: create(Plan_ChangeDatabaseConfigSchema, {
              sheet: `projects/p1/sheets/${currentSha}`,
            }),
          },
        }),
      ],
    });
    if (mapped) {
      mocks.placementTargets.set("spec-1", currentSha);
      mocks.placements.set(commentName, {
        state: "CURRENT",
        range: { startLine: 20, endLine: 21 },
      });
    }
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => {
      root.render(
        <ReviewActivityTimeline
          comments={[
            create(IssueCommentSchema, {
              name: commentName,
              comment: "Review this line",
              threadState: IssueComment_ThreadState.OPEN,
              statementAnchor: create(StatementAnchorSchema, {
                spec: "spec-1",
                sheetSha256: mapped ? "d".repeat(64) : currentSha,
                startPosition: create(PositionSchema, { line: 12, column: 0 }),
                endPosition: create(PositionSchema, { line: 12, column: 0 }),
              }),
            }),
          ]}
          issue={create(IssueSchema, { name: "projects/p1/issues/1" })}
          plan={plan}
        />
      );
    });
    act(() => {
      container.querySelector<HTMLButtonElement>("[data-testid='anchor-context']")?.click();
    });
    expect(mocks.requestThreadFocus).toHaveBeenCalledWith({
      commentName,
      specId: "spec-1",
      lineNumber: expectedLine,
    });
    expect(mocks.expandPhase).toHaveBeenCalledWith("changes");
    act(() => root.unmount());
  });

  test("a run's line shows and hides its finding cards in place, with Show and Hide in the link color", () => {
    const sha = "c".repeat(64);
    const plan = create(PlanSchema, {
      name: "projects/p1/plans/1",
      specs: [
        create(Plan_SpecSchema, {
          id: "spec-1",
          config: {
            case: "changeDatabaseConfig",
            value: create(Plan_ChangeDatabaseConfigSchema, {
              sheet: `projects/p1/sheets/${sha}`,
            }),
          },
        }),
      ],
    });
    const completion = (
      id: string,
      attempt: bigint,
      counts: { postedCount?: number; keptCount?: number; supersededCount?: number }
    ) =>
      create(IssueCommentSchema, {
        name: `projects/p1/issues/1/issueComments/${id}`,
        event: {
          case: "reviewRunCompletion",
          value: create(IssueComment_ReviewRunCompletionSchema, {
            attempt,
            ...counts,
          }),
        },
      });
    const finding = (
      id: string,
      ms: number,
      superseded: boolean,
      attempt: bigint
    ) =>
      create(IssueCommentSchema, {
        name: `projects/p1/issues/1/issueComments/${id}`,
        comment: "No WHERE clause",
        threadState: superseded
          ? IssueComment_ThreadState.RESOLVED
          : IssueComment_ThreadState.OPEN,
        createTime: timestampFromMs(ms),
        statementAnchor: create(StatementAnchorSchema, {
          spec: "spec-1",
          sheetSha256: sha,
          startPosition: create(PositionSchema, { line: 1, column: 0 }),
          endPosition: create(PositionSchema, { line: 1, column: 0 }),
        }),
        reviewMetadata: create(IssueComment_ReviewMetadataSchema, {
          superseded,
          attempt,
        }),
      });
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => {
      root.render(
        <ReviewActivityTimeline
          comments={[
            completion("c0", 0n, { postedCount: 1 }),
            finding("old", 0, true, 0n),
            completion("c1", 1n, { postedCount: 1, supersededCount: 1 }),
            finding("new", 60_000, false, 1n),
            completion("c2", 2n, { keptCount: 1 }),
          ]}
          issue={create(IssueSchema, { name: "projects/p1/issues/1" })}
          plan={plan}
        />
      );
    });
    const cards = () =>
      [...container.querySelectorAll("[data-testid='thread-card']")].map(
        (card) => card.getAttribute("data-root")?.split("/").pop()
      );
    const runButtons = () =>
      [...container.querySelectorAll("button[aria-expanded]")];
    expect(cards()).toEqual(["new"]);
    // The run that found nothing new has its line but nothing to show.
    expect(runButtons().map((button) => button.textContent)).toEqual([
      "common.show",
      "common.hide",
    ]);
    for (const button of runButtons()) {
      expect(button.className).toContain("text-accent");
      expect(button.className).not.toContain("text-control-light");
    }
    const text = container.textContent ?? "";
    expect(text).toContain("plan.review.activity.n-findings");
    expect(text).toContain("plan.review.activity.n-no-longer-found");
    expect(text).toContain("plan.review.activity.n-still-found");
    // The re-run posted an open finding, but its line counts only changes.
    expect(text).not.toContain("plan.automated-review.n-open");

    act(() => (runButtons()[0] as HTMLButtonElement).click());
    expect(cards()).toEqual(["old", "new"]);

    act(() => (runButtons()[1] as HTMLButtonElement).click());
    expect(cards()).toEqual(["old"]);
    expect(mocks.requestThreadFocus).not.toHaveBeenCalled();
    act(() => root.unmount());
  });

  test("unfolds the timeline to reach a card asked for from elsewhere", () => {
    const name = (i: number) => `projects/p1/issues/1/issueComments/t${i}`;
    const comments = Array.from({ length: 12 }, (_, i) =>
      create(IssueCommentSchema, {
        name: name(i),
        comment: `thread ${i}`,
        threadState: IssueComment_ThreadState.OPEN,
        createTime: timestampFromMs(i * 1000),
      })
    );
    const scrollIntoView = vi.fn();
    const original = Element.prototype.scrollIntoView;
    Element.prototype.scrollIntoView = scrollIntoView;
    mocks.activityFocus = { commentName: name(6), nonce: 4 };
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    act(() => {
      root.render(
        <ReviewActivityTimeline
          comments={comments}
          issue={create(IssueSchema, { name: "projects/p1/issues/1" })}
          plan={create(PlanSchema, { name: "projects/p1/plans/1" })}
        />
      );
    });
    const card = document.getElementById(name(6));
    expect(card).not.toBeNull();
    expect(scrollIntoView.mock.contexts).toContain(card);
    expect(mocks.clearActivityFocus).toHaveBeenCalledWith(4);
    act(() => root.unmount());
    container.remove();
    Element.prototype.scrollIntoView = original;
  });

  test("hands a thread card the plan spec its anchor names, and none for a spec the plan lacks", () => {
    const plan = create(PlanSchema, {
      name: "projects/p1/plans/1",
      specs: [create(Plan_SpecSchema, { id: "spec-1" })],
    });
    const thread = (id: string, spec: string) =>
      create(IssueCommentSchema, {
        name: `projects/p1/issues/1/issueComments/${id}`,
        comment: "Review this line",
        threadState: IssueComment_ThreadState.OPEN,
        statementAnchor: create(StatementAnchorSchema, { spec }),
      });
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => {
      root.render(
        <ReviewActivityTimeline
          comments={[thread("kept", "spec-1"), thread("gone", "spec-gone")]}
          issue={create(IssueSchema, { name: "projects/p1/issues/1" })}
          plan={plan}
        />
      );
    });
    const cardOf = (id: string) =>
      container.querySelector(
        `[data-root='projects/p1/issues/1/issueComments/${id}']`
      );
    expect(cardOf("kept")).not.toBeNull();
    expect(cardOf("gone")).not.toBeNull();
    expect(cardOf("kept")?.getAttribute("data-spec")).toBe("spec-1");
    expect(cardOf("gone")?.hasAttribute("data-spec")).toBe(false);
    act(() => root.unmount());
  });

  test("times an activity entry in the work-queue form", () => {
    // A feed is read for what just happened, so its entries age with the
    // clock rather than naming a date.
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(
        <ReviewActivityTimeline
          comments={[
            create(IssueCommentSchema, {
              name: "comments/timed",
              creator: "users/submitter@example.com",
              createTime: timestampFromMs(Date.UTC(2026, 2, 2, 12)),
              event: {
                case: "reviewSubmission",
                value: create(IssueComment_ReviewSubmissionSchema),
              },
            }),
          ]}
          issue={create(IssueSchema, { name: "projects/p1/issues/1" })}
          plan={create(PlanSchema, { name: "projects/p1/plans/1" })}
        />
      );
    });

    expect(shownTimestampModes(container)).toEqual(["queue"]);

    act(() => root.unmount());
  });

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

  test("a thread that is not on lines gets a card with a bar and no excerpt", () => {
    const issue = create(IssueSchema, { name: "projects/p1/issues/1", draft: false });
    const plan = create(PlanSchema, { name: "projects/p1/plans/1" });
    const rootName = "projects/p1/issues/1/issueComments/api";
    const comments = [
      create(IssueCommentSchema, {
        name: rootName,
        comment: "Posted from an API client",
        creator: "users/submitter@example.com",
        threadState: IssueComment_ThreadState.OPEN,
      }),
    ];
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => {
      root.render(<ReviewActivityTimeline comments={comments} issue={issue} plan={plan} />);
    });
    const card = container.querySelector("[data-testid='thread-card']");
    expect(card?.getAttribute("data-root")).toBe(rootName);
    expect(card?.getAttribute("data-fold-label")).toBe("on the plan");
    expect(card?.querySelector("[data-testid='anchor-context']")).toBeNull();
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
});
