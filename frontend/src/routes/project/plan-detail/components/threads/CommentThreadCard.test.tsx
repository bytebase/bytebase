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
} from "@/types/proto-es/v1/issue_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

const mocks = vi.hoisted(() => ({
  sqlReviewV2: false,
  createIssueComment: vi.fn(),
  notify: vi.fn(),
  updateIssueComment: vi.fn(),
  hasPermission: vi.fn((_project: unknown, _permission: unknown) => true),
}));

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

import { CommentThreadCard, type ThreadBarProps } from "./CommentThreadCard";
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

// A bar whose owner keeps the card unfolded, for behavior the place does not
// change.
const OPEN_BAR = { place: "changes" as const, title: "Comment on line 3", foldLabel: "Comment on line 3", folded: false, onToggleFold: () => {} };

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
    render(<CommentThreadCard bar={OPEN_BAR} {...props} issue={create(IssueSchema, { ...issue, draft: true })} />);
    expect(container.textContent).toContain("Root question");
    expect(container.textContent).toContain("Existing reply");
    expect(buttonByText("plan.review.thread.reply-placeholder")).toBeUndefined();

    render(<CommentThreadCard bar={OPEN_BAR} {...props} issue={issue} replyDraft="New reply" />);
    click(buttonByText("plan.review.thread.reply-placeholder"));
    mocks.createIssueComment.mockResolvedValueOnce(comment("new-reply", "New reply", { root: rootName }));
    submitReply();
    await act(async () => {});
    expect(mocks.createIssueComment).toHaveBeenCalledExactlyOnceWith({ issueName: ISSUE, comment: "New reply", root: rootName });
  });

  test("under SQL Review V2 a draft takes replies", () => {
    mocks.sqlReviewV2 = true;
    const [thread] = groupThreads([comment("root", "Root question")]);
    render(<CommentThreadCard bar={OPEN_BAR} project={project} thread={thread} issue={create(IssueSchema, { ...issue, draft: true })} />);
    expect(buttonByText("plan.review.thread.reply-placeholder")).toBeDefined();
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
      <CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} />
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
      <CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} />
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

  test("indents replies by the avatar column and joins them to the first comment by a line", () => {
    const [thread] = groupThreads([comment("root", "Root"), comment("r1", "Reply", { root: rootName })]);
    render(<CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} />);
    const [first, reply] = Array.from(container.querySelectorAll("[data-testid='thread-comment']"));
    const replies = container.querySelector("[data-testid='thread-replies']");
    expect(replies?.contains(first)).toBe(false);
    expect(replies?.contains(reply)).toBe(true);
    expect(replies?.className).toContain("ml-7");
    expect(replies?.className).not.toContain("bg-control-bg");
    const rail = container.querySelector("[data-testid='thread-replies-rail']");
    expect(rail?.getAttribute("aria-orientation")).toBe("vertical");
    for (const node of [first, reply]) {
      const avatar = node.querySelector("[data-testid='comment-avatar']");
      expect(avatar?.nextElementSibling?.getAttribute("data-testid")).toBe("comment-content");
    }

    render(<CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={groupThreads([comment("root", "Root")])[0]} />);
    expect(container.querySelector("[data-testid='thread-replies']")).toBeNull();
    expect(container.querySelector("[data-testid='thread-replies-rail']")).toBeNull();
  });

  test("the reply composer shows the reader's avatar only while open", () => {
    const [thread] = groupThreads([comment("root", "Root")]);
    render(<CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} />);
    expect(container.querySelector("[data-testid='reply-composer-avatar']")).toBeNull();
    click(buttonByText("plan.review.thread.reply-placeholder"));
    expect(container.querySelector("[data-testid='reply-composer-avatar']")).not.toBeNull();
    click(buttonByText("common.cancel"));
    expect(container.querySelector("[data-testid='reply-composer-avatar']")).toBeNull();
  });

  test("new replies do not hide replies already visible in a short thread", () => {
    const replies = Array.from({ length: 6 }, (_, index) => comment(`r${index}`, `Reply ${index}`, { root: rootName }));
    const [short] = groupThreads([comment("root", "Root"), ...replies.slice(0, 4)]);
    render(<CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={short} />);
    expect(container.querySelectorAll("[data-testid='preview']")).toHaveLength(5);
    const [long] = groupThreads([comment("root", "Root"), ...replies]);
    render(<CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={long} />);
    expect(container.querySelectorAll("[data-testid='preview']")).toHaveLength(7);
  });

  test("folds long threads and shows the tail until expanded", () => {
    const replies = Array.from({ length: 6 }, (_, i) =>
      comment(`r${i}`, `Reply ${i}`, { root: rootName })
    );
    const [thread] = groupThreads([comment("root", "Root"), ...replies]);
    render(
      <CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} />
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
      <CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} />
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
    render(<CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} onThreadStateChanged={onThreadStateChanged} />);
    click(buttonByText(resolved ? "common.reopen" : "plan.review.thread.resolve"));
    await act(async () => {});
    expect(onThreadStateChanged).toHaveBeenCalledExactlyOnceWith(!resolved);
  });

  test("does not advance on failure or a remote state refresh", async () => {
    const [thread] = groupThreads([comment("root", "Root")]);
    const onThreadStateChanged = vi.fn();
    mocks.updateIssueComment.mockRejectedValueOnce(new Error("offline"));
    render(<CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} onThreadStateChanged={onThreadStateChanged} />);
    click(buttonByText("plan.review.thread.resolve"));
    await act(async () => {});
    expect(onThreadStateChanged).not.toHaveBeenCalled();
    const [updated] = groupThreads([comment("root", "Root", {resolved: true})]);
    render(<CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={updated} onThreadStateChanged={onThreadStateChanged} />);
    expect(onThreadStateChanged).not.toHaveBeenCalled();
    expect(container.querySelector("[data-testid='preview']")).not.toBeNull();
  });

  test("does not advance if the user leaves the card before Resolve completes", async () => {
    const [thread] = groupThreads([comment("root", "Root")]);
    const onThreadStateChanged = vi.fn();
    let complete!: (value: ReturnType<typeof comment>) => void;
    mocks.updateIssueComment.mockReturnValueOnce(new Promise((resolve) => {complete = resolve;}));
    render(<CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} onThreadStateChanged={onThreadStateChanged} />);
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
    render(<CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} replyDraft="Decision details" onReplyDraftChange={onReplyDraftChange} onThreadStateChanged={onThreadStateChanged} />);
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
    render(<CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} replyDraft="Draft" onReplyDraftChange={vi.fn()} />);
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
    render(<CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} replyDraft="Just a note" onReplyDraftChange={vi.fn()} onThreadStateChanged={onThreadStateChanged} />);
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
    const props = { issue, project, thread, onReplyDraftChange: vi.fn(), onThreadStateChanged };
    render(<CommentThreadCard bar={OPEN_BAR} {...props} replyDraft="Not done yet" />);
    expect(stateCheckbox()?.getAttribute("aria-checked")).toBe("true");
    submitReply();
    await act(async () => {});
    expect(mocks.updateIssueComment).toHaveBeenCalledExactlyOnceWith({ issueCommentName: rootName, threadState: IssueComment_ThreadState.OPEN });
    expect(onThreadStateChanged).toHaveBeenCalledExactlyOnceWith(false);

    vi.clearAllMocks();
    mocks.hasPermission.mockReturnValue(true);
    mocks.createIssueComment.mockResolvedValue(comment("reply", "Still resolved", { root: rootName }));
    render(<CommentThreadCard bar={OPEN_BAR} {...props} replyDraft="Still resolved" />);
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
    render(<CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} replyDraft="Draft" onReplyDraftChange={vi.fn()} />);
    expect(stateCheckbox()).toBeNull();
    expect(buttonByText("plan.review.thread.reply")).toBeDefined();
  });

  test("reply failure preserves the draft and never changes status", async () => {
    const [thread] = groupThreads([comment("root", "Root")]);
    const onReplyDraftChange = vi.fn();
    mocks.createIssueComment.mockRejectedValueOnce(new Error("offline"));
    render(<CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} replyDraft="Keep this draft" onReplyDraftChange={onReplyDraftChange} />);
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
    render(<CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} replyDraft="Posted" onReplyDraftChange={onReplyDraftChange} onThreadStateChanged={onThreadStateChanged} />);
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
      <CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} />
    );
    expect(buttonByText("plan.review.thread.resolve")).toBeUndefined();
  });

  test("hides Resolve from users who may neither update nor own the root", () => {
    denySettle();
    const [thread] = groupThreads([comment("root", "Root")]);
    render(
      <CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} />
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
  render(<CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} replyDraft="My reply" onReplyDraftChange={onReplyDraftChange} />);
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
  const props = {issue, project, thread, onReplyDraftChange: vi.fn()};
  render(<CommentThreadCard bar={OPEN_BAR} {...props} replyDraft="   " />);
  click(stateCheckbox());
  expect(buttonByText("plan.review.thread.reply")?.disabled).toBe(true);
  act(() => {container.querySelector("textarea")!.dispatchEvent(new KeyboardEvent("keydown", {key: "Enter", metaKey: true, bubbles: true}));});
  expect(mocks.createIssueComment).not.toHaveBeenCalled();
  expect(mocks.updateIssueComment).not.toHaveBeenCalled();
  render(<CommentThreadCard bar={OPEN_BAR} {...props} replyDraft="Now with a comment" />);
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
    return visible ? <CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} replyDraft={draft} onReplyDraftChange={setDraft} /> : null;
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
  const view = (draft: string) => <MonacoViewZoneRevealContext value={reveal}><CommentThreadCard bar={OPEN_BAR} issue={issue} project={project} thread={thread} replyDraft={draft} onReplyDraftChange={vi.fn()} /></MonacoViewZoneRevealContext>;
  render(view(""));
  expect(reveal).not.toHaveBeenCalled();
  click(buttonByText("plan.review.thread.reply-placeholder"));
  expect(document.activeElement).toBe(container.querySelector("textarea"));
  expect(reveal).toHaveBeenCalledExactlyOnceWith(container.querySelector('[data-testid="thread-reply-footer"]'));
  expect(HTMLElement.prototype.scrollIntoView).not.toHaveBeenCalled();
  render(view("Typing a reply"));
  expect(reveal).toHaveBeenCalledTimes(1);
});

