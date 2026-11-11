import { create } from "@bufbuild/protobuf";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { act, useState } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MonacoViewZoneRevealContext } from "@/components/monaco/MonacoViewZone";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { shownTimestampInstants } from "@/test-utils/humanizeTs";
import {
  IssueComment_ReviewMetadata_Priority,
  IssueComment_ReviewMetadataSchema,
  IssueComment_ThreadState,
  IssueCommentSchema,
  IssueSchema,
  ReviewRun_Type,
  StatementAnchorSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { Plan_SpecSchema } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";

const mocks = vi.hoisted(() => ({
  sqlReviewV2: false,
  createIssueComment: vi.fn(),
  notify: vi.fn(),
  updateIssueComment: vi.fn(),
  openAgentWithPrompt: vi.fn(),
  diagnosticHelp: vi.fn(),
  hasPermission: vi.fn((_project: unknown, _permission: unknown) => true),
}));

vi.mock("@/modules/agent/logic/instanceConnectionHelp", () => ({
  openAgentWithPrompt: mocks.openAgentWithPrompt,
}));
vi.mock("@/modules/agent/logic/diagnosticHelp", async (importOriginal) => {
  const original = await importOriginal<typeof import("@/modules/agent/logic/diagnosticHelp")>();
  return { ...original, openAgentDiagnosticHelp: (context: Parameters<typeof original.openAgentDiagnosticHelp>[0]) => {
    mocks.diagnosticHelp(context);
    return original.openAgentDiagnosticHelp(context);
  } };
});

vi.mock("@/utils/featureGates", () => ({
  sqlReviewV2Enabled: () => mocks.sqlReviewV2,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, options?: { count?: number }) =>
      options?.count !== undefined ? `${key}:${options.count}` : key,
  }),
}));

vi.mock("@/components/HumanizeTs", async () => ({
  ...(await import("@/test-utils/humanizeTs")).humanizeTsStub(),
}));

// The activity module drags Monaco into the test; only the edit rule matters.
vi.mock("@/components/issue-activity/IssueCommentActivity", () => ({
  canEditIssueComment: () => true,
}));

vi.mock("@/components/MarkdownEditor", async (importOriginal) => ({
  markdownSummary: (
    await importOriginal<typeof import("@/components/MarkdownEditor")>()
  ).markdownSummary,
  MarkdownEditor: ({
    content,
    autoFocus,
    mode,
    onChange,
    onSubmit,
    placeholder,
  }: {
    content: string;
    autoFocus?: boolean;
    mode?: string;
    onChange?: (value: string) => void;
    onSubmit?: () => void;
    placeholder?: string;
  }) =>
    mode === "preview" ? (
      <p data-testid="preview">{content}</p>
    ) : (
      <textarea
        autoFocus={autoFocus}
        onChange={(event) => onChange?.(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter" && (event.ctrlKey || event.metaKey)) onSubmit?.();
        }}
        placeholder={placeholder}
        value={content}
      />
    ),
}));

