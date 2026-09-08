import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { act, useState } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MonacoViewZoneRevealContext } from "@/components/monaco/MonacoViewZone";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  IssueComment_ThreadState,
  IssueCommentSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

const mocks = vi.hoisted(() => ({
  createIssueComment: vi.fn(),
  notify: vi.fn(),
  updateIssueComment: vi.fn(),
  hasPermission: vi.fn((_project: unknown, _permission: unknown) => true),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, options?: { count?: number }) =>
      options?.count !== undefined ? `${key}:${options.count}` : key,
  }),
}));

vi.mock("@/components/HumanizeTs", () => ({ HumanizeTs: () => null }));

// The activity module drags Monaco into the test; only the edit rule matters.
vi.mock("@/components/issue-activity/IssueCommentActivity", () => ({
  canEditIssueComment: () => true,
}));

vi.mock("@/components/MarkdownEditor", () => ({
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
  useUserByIdentifier: (principal: string) => ({
    name: principal,
    email: principal.replace(/^users\//, ""),
    title: principal.replace(/^users\//, "").split("@")[0],
  }),
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

import { CommentThreadCard } from "./CommentThreadCard";
import { groupThreads } from "./threadModel";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const ISSUE = "projects/p/issues/1";
const project = { name: "projects/p" } as Project;

const comment = (
  id: string,
  text: string,
  extra: { root?: string; resolved?: boolean; creator?: string } = {}
) =>
  create(IssueCommentSchema, {
    name: `${ISSUE}/issueComments/${id}`,
    comment: text,
    creator: extra.creator ?? "users/alice@example.com",
    createTime: create(TimestampSchema, { seconds: BigInt(1) }),
    updateTime: create(TimestampSchema, { seconds: BigInt(1) }),
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

const chooseReplyAction = (action = "reply") => {
  click(container.querySelector('button[aria-label="common.actions"]'));
  const item = Array.from(document.querySelectorAll('[role="menuitem"]'))
    .find((el) => el.textContent === `plan.review.thread.${action}`);
  click(item);
  expect(mocks.createIssueComment).not.toHaveBeenCalled();
  expect(mocks.updateIssueComment).not.toHaveBeenCalled();
  click(buttonByText(`plan.review.thread.${action}`));
};

describe("CommentThreadCard", () => {
  test("places statement context above the root author as a separate full-width region", () => {
    const [thread] = groupThreads([comment("root", "Root question")]);
    render(
      <CommentThreadCard
        issueName={ISSUE}
        project={project}
        thread={thread}
        context={<div data-testid="anchor-context">SELECT 1;</div>}
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

  test("renders the root and replies oldest first with the thread footer", () => {
    const [thread] = groupThreads([
      comment("root", "Root question"),
      comment("r2", "Second reply", { root: rootName }),
      comment("r1", "First reply", { root: rootName }),
    ]);
    render(
      <CommentThreadCard issueName={ISSUE} project={project} thread={thread} />
    );
    const bodies = Array.from(
      container.querySelectorAll("[data-testid='preview']")
    ).map((node) => node.textContent);
    expect(bodies).toEqual(["Root question", "First reply", "Second reply"]);
    expect(container.querySelector("[data-thread-state]")?.getAttribute("data-thread-state")).toBe("open");
    expect(buttonByText("plan.review.thread.resolve")).toBeDefined();
    expect(buttonByText("plan.review.thread.reply-placeholder")).toBeDefined();
  });

  test("a resolved thread collapses to a summary and expands in place", () => {
    const [thread] = groupThreads([
      comment("root", "Root", { resolved: true }),
      comment("r1", "Reply", { root: rootName, creator: "users/bob@example.com" }),
    ]);
    render(
      <CommentThreadCard issueName={ISSUE} project={project} thread={thread} />
    );
    expect(container.textContent).toContain("plan.review.thread.n-replies:1");
    expect(container.querySelector("[data-testid='preview']")).toBeNull();

    click(buttonByText("plan.review.thread.resolved"));
    expect(
      container.querySelectorAll("[data-testid='preview']")
    ).toHaveLength(2);
    expect(buttonByText("common.reopen")).toBeDefined();
    expect(buttonByText("plan.review.thread.resolve")).toBeUndefined();
  });

  test("roots and replies share header avatars and a full-width body", () => {
    const [thread] = groupThreads([comment("root", "Root"), comment("r1", "Reply", { root: rootName })]);
    render(<CommentThreadCard issueName={ISSUE} project={project} thread={thread} />);
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
    render(<CommentThreadCard issueName={ISSUE} project={project} thread={thread} />);
    expect(container.querySelector("[data-testid='thread-comment'] [data-testid='comment-avatar']")).not.toBeNull();
  });

  test("new replies do not hide replies already visible in a short thread", () => {
    const replies = Array.from({ length: 6 }, (_, index) => comment(`r${index}`, `Reply ${index}`, { root: rootName }));
    const [short] = groupThreads([comment("root", "Root"), ...replies.slice(0, 4)]);
    render(<CommentThreadCard issueName={ISSUE} project={project} thread={short} />);
    expect(container.querySelectorAll("[data-testid='preview']")).toHaveLength(5);
    const [long] = groupThreads([comment("root", "Root"), ...replies]);
    render(<CommentThreadCard issueName={ISSUE} project={project} thread={long} />);
    expect(container.querySelectorAll("[data-testid='preview']")).toHaveLength(7);
  });

  test("folds long threads and shows the tail until expanded", () => {
    const replies = Array.from({ length: 6 }, (_, i) =>
      comment(`r${i}`, `Reply ${i}`, { root: rootName })
    );
    const [thread] = groupThreads([comment("root", "Root"), ...replies]);
    render(
      <CommentThreadCard issueName={ISSUE} project={project} thread={thread} />
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
      <CommentThreadCard issueName={ISSUE} project={project} thread={thread} />
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
    chooseReplyAction();
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
    render(<CommentThreadCard issueName={ISSUE} project={project} thread={thread} collapsible={false} onThreadStateChanged={onThreadStateChanged} />);
    click(buttonByText(resolved ? "common.reopen" : "plan.review.thread.resolve"));
    await act(async () => {});
    expect(onThreadStateChanged).toHaveBeenCalledExactlyOnceWith(!resolved);
  });

  test("does not advance on failure or a remote state refresh", async () => {
    const [thread] = groupThreads([comment("root", "Root")]);
    const onThreadStateChanged = vi.fn();
    mocks.updateIssueComment.mockRejectedValueOnce(new Error("offline"));
    render(<CommentThreadCard issueName={ISSUE} project={project} thread={thread} collapsible={false} onThreadStateChanged={onThreadStateChanged} />);
    click(buttonByText("plan.review.thread.resolve"));
    await act(async () => {});
    expect(onThreadStateChanged).not.toHaveBeenCalled();
    const [updated] = groupThreads([comment("root", "Root", {resolved: true})]);
    render(<CommentThreadCard issueName={ISSUE} project={project} thread={updated} collapsible={false} onThreadStateChanged={onThreadStateChanged} />);
    expect(onThreadStateChanged).not.toHaveBeenCalled();
    expect(container.querySelector("[data-testid='preview']")).not.toBeNull();
  });

  test("does not advance if the user leaves the card before Resolve completes", async () => {
    const [thread] = groupThreads([comment("root", "Root")]);
    const onThreadStateChanged = vi.fn();
    let complete!: (value: ReturnType<typeof comment>) => void;
    mocks.updateIssueComment.mockReturnValueOnce(new Promise((resolve) => {complete = resolve;}));
    render(<CommentThreadCard issueName={ISSUE} project={project} thread={thread} collapsible={false} onThreadStateChanged={onThreadStateChanged} />);
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
    render(<CommentThreadCard issueName={ISSUE} project={project} thread={thread} collapsible={false} replyDraft="Decision details" onReplyDraftChange={onReplyDraftChange} onThreadStateChanged={onThreadStateChanged} />);
    chooseReplyAction(resolved ? "reopen-with-comment" : "resolve-with-comment");
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

  test("reply failure preserves the draft and never changes status", async () => {
    const [thread] = groupThreads([comment("root", "Root")]);
    const onReplyDraftChange = vi.fn();
    mocks.createIssueComment.mockRejectedValueOnce(new Error("offline"));
    render(<CommentThreadCard issueName={ISSUE} project={project} thread={thread} replyDraft="Keep this draft" onReplyDraftChange={onReplyDraftChange} />);
    chooseReplyAction("resolve-with-comment");
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
    render(<CommentThreadCard issueName={ISSUE} project={project} thread={thread} replyDraft="Posted" onReplyDraftChange={onReplyDraftChange} onThreadStateChanged={onThreadStateChanged} />);
    chooseReplyAction("resolve-with-comment");
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

  test("without settle permission the submission menu only offers Reply", () => {
    mocks.hasPermission.mockImplementation((_project, permission) => permission === "bb.issueComments.create");
    const [thread] = groupThreads([comment("root", "Root")]);
    render(<CommentThreadCard issueName={ISSUE} project={project} thread={thread} replyDraft="Reply only" onReplyDraftChange={vi.fn()} />);
    click(container.querySelector('button[aria-label="common.actions"]'));
    expect(Array.from(document.querySelectorAll('[role="menuitem"]')).map((el) => el.textContent)).toEqual(["plan.review.thread.reply"]);
  });

  test("hides Resolve from users who may neither update nor own the root", () => {
    mocks.hasPermission.mockImplementation(
      (_project: unknown, permission: unknown) =>
        permission === "bb.issueComments.create"
    );
    const [thread] = groupThreads([comment("root", "Root")]);
    render(
      <CommentThreadCard issueName={ISSUE} project={project} thread={thread} />
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
  render(<CommentThreadCard issueName={ISSUE} project={project} thread={thread} replyDraft="My reply" onReplyDraftChange={onReplyDraftChange} />);
  chooseReplyAction();
  expect(mocks.createIssueComment).toHaveBeenCalledTimes(1);
  expect(buttonByText("plan.review.thread.reply")?.disabled).toBe(true);
  act(() => {container.querySelector("textarea")!.dispatchEvent(new KeyboardEvent("keydown", {key: "Enter", [modifier]: true, bubbles: true}));});
  expect(mocks.createIssueComment).toHaveBeenCalledTimes(1);
  await act(async () => {complete(comment("reply", "My reply", {root: rootName}));});
  expect(onReplyDraftChange).toHaveBeenCalledWith(expect.any(Function));
});


test.each(["reply", "resolve-with-comment", "reopen-with-comment"])("validates an empty comment after clicking %s", (action) => {
  const [thread] = groupThreads([comment("root", "Root", {resolved: action === "reopen-with-comment"})]);
  const props = {issueName: ISSUE, project, thread, collapsible: false, onReplyDraftChange: vi.fn()};
  render(<CommentThreadCard {...props} replyDraft="   " />);
  expect(document.querySelector('[role="alert"]')).toBeNull();
  chooseReplyAction(action);
  expect(buttonByText(`plan.review.thread.${action}`)?.disabled).toBe(false);
  expect(document.querySelector('[role="alert"]')?.textContent).toBe("plan.review.thread.comment-required");
  expect(container.querySelector('[role="alert"]')).toBeNull();
  expect(mocks.createIssueComment).not.toHaveBeenCalled();
  expect(mocks.updateIssueComment).not.toHaveBeenCalled();
  render(<CommentThreadCard {...props} replyDraft="Now with a comment" />);
  expect(document.querySelector('[role="alert"]')).toBeNull();
  render(<CommentThreadCard {...props} replyDraft="" />);
  expect(document.querySelector('[role="alert"]')).toBeNull();
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
    return visible ? <CommentThreadCard issueName={ISSUE} project={project} thread={thread} collapsible={false} replyDraft={draft} onReplyDraftChange={setDraft} /> : null;
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
  const view = (draft: string) => <MonacoViewZoneRevealContext value={reveal}><CommentThreadCard issueName={ISSUE} project={project} thread={thread} replyDraft={draft} onReplyDraftChange={vi.fn()} /></MonacoViewZoneRevealContext>;
  render(view(""));
  expect(reveal).not.toHaveBeenCalled();
  click(buttonByText("plan.review.thread.reply-placeholder"));
  expect(document.activeElement).toBe(container.querySelector("textarea"));
  expect(reveal).toHaveBeenCalledExactlyOnceWith(container.querySelector('[data-testid="thread-reply-footer"]'));
  expect(HTMLElement.prototype.scrollIntoView).not.toHaveBeenCalled();
  render(view("Typing a reply"));
  expect(reveal).toHaveBeenCalledTimes(1);
});