describe("title bar", () => {
  const bar = (folded: boolean, onToggleFold = vi.fn()) => ({
    place: "changes" as const,
    title: "plan.review.thread.comment-title:7",
    foldLabel: "plan.review.thread.comment-title:7",
    folded,
    onToggleFold,
  });
  const findingThread = (resolved = false) => {
    const root = comment("root", "Missing WHERE clause.", { creator: "", resolved });
    root.reviewMetadata = create(IssueComment_ReviewMetadataSchema, {
      priority: IssueComment_ReviewMetadata_Priority.P1,
      title: "Add a WHERE clause",
    });
    return groupThreads([root, comment("r1", "A reply", { root: rootName })])[0];
  };
  const barNode = () => container.querySelector("[data-testid='thread-bar']");
  const foldControl = () => barNode()?.querySelector("button");

  test("is the same folded and unfolded, and a folded card shows nothing from the body", () => {
    const thread = findingThread(true);
    render(<CommentThreadCard bar={bar(false)} issue={issue} project={project} thread={thread} />);
    const unfolded = barNode()?.textContent;
    expect(container.querySelector("[data-testid='thread-comment']")).not.toBeNull();

    render(<CommentThreadCard bar={bar(true)} issue={issue} project={project} thread={thread} />);
    expect(barNode()?.textContent).toBe(unfolded);
    expect(unfolded).toBe("plan.review.thread.comment-title:7P1plan.review.thread.resolved");
    const card = container.querySelector("[data-testid='comment-thread']");
    expect(card?.children).toHaveLength(1);
    expect(card?.textContent).not.toContain("sql-review.reviewer");
    expect(card?.textContent).not.toContain("Add a WHERE clause");
    expect(card?.textContent).not.toContain("A reply");
    // The bar says how urgent the thread is; nothing else of the finding shows.
    expect(card?.querySelectorAll("[data-testid='review-priority']")).toHaveLength(1);
    expect(barNode()?.querySelector("[data-testid='review-priority']")).not.toBeNull();
    expect(card?.querySelector("[data-testid='thread-reply-footer']")).toBeNull();
  });

  test("names its fold control by the title and reports the click", () => {
    const onToggleFold = vi.fn();
    render(<CommentThreadCard bar={bar(true, onToggleFold)} issue={issue} project={project} thread={findingThread()} />);
    expect(foldControl()?.getAttribute("aria-label")).toBe("plan.review.thread.comment-title:7");
    expect(foldControl()?.getAttribute("aria-expanded")).toBe("false");
    click(foldControl());
    expect(onToggleFold).toHaveBeenCalledOnce();
    // The owner keeps the fold state: the card stays folded until it says so.
    expect(container.querySelector("[data-testid='thread-comment']")).toBeNull();

    render(<CommentThreadCard bar={bar(false, onToggleFold)} issue={issue} project={project} thread={findingThread()} />);
    expect(foldControl()?.getAttribute("aria-expanded")).toBe("true");
    expect(container.querySelector('button[aria-label="common.close"]')).toBeNull();
  });

  test("a click on the bar's title folds and unfolds; the fold control folds once; a drag that selects text does not", () => {
    const onToggleFold = vi.fn();
    render(<CommentThreadCard bar={bar(false, onToggleFold)} issue={issue} project={project} thread={findingThread()} />);
    const title = foldControl()?.nextElementSibling as HTMLElement;
    click(title);
    expect(onToggleFold).toHaveBeenCalledTimes(1);
    click(barNode());
    expect(onToggleFold).toHaveBeenCalledTimes(2);
    // The fold control is still the one control that folds, never twice.
    click(foldControl());
    expect(onToggleFold).toHaveBeenCalledTimes(3);
    // A drag that ends with selected text in the bar does not fold it.
    const range = document.createRange();
    range.selectNodeContents(title);
    window.getSelection()?.removeAllRanges();
    window.getSelection()?.addRange(range);
    click(title);
    expect(onToggleFold).toHaveBeenCalledTimes(3);
    window.getSelection()?.removeAllRanges();
    // The bar adds no focus stop of its own.
    expect(barNode()?.getAttribute("tabindex")).toBeNull();
    expect(barNode()?.getAttribute("role")).toBeNull();
  });

  test("says Resolved in the bar, never in the footer, and nothing for an open thread", () => {
    render(<CommentThreadCard bar={bar(false)} issue={issue} project={project} thread={findingThread(true)} />);
    const states = container.querySelectorAll("[data-testid='thread-state']");
    expect(states).toHaveLength(1);
    expect(barNode()?.contains(states[0])).toBe(true);
    // State labels follow the title in every place: the title does not stretch to fill the bar.
    const lead = barNode()?.querySelector("[data-testid='thread-bar-lead']");
    const title = lead?.firstElementChild;
    expect(title?.className.split(" ")).toContain("max-w-max");
    // Title, then the markers (this P1 finding's badge), then the state.
    expect(title?.nextElementSibling?.getAttribute("data-testid")).toBe("review-priority");
    expect(title?.nextElementSibling?.nextElementSibling).toBe(states[0]);
    expect(lead?.nextElementSibling).toBeNull();

    render(<CommentThreadCard bar={bar(false)} issue={issue} project={project} thread={findingThread(false)} />);
    expect(container.querySelector("[data-testid='thread-state']")).toBeNull();
  });

  test("carries no text-control-light, folded or not", () => {
    for (const folded of [false, true]) {
      render(<CommentThreadCard bar={bar(folded)} issue={issue} project={project} thread={findingThread(true)} />);
      const nodes = [barNode()!, ...barNode()!.querySelectorAll("*")];
      expect(nodes.filter((node) => node.getAttribute("class")?.includes("text-control-light"))).toEqual([]);
      expect(barNode()?.className).toContain("text-control");
    }
  });

  test("truncates a text title with an ellipsis and keeps its full text on hover", () => {
    render(<CommentThreadCard bar={bar(false)} issue={issue} project={project} thread={findingThread(false)} />);
    const title = (barNode() as HTMLElement).querySelector("[data-testid='thread-bar-lead'] > span") as HTMLElement;
    expect(title.textContent).toBe("plan.review.thread.comment-title:7");
    expect(title.getAttribute("title")).toBe("plan.review.thread.comment-title:7");
    expect(title.className.split(" ")).toContain("truncate");
    expect(title.className.split(" ")).not.toContain("flex");
  });

  test.each(["changes", "activity"] as const)("draws the bar and the footer white in %s, with no gray of their own", (place) => {
    render(<CommentThreadCard bar={{ ...bar(false), place }} issue={issue} project={project} thread={findingThread(false)} />);
    const footer = container.querySelector("[data-testid='thread-reply-footer']") as HTMLElement;
    for (const part of [barNode() as HTMLElement, footer]) {
      expect(part.className.split(" ")).toContain("bg-background");
      expect(part.className).not.toMatch(/\bbg-control-bg/);
    }
  });

  test.each([
    { reply: false, settle: false, footer: false },
    { reply: true, settle: false, footer: true },
    { reply: false, settle: true, footer: true },
  ])("has a footer only when the reader can act: %o", ({ reply, settle, footer }) => {
    mocks.hasPermission.mockImplementation((_project: unknown, permission: unknown) =>
      permission === "bb.issueComments.create" ? reply : settle
    );
    render(<CommentThreadCard bar={bar(false)} issue={issue} project={project} thread={findingThread(true)} />);
    expect(container.querySelector("[data-testid='thread-reply-footer']") !== null).toBe(footer);
  });
});