vi.mock("@/hooks/useAppState", () => ({
  useCurrentUser: () => ({ email: "me@example.com", title: "Me" }),
  useUserByIdentifier: (principal: string | undefined) =>
    principal
      ? {
          name: principal,
          email: principal.replace(/^users\//, ""),
          title: principal.replace(/^users\//, "").split("@")[0],
        }
      : undefined,
}));

vi.mock("@/stores", () => ({
  pushNotification: mocks.notify,
  extractUserEmail: (identifier: string) => identifier.replace(/^users\//, ""),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: unknown) => unknown) =>
      selector({
        databasesByName: {},
        environmentList: [],
        getUserByIdentifier: (principal: string) => ({
          name: principal,
          email: principal.replace(/^users\//, ""),
          title: principal.replace(/^users\//, "").split("@")[0],
        }),
      }),
    {
      getState: () => ({
        createIssueComment: mocks.createIssueComment,
        updateIssueComment: mocks.updateIssueComment,
      }),
    }
  ),
}));

vi.mock("@/utils/iam/permission", () => ({
  hasProjectPermissionV2: (project: unknown, permission: unknown) =>
    mocks.hasPermission(project, permission),
}));

import { CommentThreadCard } from "./CommentThreadCard";
import { groupThreads } from "./threadModel";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const ISSUE = "projects/p/issues/1";
const issue = create(IssueSchema, { name: ISSUE });
const project = { name: "projects/p" } as Project;

const comment = (
  id: string,
  text: string,
  extra: {
    root?: string;
    resolved?: boolean;
    creator?: string;
    writtenMs?: number;
    editedMs?: number;
  } = {}
) =>
  create(IssueCommentSchema, {
    name: `${ISSUE}/issueComments/${id}`,
    comment: text,
    creator: extra.creator ?? "users/alice@example.com",
    createTime: timestampFromMs(extra.writtenMs ?? 1_000),
    updateTime: timestampFromMs(extra.editedMs ?? extra.writtenMs ?? 1_000),
    root: extra.root,
    threadState:
      extra.root !== undefined
        ? undefined
        : extra.resolved
          ? IssueComment_ThreadState.RESOLVED
          : IssueComment_ThreadState.OPEN,
  });

const rootName = `${ISSUE}/issueComments/root`;

let container: HTMLDivElement;
let root: Root;

beforeEach(() => {
  mocks.sqlReviewV2 = false;
  HTMLElement.prototype.scrollIntoView = vi.fn();
  vi.clearAllMocks();
  mocks.hasPermission.mockReturnValue(true);
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => root.unmount());
  container.remove();
  Reflect.deleteProperty(HTMLElement.prototype, "scrollIntoView");
});

const render = (ui: React.ReactElement) => act(() => root.render(ui));

const click = (element: Element | null | undefined) => {
  if (!element) throw new Error("element not found");
  act(() => {
    element.dispatchEvent(new MouseEvent("click", { bubbles: true }));
  });
};

const buttonByText = (text: string) =>
  Array.from(container.querySelectorAll("button")).find((button) =>
    button.textContent?.includes(text)
  );

const stateCheckbox = () => container.querySelector('[role="checkbox"]');

const originalHash = "a".repeat(64);
const finding = () => {
  const root = comment("finding", "Missing WHERE clause in the original SQL.", {
    creator: "",
  });
  root.statementAnchor = create(StatementAnchorSchema, {
    spec: "spec-1",
    sheetSha256: originalHash,
    startPosition: { line: 3, column: 0 },
    endPosition: { line: 5, column: 0 },
  });
  root.reviewMetadata = create(IssueComment_ReviewMetadataSchema, {
    runType: ReviewRun_Type.RULE,
    ruleType: ReviewRuleType.WALK_THROUGH,
    priority: IssueComment_ReviewMetadata_Priority.P0,
    attempt: 7n,
    codes: ["WT101"],
    targets: ["instances/main/databases/books"],
  });
  const spec = create(Plan_SpecSchema, {
    id: "spec-1",
    config: {
      case: "changeDatabaseConfig",
      value: {
        targets: ["instances/main/databases/books"],
        sheet: `projects/p/sheets/${"b".repeat(64)}`,
      },
    },
  });
  return { root, spec };
};

const diagnosticSnapshot = () => {
  const [prompt, references, , options] = mocks.openAgentWithPrompt.mock.lastCall!;
  return {
    snapshot: JSON.parse(prompt.slice(prompt.indexOf("\n") + 1)),
    references,
    options,
  };
};

describe("automated finding diagnostic help", () => {
  test("click binds one owning Plan and preserves original finding evidence, not the current sheet", () => {
    mocks.sqlReviewV2 = true;
    const { root: findingRoot, spec } = finding();
    render(<CommentThreadCard issue={create(IssueSchema, { name: ISSUE, plan: "projects/p/plans/42" })} project={project} spec={spec} thread={groupThreads([findingRoot])[0]} />);
    expect(mocks.openAgentWithPrompt).not.toHaveBeenCalled();
    expect(container.querySelectorAll('[aria-label="agent.diagnostic-help.ask-agent"]')).toHaveLength(1);

    click(buttonByText("agent.diagnostic-help.ask-agent"));

    const { snapshot, references, options } = diagnosticSnapshot();
    expect(references).toEqual([{ kind: "plan", name: "projects/p/plans/42", project: "projects/p", label: "projects/p/plans/42" }]);
    expect(options).toEqual({ capability: { kind: "diagnostic", resource: references[0] } });
    expect(snapshot).toMatchObject({ kind: "review-finding", sourceId: `${ISSUE}/issueComments/finding`, detail: "Missing WHERE clause in the original SQL." });
    expect(snapshot.facts).toEqual(expect.arrayContaining([
      { label: "Issue", value: ISSUE },
      { label: "Review run", value: `${ISSUE}/reviewRuns/rule (RULE), attempt 7` },
      { label: "Rule", value: "WALK_THROUGH: WT101" },
      { label: "Priority", value: "P0" },
      { label: "Targets", value: "instances/main/databases/books" },
      { label: "Spec", value: "spec-1" },
      { label: "Original sheet SHA256", value: originalHash },
      { label: "Original SQL range", value: "3:0-5:0" },
    ]));
    expect(snapshot.facts).toContainEqual({ label: "Evidence", value: "Partial: original SQL snapshot unavailable; use the original revision identifiers, not current SQL." });
    expect(snapshot.sql).toBeUndefined();
    expect(mocks.diagnosticHelp.mock.lastCall?.[0].sqlFixSource).toEqual({ specId: "spec-1", sha256: originalHash, targets: ["instances/main/databases/books"] });
    expect(JSON.stringify(snapshot)).not.toContain("b".repeat(64));
    expect(mocks.createIssueComment).not.toHaveBeenCalled();
    expect(mocks.updateIssueComment).not.toHaveBeenCalled();
  });

  test("resolved and ordinary threads have no action; an explicitly reopened superseded finding retains its revision", () => {
    const { root: findingRoot, spec } = finding();
    findingRoot.reviewMetadata!.superseded = true;
    const props = { issue, project, spec, collapsible: false };
    const view = () => <CommentThreadCard {...props} thread={groupThreads([findingRoot])[0]} />;
    findingRoot.threadState = IssueComment_ThreadState.RESOLVED;
    render(view());
    expect(buttonByText("agent.diagnostic-help.ask-agent")).toBeUndefined();
    findingRoot.threadState = IssueComment_ThreadState.OPEN;
    render(view());
    click(buttonByText("agent.diagnostic-help.ask-agent"));
    expect(diagnosticSnapshot().references).toEqual([{ kind: "issue", name: ISSUE, project: "projects/p", label: ISSUE }]);
    expect(diagnosticSnapshot().snapshot.facts).toContainEqual({ label: "Original sheet SHA256", value: originalHash });
    expect(mocks.diagnosticHelp.mock.lastCall?.[0].sqlFixSource).toEqual({ specId: "spec-1", sha256: originalHash, targets: ["instances/main/databases/books"] });
    findingRoot.threadState = IssueComment_ThreadState.RESOLVED;
    render(view());
    expect(buttonByText("agent.diagnostic-help.ask-agent")).toBeUndefined();
    render(<CommentThreadCard {...props} thread={groupThreads([comment("ordinary", "A normal comment")])[0]} />);
    expect(buttonByText("agent.diagnostic-help.ask-agent")).toBeUndefined();
  });

  test("an unanchored AI finding uses its Issue without inventing a SQL revision", () => {
    const findingRoot = comment("ai-finding", "The proposed change may discard records.", { creator: "" });
    findingRoot.reviewMetadata = create(IssueComment_ReviewMetadataSchema, {
      runType: ReviewRun_Type.AI,
      title: "Potential data loss",
      priority: IssueComment_ReviewMetadata_Priority.P1,
      attempt: 2n,
    });
    render(<CommentThreadCard issue={issue} project={project} thread={groupThreads([findingRoot])[0]} />);

    click(buttonByText("agent.diagnostic-help.ask-agent"));

    const { snapshot, references } = diagnosticSnapshot();
    expect(references).toEqual([{ kind: "issue", name: ISSUE, project: "projects/p", label: ISSUE }]);
    expect(snapshot.summary).toBe("Potential data loss");
    expect(snapshot.detail).toBe("The proposed change may discard records.");
    expect(snapshot.facts).toContainEqual({ label: "Review run", value: `${ISSUE}/reviewRuns/ai (AI), attempt 2` });
    expect(snapshot.facts.some((fact: { label: string }) => fact.label === "Original sheet SHA256")).toBe(false);
    expect(snapshot.sql).toBeUndefined();
    expect(mocks.diagnosticHelp.mock.lastCall?.[0].sqlFixSource).toBeUndefined();
  });

  test("retains the finding's database targets instead of its retargeted spec", () => {
    const { root: findingRoot, spec } = finding();
    if (spec.config.case === "changeDatabaseConfig")
      spec.config.value.targets = ["instances/main/databases/other"];
    render(<CommentThreadCard issue={issue} project={project} spec={spec} thread={groupThreads([findingRoot])[0]} />);

    click(buttonByText("agent.diagnostic-help.ask-agent"));
    findingRoot.reviewMetadata!.targets.push("instances/main/databases/later");

    expect(mocks.diagnosticHelp.mock.lastCall?.[0].sqlFixSource).toEqual({
      specId: "spec-1",
      sha256: originalHash,
      targets: ["instances/main/databases/books"],
    });
  });

  test.each(["missing-spec", "missing-hash", "invalid-hash"])("keeps incomplete anchor %s diagnose-only", (missing) => {
    const { root: findingRoot } = finding();
    findingRoot.reviewMetadata!.targets = [];
    if (missing === "missing-spec") findingRoot.statementAnchor!.spec = "";
    if (missing === "missing-hash") findingRoot.statementAnchor!.sheetSha256 = "";
    if (missing === "invalid-hash") findingRoot.statementAnchor!.sheetSha256 = "g".repeat(64);
    render(<CommentThreadCard issue={issue} project={project} thread={groupThreads([findingRoot])[0]} />);

    click(buttonByText("agent.diagnostic-help.ask-agent"));

    expect(mocks.diagnosticHelp.mock.lastCall?.[0].sqlFixSource).toBeUndefined();
  });

  test.each(["project", "comment", "spec"])("suppresses mismatched %s evidence", (mismatch) => {
    const { root: findingRoot, spec } = finding();
    if (mismatch === "comment") findingRoot.name = "projects/p/issues/2/issueComments/finding";
    if (mismatch === "spec") findingRoot.reviewMetadata!.targets = [];
    const thread = groupThreads([findingRoot])[0];
    render(<CommentThreadCard issue={issue} project={mismatch === "project" ? { ...project, name: "projects/other" } : project} spec={mismatch === "spec" ? create(Plan_SpecSchema, { ...spec, id: "another-spec" }) : spec} thread={thread} />);
    expect(buttonByText("agent.diagnostic-help.ask-agent")).toBeUndefined();
    expect(mocks.openAgentWithPrompt).not.toHaveBeenCalled();
  });

  test.each([
    { name: "projects/p/issues/create" },
    { name: "projects/p/issues/-" },
    { name: "projects/other/issues/1" },
    { name: ISSUE, plan: "projects/other/plans/42" },
    { name: ISSUE, plan: "projects/p/plans/create" },
    { name: ISSUE, plan: "projects/p/plans/0" },
    { name: ISSUE, plan: "projects/p/plans/42/extra" },
  ])("suppresses invalid or mismatched owning names %j after a valid render", (owner) => {
    const { root: findingRoot, spec } = finding();
    const thread = groupThreads([findingRoot])[0];
    render(<CommentThreadCard issue={issue} project={project} spec={spec} thread={thread} />);
    expect(buttonByText("agent.diagnostic-help.ask-agent")).toBeDefined();
    render(<CommentThreadCard issue={create(IssueSchema, owner)} project={project} spec={spec} thread={thread} />);
    expect(buttonByText("agent.diagnostic-help.ask-agent")).toBeUndefined();
    expect(mocks.openAgentWithPrompt).not.toHaveBeenCalled();
  });
});

// Submits the open composer, optionally flipping the thread-state checkbox
// first so the reply also resolves or reopens the thread.
const submitReply = ({ toggleState = false } = {}) => {
  if (toggleState) click(stateCheckbox());
  click(buttonByText("plan.review.thread.reply"));
};

// Grants reply but not the update permission the settle actions need.
const denySettle = () =>
  mocks.hasPermission.mockImplementation(
    (_project: unknown, permission: unknown) =>
      permission === "bb.issueComments.create"
  );

describe("CommentThreadCard", () => {
  test("draft issues keep existing discussion visible and allow replies only after submission", async () => {
    const [thread] = groupThreads([
      comment("root", "Root question"),
      comment("reply", "Existing reply", { root: rootName }),
    ]);
    const props = { project, thread };
    render(<CommentThreadCard {...props} issue={create(IssueSchema, { ...issue, draft: true })} />);
    expect(container.textContent).toContain("Root question");
    expect(container.textContent).toContain("Existing reply");
    expect(buttonByText("plan.review.thread.reply-placeholder")).toBeUndefined();

    render(<CommentThreadCard {...props} issue={issue} replyDraft="New reply" />);
    click(buttonByText("plan.review.thread.reply-placeholder"));
    mocks.createIssueComment.mockResolvedValueOnce(comment("new-reply", "New reply", { root: rootName }));
    submitReply();
    await act(async () => {});
    expect(mocks.createIssueComment).toHaveBeenCalledExactlyOnceWith({ issueName: ISSUE, comment: "New reply", root: rootName });
  });

  test("under SQL Review V2 a draft takes replies", () => {
    mocks.sqlReviewV2 = true;
    const [thread] = groupThreads([comment("root", "Root question")]);
    render(<CommentThreadCard project={project} thread={thread} issue={create(IssueSchema, { ...issue, draft: true })} />);
    expect(buttonByText("plan.review.thread.reply-placeholder")).toBeDefined();
  });

  test("places statement context above the root author as a separate full-width region", () => {
    const [thread] = groupThreads([comment("root", "Root question")]);
    render(
      <CommentThreadCard
        issue={issue}
        project={project}
        thread={thread}
        renderContext={() => <div data-testid="anchor-context">SELECT 1;</div>}
      />
    );
    const card = container.querySelector("[data-testid='comment-thread']");
    const anchor = container.querySelector("[data-testid='anchor-context']");
    const rootComment = container.querySelector("[data-testid='thread-comment']");
    expect(card?.firstElementChild?.contains(anchor)).toBe(true);
    expect(rootComment?.contains(anchor)).toBe(false);
    expect(anchor?.compareDocumentPosition(rootComment!)).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING
    );
  });

  test("times a reply by when it was written, not when it was edited", () => {
    // The two instants sit beside each other in ThreadComment and feed the
    // same "edited" marker, so the row has to say which one it shows.
    const writtenMs = Date.UTC(2026, 2, 2, 12);
    const reply = comment("edited", "Reply", {
      root: `${ISSUE}/issueComments/root`,
      writtenMs,
      editedMs: writtenMs + 60_000,
    });
    const [thread] = groupThreads([comment("root", "Root question"), reply]);
    render(
      <CommentThreadCard issue={issue} project={project} thread={thread} />
    );

    const shown = shownTimestampInstants(container);
    expect(shown).toContain(String(writtenMs));
    expect(shown).not.toContain(String(writtenMs + 60_000));
  });

  test("renders the root and replies oldest first with the thread footer", () => {
    const [thread] = groupThreads([
      comment("root", "Root question"),
      comment("r2", "Second reply", { root: rootName }),
      comment("r1", "First reply", { root: rootName }),
    ]);
    render(
      <CommentThreadCard issue={issue} project={project} thread={thread} />
    );
    const bodies = Array.from(
      container.querySelectorAll("[data-testid='preview']")
    ).map((node) => node.textContent);
    expect(bodies).toEqual(["Root question", "First reply", "Second reply"]);
    const comments = container.querySelectorAll("[data-testid='thread-comment']");
    const replies = container.querySelector("[data-testid='thread-replies']");
    expect(replies?.contains(comments[0])).toBe(false);
    expect(replies?.contains(comments[1])).toBe(true);
    expect(replies?.contains(comments[2])).toBe(true);
    expect(container.querySelector("[data-thread-state]")?.getAttribute("data-thread-state")).toBe("open");
    expect(container.querySelector("[data-testid='thread-state']")).toBeNull();
    expect(buttonByText("plan.review.thread.resolve")).toBeDefined();
    expect(buttonByText("plan.review.thread.reply-placeholder")).toBeDefined();
  });

  test("a resolved thread collapses to a summary and expands in place", () => {
    const [thread] = groupThreads([
      comment("root", "Root", { resolved: true }),
      comment("r1", "Reply", { root: rootName, creator: "users/bob@example.com" }),
    ]);
    render(
      <CommentThreadCard issue={issue} project={project} thread={thread} />
    );
    expect(container.textContent).toContain("plan.review.thread.n-replies:1");
    expect(container.querySelector("[data-testid='preview']")).toBeNull();

    click(buttonByText("plan.review.thread.resolved"));
    expect(
      container.querySelectorAll("[data-testid='preview']")
    ).toHaveLength(2);
    expect(buttonByText("common.reopen")).toBeDefined();
    expect(buttonByText("plan.review.thread.resolve")).toBeUndefined();
    const states = container.querySelectorAll("[data-testid='thread-state']");
    expect(states).toHaveLength(1);
    expect(
      container.querySelector("[data-testid='thread-reply-footer']")?.contains(states[0])
    ).toBe(true);
  });

  test("a resolved thread says so in its footer when the reader can neither reply nor reopen", () => {
    mocks.hasPermission.mockReturnValue(false);
    const [thread] = groupThreads([comment("root", "Root", { resolved: true })]);
    render(
      <CommentThreadCard
        collapsible={false}
        issue={issue}
        project={project}
        thread={thread}
      />
    );
    const footer = container.querySelector("[data-testid='thread-reply-footer']");
    expect(footer?.querySelector("[data-testid='thread-state']")?.textContent).toBe(
      "plan.review.thread.resolved"
    );
    expect(footer?.querySelectorAll("button")).toHaveLength(0);
  });

  test("a resolved review result summarizes as its reviewer", () => {
    const root = comment("root", "Missing WHERE clause.", {
      creator: "",
      resolved: true,
    });
    root.reviewMetadata = create(IssueComment_ReviewMetadataSchema, {
      runType: ReviewRun_Type.AI,
      priority: IssueComment_ReviewMetadata_Priority.P1,
      title: "Add a WHERE clause",
    });
    render(
      <CommentThreadCard
        issue={issue}
        project={project}
        thread={groupThreads([root])[0]}
      />
    );
    const summary = buttonByText("plan.review.thread.resolved");
    expect(summary?.textContent).toContain("sql-review.ai-review.self");
    expect(summary?.querySelector("[data-testid='review-priority']")?.textContent).toBe("P1");
    expect(summary?.textContent).toContain("Add a WHERE clause");
    expect(summary?.textContent).not.toContain("Missing WHERE clause.");
    expect(summary?.textContent).not.toContain("common.unknown");
  });

  test("a resolved anchored thread collapses from its context header", () => {
    const [thread] = groupThreads([comment("root", "Root", { resolved: true })]);
    render(
      <CommentThreadCard
        issue={issue}
        project={project}
        renderContext={(onCollapse) => (
          <button data-testid="context-collapse" onClick={onCollapse} type="button">
            Collapse context
          </button>
        )}
        thread={thread}
      />
    );
    click(buttonByText("plan.review.thread.resolved"));
    expect(container.querySelector("[data-testid='context-collapse']")).not.toBeNull();
    expect(container.querySelectorAll('button[aria-label="common.collapse"]')).toHaveLength(0);
    click(container.querySelector("[data-testid='context-collapse']"));
    expect(container.querySelector("[data-testid='context-collapse']")).toBeNull();
    expect(container.querySelector("[data-testid='preview']")).toBeNull();
  });

  test("roots and replies share header avatars and a full-width body", () => {
    const [thread] = groupThreads([comment("root", "Root"), comment("r1", "Reply", { root: rootName })]);
    render(<CommentThreadCard issue={issue} project={project} thread={thread} />);
    const comments = container.querySelectorAll("[data-testid='thread-comment']");
    for (const comment of comments) {
      const content = comment.querySelector("[data-testid='comment-content']");
      const avatar = comment.querySelector("[data-testid='comment-avatar']");
      expect(content?.firstElementChild?.contains(avatar)).toBe(true);
      expect(content?.querySelector("[data-testid='preview']")).not.toBeNull();
    }
    expect(container.querySelector("[data-testid='reply-composer-avatar']")).toBeNull();
    click(buttonByText("plan.review.thread.reply-placeholder"));
    expect(container.querySelector("[data-testid='reply-composer-avatar']")).not.toBeNull();
    click(buttonByText("common.cancel"));
    expect(container.querySelector("[data-testid='reply-composer-avatar']")).toBeNull();
    render(<CommentThreadCard issue={issue} project={project} thread={thread} />);
    expect(container.querySelector("[data-testid='thread-comment'] [data-testid='comment-avatar']")).not.toBeNull();
  });

  test("new replies do not hide replies already visible in a short thread", () => {
    const replies = Array.from({ length: 6 }, (_, index) => comment(`r${index}`, `Reply ${index}`, { root: rootName }));
    const [short] = groupThreads([comment("root", "Root"), ...replies.slice(0, 4)]);
    render(<CommentThreadCard issue={issue} project={project} thread={short} />);
    expect(container.querySelectorAll("[data-testid='preview']")).toHaveLength(5);
    const [long] = groupThreads([comment("root", "Root"), ...replies]);
    render(<CommentThreadCard issue={issue} project={project} thread={long} />);
    expect(container.querySelectorAll("[data-testid='preview']")).toHaveLength(7);
  });

  test("folds long threads and shows the tail until expanded", () => {
    const replies = Array.from({ length: 6 }, (_, i) =>
      comment(`r${i}`, `Reply ${i}`, { root: rootName })
    );
    const [thread] = groupThreads([comment("root", "Root"), ...replies]);
    render(
      <CommentThreadCard issue={issue} project={project} thread={thread} />
    );
    const visible = () =>
      Array.from(container.querySelectorAll("[data-testid='preview']")).map(
        (node) => node.textContent
      );
    expect(visible()).toEqual(["Root", "Reply 4", "Reply 5"]);
    expect(container.textContent).toContain(
      "plan.review.thread.n-replies-hidden:4"
    );
    click(buttonByText("plan.review.activity.show-all"));
    expect(visible()).toHaveLength(7);
  });

  test("replying creates a reply on the root and resolving updates the thread state", async () => {
    const [thread] = groupThreads([comment("root", "Root")]);
    mocks.createIssueComment.mockResolvedValue(
      comment("new", "Sounds good", { root: rootName })
    );
    mocks.updateIssueComment.mockResolvedValue(
      comment("root", "Root", { resolved: true })
    );
    render(
      <CommentThreadCard issue={issue} project={project} thread={thread} />
    );

    click(buttonByText("plan.review.thread.reply-placeholder"));
    const textarea = container.querySelector("textarea");
    expect(textarea?.placeholder).toBe("plan.review.thread.reply-placeholder");
    act(() => {
      const setter = Object.getOwnPropertyDescriptor(
        HTMLTextAreaElement.prototype,
        "value"
      )?.set;
      setter?.call(textarea, "Sounds good");
      textarea?.dispatchEvent(new Event("input", { bubbles: true }));
    });
    submitReply();
    await act(async () => {});
    expect(mocks.createIssueComment).toHaveBeenCalledWith({
      issueName: ISSUE,
      comment: "Sounds good",
      root: rootName,
    });
    // The reply box collapses after a successful post.
    expect(container.querySelector("textarea")).toBeNull();

    click(buttonByText("plan.review.thread.resolve"));
    await act(async () => {});
    expect(mocks.updateIssueComment).toHaveBeenCalledWith({
      issueCommentName: rootName,
      threadState: IssueComment_ThreadState.RESOLVED,
    });
  });

  test.each([false, true])("notifies the owner after a successful local state change: resolved=%s", async (resolved) => {
    const [thread] = groupThreads([comment("root", "Root", { resolved })]);
    const onThreadStateChanged = vi.fn();
    mocks.updateIssueComment.mockResolvedValue(comment("root", "Root", { resolved: !resolved }));
    render(<CommentThreadCard issue={issue} project={project} thread={thread} collapsible={false} onThreadStateChanged={onThreadStateChanged} />);
    click(buttonByText(resolved ? "common.reopen" : "plan.review.thread.resolve"));
    await act(async () => {});
    expect(onThreadStateChanged).toHaveBeenCalledExactlyOnceWith(!resolved);
  });

  test("does not advance on failure or a remote state refresh", async () => {
    const [thread] = groupThreads([comment("root", "Root")]);
    const onThreadStateChanged = vi.fn();
    mocks.updateIssueComment.mockRejectedValueOnce(new Error("offline"));
    render(<CommentThreadCard issue={issue} project={project} thread={thread} collapsible={false} onThreadStateChanged={onThreadStateChanged} />);
    click(buttonByText("plan.review.thread.resolve"));
    await act(async () => {});
    expect(onThreadStateChanged).not.toHaveBeenCalled();
    const [updated] = groupThreads([comment("root", "Root", {resolved: true})]);
    render(<CommentThreadCard issue={issue} project={project} thread={updated} collapsible={false} onThreadStateChanged={onThreadStateChanged} />);
    expect(onThreadStateChanged).not.toHaveBeenCalled();
    expect(container.querySelector("[data-testid='preview']")).not.toBeNull();
  });

  test("does not advance if the user leaves the card before Resolve completes", async () => {
    const [thread] = groupThreads([comment("root", "Root")]);
    const onThreadStateChanged = vi.fn();
    let complete!: (value: ReturnType<typeof comment>) => void;
    mocks.updateIssueComment.mockReturnValueOnce(new Promise((resolve) => {complete = resolve;}));
    render(<CommentThreadCard issue={issue} project={project} thread={thread} collapsible={false} onThreadStateChanged={onThreadStateChanged} />);
    click(buttonByText("plan.review.thread.resolve"));
    render(<div />);
    await act(async () => {complete(comment("root", "Root", {resolved: true}));});
    expect(onThreadStateChanged).not.toHaveBeenCalled();
  });

  test.each([false, true])("posts the reply before changing status: initially resolved=%s", async (resolved) => {
    const [thread] = groupThreads([comment("root", "Root", {resolved})]);
    let finishReply!: (value: ReturnType<typeof comment>) => void;
    let finishState!: (value: ReturnType<typeof comment>) => void;
    mocks.createIssueComment.mockImplementationOnce(() => new Promise((resolve) => {finishReply = resolve;}));
    mocks.updateIssueComment.mockImplementationOnce(() => new Promise((resolve) => {finishState = resolve;}));
    const onReplyDraftChange = vi.fn();
    const onThreadStateChanged = vi.fn();
    render(<CommentThreadCard issue={issue} project={project} thread={thread} collapsible={false} replyDraft="Decision details" onReplyDraftChange={onReplyDraftChange} onThreadStateChanged={onThreadStateChanged} />);
    // Open threads resolve when the box is ticked; resolved ones reopen by
    // default, so the box is left alone there.
    submitReply({ toggleState: !resolved });
    expect(mocks.createIssueComment).toHaveBeenCalledExactlyOnceWith({issueName: ISSUE, root: rootName, comment: "Decision details"});
    expect(mocks.updateIssueComment).not.toHaveBeenCalled();
    await act(async () => {finishReply(comment("reply", "Decision details", {root: rootName}));});
    expect(onReplyDraftChange).toHaveBeenCalledWith(expect.any(Function));
    expect(mocks.updateIssueComment).toHaveBeenCalledExactlyOnceWith({issueCommentName: rootName, threadState: resolved ? IssueComment_ThreadState.OPEN : IssueComment_ThreadState.RESOLVED});
    expect(buttonByText(resolved ? "common.reopen" : "plan.review.thread.resolve")?.disabled).toBe(true);
    expect(onThreadStateChanged).not.toHaveBeenCalled();
    await act(async () => {finishState(comment("root", "Root", {resolved: !resolved}));});
    expect(onThreadStateChanged).toHaveBeenCalledExactlyOnceWith(!resolved);
  });

  test.each([false, true])("the composer checkbox mirrors the thread state: resolved=%s", (resolved) => {
    const [thread] = groupThreads([comment("root", "Root", { resolved })]);
    render(<CommentThreadCard issue={issue} project={project} thread={thread} collapsible={false} replyDraft="Draft" onReplyDraftChange={vi.fn()} />);
    expect(stateCheckbox()?.getAttribute("aria-checked")).toBe(String(resolved));
    expect(container.querySelector("label")?.textContent).toBe(
      resolved ? "plan.review.thread.reopen-thread" : "plan.review.thread.resolve-thread"
    );
    expect(buttonByText("plan.review.thread.reply")).toBeDefined();
    expect(container.querySelector('button[aria-label="common.actions"]')).toBeNull();
  });

  test("an untouched checkbox on an open thread posts a plain reply", async () => {
    const [thread] = groupThreads([comment("root", "Root")]);
    const onThreadStateChanged = vi.fn();
    mocks.createIssueComment.mockResolvedValueOnce(comment("reply", "Just a note", { root: rootName }));
    render(<CommentThreadCard issue={issue} project={project} thread={thread} collapsible={false} replyDraft="Just a note" onReplyDraftChange={vi.fn()} onThreadStateChanged={onThreadStateChanged} />);
    submitReply();
    await act(async () => {});
    expect(mocks.createIssueComment).toHaveBeenCalledTimes(1);
    expect(mocks.updateIssueComment).not.toHaveBeenCalled();
    expect(onThreadStateChanged).not.toHaveBeenCalled();
  });

  test("an untouched checkbox on a resolved thread reopens it with the reply; unticking keeps it resolved", async () => {
    const [thread] = groupThreads([comment("root", "Root", { resolved: true })]);
    const onThreadStateChanged = vi.fn();
    mocks.createIssueComment.mockResolvedValue(comment("reply", "Not done yet", { root: rootName }));
    mocks.updateIssueComment.mockResolvedValueOnce(comment("root", "Root", { resolved: false }));
    const props = { issue, project, thread, collapsible: false, onReplyDraftChange: vi.fn(), onThreadStateChanged };
    render(<CommentThreadCard {...props} replyDraft="Not done yet" />);
    expect(stateCheckbox()?.getAttribute("aria-checked")).toBe("true");
    submitReply();
    await act(async () => {});
    expect(mocks.updateIssueComment).toHaveBeenCalledExactlyOnceWith({ issueCommentName: rootName, threadState: IssueComment_ThreadState.OPEN });
    expect(onThreadStateChanged).toHaveBeenCalledExactlyOnceWith(false);

    vi.clearAllMocks();
    mocks.hasPermission.mockReturnValue(true);
    mocks.createIssueComment.mockResolvedValue(comment("reply", "Still resolved", { root: rootName }));
    render(<CommentThreadCard {...props} replyDraft="Still resolved" />);
    // The composer closed after the first reply; open it again.
    click(buttonByText("plan.review.thread.reply-placeholder"));
    submitReply({ toggleState: true });
    await act(async () => {});
    expect(mocks.createIssueComment).toHaveBeenCalledTimes(1);
    expect(mocks.updateIssueComment).not.toHaveBeenCalled();
  });

  test("hides the state checkbox without the update permission", () => {
    denySettle();
    const [thread] = groupThreads([comment("root", "Root")]);
    render(<CommentThreadCard issue={issue} project={project} thread={thread} collapsible={false} replyDraft="Draft" onReplyDraftChange={vi.fn()} />);
    expect(stateCheckbox()).toBeNull();
    expect(buttonByText("plan.review.thread.reply")).toBeDefined();
  });

  test("reply failure preserves the draft and never changes status", async () => {
    const [thread] = groupThreads([comment("root", "Root")]);
    const onReplyDraftChange = vi.fn();
    mocks.createIssueComment.mockRejectedValueOnce(new Error("offline"));
    render(<CommentThreadCard issue={issue} project={project} thread={thread} replyDraft="Keep this draft" onReplyDraftChange={onReplyDraftChange} />);
    submitReply({ toggleState: true });
    await act(async () => {});
    expect(mocks.updateIssueComment).not.toHaveBeenCalled();
    expect(onReplyDraftChange).not.toHaveBeenCalled();
    expect(container.querySelector("textarea")?.value).toBe("Keep this draft");
  });

  test("a failed status update can be retried without creating another reply", async () => {
    const [thread] = groupThreads([comment("root", "Root")]);
    const onThreadStateChanged = vi.fn();
    const onReplyDraftChange = vi.fn();
    mocks.createIssueComment.mockResolvedValueOnce(comment("reply", "Posted", {root: rootName}));
    mocks.updateIssueComment.mockRejectedValueOnce(new Error("status update failed"));
    render(<CommentThreadCard issue={issue} project={project} thread={thread} replyDraft="Posted" onReplyDraftChange={onReplyDraftChange} onThreadStateChanged={onThreadStateChanged} />);
    submitReply({ toggleState: true });
    await act(async () => {});
    expect(onReplyDraftChange).toHaveBeenCalledWith(expect.any(Function));
    expect(onThreadStateChanged).not.toHaveBeenCalled();
    expect(mocks.notify).toHaveBeenCalledWith(expect.objectContaining({title: "plan.review.thread.reply-posted-status-failed"}));
    expect(container.querySelector("textarea")).toBeNull();
    mocks.updateIssueComment.mockResolvedValueOnce(comment("root", "Root", {resolved: true}));
    click(buttonByText("plan.review.thread.resolve"));
    await act(async () => {});
    expect(mocks.createIssueComment).toHaveBeenCalledTimes(1);
    expect(mocks.updateIssueComment).toHaveBeenCalledTimes(2);
    expect(onThreadStateChanged).toHaveBeenCalledExactlyOnceWith(true);
  });

  test("authoring the root grants no Resolve without the update permission", () => {
    denySettle();
    const [thread] = groupThreads([
      comment("root", "Root", { creator: "users/me@example.com" }),
    ]);
    render(
      <CommentThreadCard issue={issue} project={project} thread={thread} />
    );
    expect(buttonByText("plan.review.thread.resolve")).toBeUndefined();
  });

  test("hides Resolve from users who may neither update nor own the root", () => {
    denySettle();
    const [thread] = groupThreads([comment("root", "Root")]);
    render(
      <CommentThreadCard issue={issue} project={project} thread={thread} />
    );
    expect(buttonByText("plan.review.thread.reply-placeholder")).toBeDefined();
    expect(buttonByText("plan.review.thread.resolve")).toBeUndefined();
  });
});


test.each(["ctrlKey", "metaKey"])("ignores %s+Enter while a reply is pending", async (modifier) => {
  const [thread] = groupThreads([comment("root", "Root question")]);
  let complete!: (value: ReturnType<typeof comment>) => void;
  mocks.createIssueComment.mockImplementationOnce(() => new Promise(resolve => {complete = resolve;}));
  const onReplyDraftChange = vi.fn();
  render(<CommentThreadCard issue={issue} project={project} thread={thread} replyDraft="My reply" onReplyDraftChange={onReplyDraftChange} />);
  submitReply();
  expect(mocks.createIssueComment).toHaveBeenCalledTimes(1);
  expect(buttonByText("plan.review.thread.reply")?.disabled).toBe(true);
  act(() => {container.querySelector("textarea")!.dispatchEvent(new KeyboardEvent("keydown", {key: "Enter", [modifier]: true, bubbles: true}));});
  expect(mocks.createIssueComment).toHaveBeenCalledTimes(1);
  await act(async () => {complete(comment("reply", "My reply", {root: rootName}));});
  expect(onReplyDraftChange).toHaveBeenCalledWith(expect.any(Function));
});


test("disables Reply on an empty draft, checkbox or not", () => {
  const [thread] = groupThreads([comment("root", "Root")]);
  const props = {issue, project, thread, collapsible: false, onReplyDraftChange: vi.fn()};
  render(<CommentThreadCard {...props} replyDraft="   " />);
  click(stateCheckbox());
  expect(buttonByText("plan.review.thread.reply")?.disabled).toBe(true);
  act(() => {container.querySelector("textarea")!.dispatchEvent(new KeyboardEvent("keydown", {key: "Enter", metaKey: true, bubbles: true}));});
  expect(mocks.createIssueComment).not.toHaveBeenCalled();
  expect(mocks.updateIssueComment).not.toHaveBeenCalled();
  render(<CommentThreadCard {...props} replyDraft="Now with a comment" />);
  expect(buttonByText("plan.review.thread.reply")?.disabled).toBe(false);
});


test.each([
  { remount: true, nextDraft: "My next reply", expected: "My next reply" },
  { remount: true, nextDraft: "First reply", expected: "" },
  { remount: false, nextDraft: "My next reply", expected: "My next reply" },
])("clears only the submitted draft: $remount / $nextDraft", async ({remount, nextDraft, expected}) => {
  const [thread] = groupThreads([comment("root", "Root")]);
  let finish!: (value: ReturnType<typeof comment>) => void;
  let changeDraft!: (value: string) => void;
  mocks.createIssueComment.mockReturnValueOnce(new Promise(resolve => { finish = resolve; }));
  function DraftOwner({ visible }: { visible: boolean }) {
    const [draft, setDraft] = useState("First reply");
    changeDraft = setDraft;
    return visible ? <CommentThreadCard issue={issue} project={project} thread={thread} collapsible={false} replyDraft={draft} onReplyDraftChange={setDraft} /> : null;
  }
  render(<DraftOwner visible />);
  click(buttonByText("plan.review.thread.reply"));
  expect(mocks.createIssueComment).toHaveBeenCalledTimes(1);
  if (remount) {
    render(<DraftOwner visible={false} />);
    render(<DraftOwner visible />);
  }
  act(() => changeDraft(nextDraft));
  await act(async () => { finish(comment("posted", "First reply", {root:rootName})); });
  expect(container.querySelector("textarea")?.value).toBe(expected);
});


test("focuses and reveals the reply footer once when opened in Monaco", () => {
  const [thread] = groupThreads([comment("root", "Root")]);
  const reveal = vi.fn();
  const view = (draft: string) => <MonacoViewZoneRevealContext value={reveal}><CommentThreadCard issue={issue} project={project} thread={thread} replyDraft={draft} onReplyDraftChange={vi.fn()} /></MonacoViewZoneRevealContext>;
  render(view(""));
  expect(reveal).not.toHaveBeenCalled();
  click(buttonByText("plan.review.thread.reply-placeholder"));
  expect(document.activeElement).toBe(container.querySelector("textarea"));
  expect(reveal).toHaveBeenCalledExactlyOnceWith(container.querySelector('[data-testid="thread-reply-footer"]'));
  expect(HTMLElement.prototype.scrollIntoView).not.toHaveBeenCalled();
  render(view("Typing a reply"));
  expect(reveal).toHaveBeenCalledTimes(1);
});