describe("priority in the bar", () => {
  const finding = (priority: IssueComment_ReviewMetadata_Priority) => {
    const root = comment("root", "Drops a table.", { creator: "" });
    root.reviewMetadata = create(IssueComment_ReviewMetadataSchema, { priority });
    return groupThreads([root])[0];
  };
  const changesBar = (folded: boolean) => ({ place: "changes" as const, title: "Comment on line 3", foldLabel: "Comment on line 3", folded, onToggleFold: () => {} });
  const barBadge = () => container.querySelector("[data-testid='thread-bar'] [data-testid='review-priority']");

  test.each([
    { priority: IssueComment_ReviewMetadata_Priority.P0, label: "P0" },
    { priority: IssueComment_ReviewMetadata_Priority.P1, label: "P1" },
  ])("a $label finding shows its badge in the bar, folded and unfolded", ({ priority, label }) => {
    for (const folded of [true, false]) {
      render(<CommentThreadCard bar={changesBar(folded)} issue={issue} project={project} thread={finding(priority)} />);
      expect(barBadge()?.textContent).toBe(label);
    }
  });

  test("a P2 finding and a person's thread show no badge in the bar", () => {
    render(<CommentThreadCard bar={changesBar(true)} issue={issue} project={project} thread={finding(IssueComment_ReviewMetadata_Priority.P2)} />);
    expect(barBadge()).toBeNull();
    render(<CommentThreadCard bar={changesBar(true)} issue={issue} project={project} thread={groupThreads([comment("root", "A question")])[0]} />);
    expect(barBadge()).toBeNull();
  });
});

describe("Activity bar", () => {
  const activityBar = (extra: Partial<ThreadBarProps> = {}): ThreadBarProps => ({
    place: "activity",
    title: "alpha_prod",
    foldLabel: "alpha_prod, lines 9–10",
    lines: { startLine: 9, endLine: 10 },
    ...extra,
  });
  const thread = (resolved = false) => groupThreads([comment("root", "Root question", { resolved })])[0];
  const barNode = () => container.querySelector("[data-testid='thread-bar']") as HTMLElement;
  const foldControl = () => barNode().querySelector("button[aria-expanded]") as HTMLButtonElement;
  const unfolded = () => container.querySelector("[data-testid='thread-comment']") !== null;

  test("shows the lines on the right, then View in Statement in the link color", () => {
    const onViewInStatement = vi.fn();
    render(
      <CommentThreadCard bar={activityBar({ anchorState: "CURRENT", onViewInStatement })} issue={issue} project={project} thread={thread()} />
    );
    const lines = barNode().querySelector("[data-testid='thread-bar-lines']");
    const link = buttonByText("plan.review.thread.anchor.view-in-statement");
    expect(lines?.textContent).toBe("plan.review.thread.anchor.line:2");
    expect(lines?.compareDocumentPosition(link!)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
    expect(link?.className).toContain("text-accent");
    click(link);
    expect(onViewInStatement).toHaveBeenCalledOnce();
    expect(unfolded()).toBe(true);
  });

  test.each([
    { state: "OUTDATED", resolved: true, labels: ["plan.review.thread.resolved", "plan.review.thread.anchor.outdated"] },
    { state: "UNAVAILABLE", resolved: false, labels: ["plan.review.thread.anchor.statement-unavailable"] },
    { state: "PENDING", resolved: false, labels: [] },
    { state: "CURRENT", resolved: false, labels: [] },
  ] as const)("a $state thread, resolved=$resolved, says $labels with no light gray and no badge", ({ state, resolved, labels }) => {
    render(<CommentThreadCard bar={activityBar({ anchorState: state })} issue={issue} project={project} thread={thread(resolved)} />);
    expect(barNode().getAttribute("data-anchor-state")).toBe(state);
    const shown = Array.from(barNode().querySelectorAll("[data-testid='thread-state']")).map((node) => node.textContent);
    expect(shown).toEqual(labels);
    const nodes = [barNode(), ...barNode().querySelectorAll("*")];
    expect(nodes.filter((node) => node.getAttribute("class")?.includes("text-control-light"))).toEqual([]);
    expect(barNode().querySelector("[data-slot='badge']")).toBeNull();
    expect(buttonByText("plan.review.thread.anchor.view-in-statement")).toBeUndefined();
  });

  test("starts unfolded when open and folded when resolved, and the reader folds and unfolds it", () => {
    render(<CommentThreadCard bar={activityBar()} issue={issue} project={project} thread={thread(false)} />);
    expect(unfolded()).toBe(true);
    expect(foldControl().getAttribute("aria-label")).toBe("alpha_prod, lines 9–10");
    click(foldControl());
    expect(unfolded()).toBe(false);
    expect(foldControl().getAttribute("aria-expanded")).toBe("false");
    click(foldControl());
    expect(unfolded()).toBe(true);

    act(() => root.unmount());
    root = createRoot(container);
    render(<CommentThreadCard bar={activityBar()} issue={issue} project={project} thread={thread(true)} />);
    expect(unfolded()).toBe(false);
  });

  test("folds when the thread becomes resolved and unfolds when it is reopened", () => {
    render(<CommentThreadCard bar={activityBar()} issue={issue} project={project} thread={thread(false)} />);
    render(<CommentThreadCard bar={activityBar()} issue={issue} project={project} thread={thread(true)} />);
    expect(unfolded()).toBe(false);
    render(<CommentThreadCard bar={activityBar()} issue={issue} project={project} thread={thread(false)} />);
    expect(unfolded()).toBe(true);
  });

  test("draws the excerpt in the body, under the bar, and hides it when folded", () => {
    render(
      <CommentThreadCard bar={activityBar()} excerpt={<div data-testid="excerpt">SELECT 1;</div>} issue={issue} project={project} thread={thread(false)} />
    );
    const excerpt = container.querySelector("[data-testid='excerpt']");
    expect(barNode().contains(excerpt)).toBe(false);
    expect(barNode().compareDocumentPosition(excerpt!)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
    expect(excerpt?.compareDocumentPosition(container.querySelector("[data-testid='thread-comment']")!)).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING
    );
    click(foldControl());
    expect(container.querySelector("[data-testid='excerpt']")).toBeNull();
  });

  test.each([
    { place: "Activity", bar: () => activityBar() },
    { place: "the Changes section", bar: () => ({ ...activityBar(), place: "changes" as const, folded: false, onToggleFold: () => {} }) },
  ])("every card in $place has the frame, 8px corners and the darker border with no shadow, and a bar", ({ bar }) => {
    render(<CommentThreadCard bar={bar()} issue={issue} project={project} thread={thread(false)} />);
    const card = container.querySelector("[data-testid='comment-thread']");
    const classes = card?.className.split(" ") ?? [];
    for (const name of ["rounded-lg", "border", "border-control-border", "bg-background"]) {
      expect(classes).toContain(name);
    }
    expect(classes).not.toContain("rounded-sm");
    expect(classes).not.toContain("border-block-border");
    expect(classes.filter((name) => name.startsWith("shadow"))).toEqual([]);
    expect(barNode().className.split(" ")).toContain("border-block-border");
    expect(card?.firstElementChild).toBe(barNode());
  });

  test("says Outdated in the warning color and Resolved and Statement unavailable in gray, none with a fill", () => {
    render(<CommentThreadCard bar={activityBar({ anchorState: "OUTDATED" })} issue={issue} project={project} thread={thread(true)} />);
    const label = (text: string) =>
      Array.from(barNode().querySelectorAll("[data-testid='thread-state']")).find((node) => node.textContent === text) as HTMLElement;
    const outdated = label("plan.review.thread.anchor.outdated");
    expect(outdated.className.split(" ")).toContain("text-warning-hover");
    const resolved = label("plan.review.thread.resolved");
    expect(resolved.className.split(" ")).toContain("text-control");
    act(() => root.unmount());
    root = createRoot(container);
    render(<CommentThreadCard bar={activityBar({ anchorState: "UNAVAILABLE" })} issue={issue} project={project} thread={thread(false)} />);
    const unavailable = label("plan.review.thread.anchor.statement-unavailable");
    for (const node of [outdated, resolved, unavailable]) {
      expect(node.className).not.toMatch(/\bbg-/);
    }
    expect(unavailable.className.split(" ")).toContain("text-control");
    expect(unavailable.className).not.toContain("warning");
    expect(resolved.className).not.toContain("warning");
  });

  test("a folded P0 finding in Activity shows its badge in the bar", () => {
    const root = comment("root", "Drops a table.", { creator: "", resolved: true });
    root.reviewMetadata = create(IssueComment_ReviewMetadataSchema, {
      priority: IssueComment_ReviewMetadata_Priority.P0,
    });
    render(<CommentThreadCard bar={activityBar()} issue={issue} project={project} thread={groupThreads([root])[0]} />);
    expect(foldControl().getAttribute("aria-expanded")).toBe("false");
    expect(barNode().querySelector("[data-testid='review-priority']")?.textContent).toBe("P0");
  });

  test("in Activity only the title, markers and state labels show the pointer; the rest of the bar does not", () => {
    render(<CommentThreadCard bar={activityBar({ anchorState: "CURRENT", onViewInStatement: () => {} })} issue={issue} project={project} thread={thread(true)} />);
    expect(barNode().className).not.toContain("cursor-pointer");
    const lead = barNode().querySelector("[data-testid='thread-bar-lead']") as HTMLElement;
    expect(lead.className).toContain("cursor-pointer");
    expect(lead.querySelector("[data-testid='thread-state']")).not.toBeNull();
    expect(lead.contains(barNode().querySelector("[data-testid='thread-bar-lines']"))).toBe(false);
  });

  test("in the Changes section the whole bar shows the pointer", () => {
    render(<CommentThreadCard bar={{ ...activityBar(), place: "changes", folded: false, onToggleFold: () => {} }} issue={issue} project={project} thread={thread(false)} />);
    expect(barNode().className).toContain("cursor-pointer");
  });

  test("in Activity a click on the change's label or a state label folds, and one on the lines, the link or the gap does not", () => {
    const onViewInStatement = vi.fn();
    render(
      <CommentThreadCard
        bar={activityBar({ title: <span data-testid="change-label">alpha_prod</span>, anchorState: "CURRENT", onViewInStatement })}
        issue={issue}
        project={project}
        thread={thread(true)}
      />
    );
    const folded = () => foldControl().getAttribute("aria-expanded") === "false";
    expect(folded()).toBe(true);
    click(container.querySelector("[data-testid='change-label']"));
    expect(folded()).toBe(false);
    click(barNode().querySelector("[data-testid='thread-state']"));
    expect(folded()).toBe(true);
    click(barNode().querySelector("[data-testid='thread-bar-lines']"));
    expect(folded()).toBe(true);
    click(barNode());
    expect(folded()).toBe(true);
    click(buttonByText("plan.review.thread.anchor.view-in-statement"));
    expect(onViewInStatement).toHaveBeenCalledOnce();
    expect(folded()).toBe(true);
  });

  test("puts the state labels after the title and the lines and link together at the right end", () => {
    render(<CommentThreadCard bar={activityBar({ anchorState: "CURRENT", onViewInStatement: () => {} })} issue={issue} project={project} thread={thread(true)} />);
    const state = barNode().querySelector("[data-testid='thread-state']") as HTMLElement;
    const end = state.closest("[data-testid='thread-bar-lead']")?.nextElementSibling as HTMLElement;
    expect(end.getAttribute("data-testid")).toBe("thread-bar-end");
    expect(end.className.split(" ")).toContain("ml-auto");
    expect(end.firstElementChild?.getAttribute("data-testid")).toBe("thread-bar-lines");
    expect(end.textContent).toContain("plan.review.thread.anchor.view-in-statement");
  });

  test("a resolved thread whose reader can neither reply nor reopen has no footer", () => {
    mocks.hasPermission.mockReturnValue(false);
    render(<CommentThreadCard bar={activityBar()} issue={issue} project={project} thread={thread(true)} />);
    click(foldControl());
    expect(unfolded()).toBe(true);
    expect(container.querySelector("[data-testid='thread-reply-footer']")).toBeNull();
  });
});
