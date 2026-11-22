import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { cleanup, fireEvent, render } from "@testing-library/react";
import { createRef, useState } from "react";
import { afterEach, describe, expect, test, vi } from "vitest";
import {
  IssueCommentSchema,
  IssueSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { Plan_SpecSchema } from "@/types/proto-es/v1/plan_service_pb";
import type { EditorThread } from "./threadModel";
import { ThreadStack } from "./ThreadStack";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, options?: { range?: string }) =>
      options?.range ? `${key}:${options.range}` : key,
  }),
}));
vi.mock("@/components/HumanizeTs", async () => ({
  ...(await import("@/test-utils/humanizeTs")).humanizeTsStub(),
}));
vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({ getUserByIdentifier: () => ({ email: "a@example.com", title: "Aurora" }) }),
}));
vi.mock("./CommentThreadCard", () => ({
  CommentThreadCard: ({ bar, className, thread, onThreadStateChanged, replyDraft, onReplyDraftChange, spec }: {
    className?: string;
    bar: { title: string; folded: boolean; onToggleFold: () => void };
    thread: EditorThread["thread"];
    onThreadStateChanged: (resolved: boolean) => void;
    replyDraft: string;
    onReplyDraftChange: (draft: string) => void;
    spec?: { id: string };
  }) => (
    <div data-class-name={className ?? ""} data-thread-name={thread.root.name} data-folded={String(bar.folded)} data-title={bar.title} data-resolved={String(thread.resolved)} data-spec={spec?.id}>
      <button onClick={bar.onToggleFold}>Fold</button>
      {!bar.folded && (
        <>
          <input aria-label="Reply draft" value={replyDraft} onChange={(e) => onReplyDraftChange(e.target.value)} />
          <button onClick={() => onThreadStateChanged(true)}>Resolved here</button>
          <button onClick={() => onThreadStateChanged(false)}>Reopened here</button>
        </>
      )}
    </div>
  ),
}));

afterEach(() => { cleanup(); vi.clearAllMocks(); });

const entry = (name: string, resolved: boolean, seconds: number, range = { startLine: 2, endLine: 2 }): EditorThread => ({
  range,
  thread: {
    root: create(IssueCommentSchema, {
      name,
      creator: "a@example.com",
      comment: `${name} summary\nsecond line`,
      createTime: create(TimestampSchema, { seconds: BigInt(seconds) }),
    }),
    replies: [],
    resolved,
  },
});
const props = {
  issue: create(IssueSchema, { name: "projects/p/issues/1" }),
  project: undefined,
  onCollapse: vi.fn(),
  onExpand: vi.fn(),
  expandedRoots: new Set<string>(),
  replyDrafts: {},
  onReplyDraftChange: vi.fn(),
  spec: create(Plan_SpecSchema, { id: "spec-1" }),
};
const order = (container: HTMLElement) =>
  Array.from(container.querySelectorAll("[data-thread-name]")).map((el) => el.getAttribute("data-thread-name"));

describe("ThreadStack", () => {
  test("keeps order and mounted drafts on status updates, then regroups on reopening", () => {
    const threads = [entry("resolved", true, 1), entry("open-a", false, 2), entry("open-b", false, 3)];
    const expandedRoots = new Set(["open-a"]);
    const { container, rerender } = render(<ThreadStack {...props} expandedRoots={expandedRoots} threads={threads} key="first-open" />);
    expect(order(container)).toEqual(["open-a", "open-b", "resolved"]);
    expect(container.querySelectorAll('[role="none"]')).toHaveLength(0);
    const draft = container.querySelector("input");
    const changed = [entry("resolved", false, 1), entry("open-a", true, 2), entry("open-b", false, 3)];
    rerender(<ThreadStack {...props} expandedRoots={expandedRoots} threads={changed} key="first-open" />);
    expect(order(container)).toEqual(["open-a", "open-b", "resolved"]);
    expect(container.querySelector("input")).toBe(draft);
    expect(container.querySelector('[data-thread-name="open-a"]')?.getAttribute("data-resolved")).toBe("true");
    rerender(<ThreadStack {...props} expandedRoots={expandedRoots} threads={changed} key="reopened" />);
    expect(order(container)).toEqual(["resolved", "open-b", "open-a"]);
  });

  test.each([false, true])("omits the separator for a single status group: %s", (resolved) => {
    const { container } = render(<ThreadStack {...props} threads={[entry("a", resolved, 1), entry("b", resolved, 2)]} />);
    expect(container.querySelectorAll('[role="none"]')).toHaveLength(0);
    expect(order(container)).toEqual(["a", "b"]);
  });

  test("keeps multiple threads expanded and folds only the chosen one", () => {
    const threads = [entry("a", false, 1), entry("b", false, 2)];
    const { container } = render(
      <ThreadStack {...props} expandedRoots={new Set(["a", "b"])} threads={threads} />
    );
    expect(container.querySelectorAll("input")).toHaveLength(2);
    fireEvent.click(container.querySelector('[data-thread-name="a"] button')!);
    expect(props.onCollapse).toHaveBeenCalledWith("a");
    expect(props.onExpand).not.toHaveBeenCalled();
  });

  test("collapses a thread resolved from its card and keeps a reopened one expanded", () => {
    const threads = [entry("a", false, 1)];
    const { getByText } = render(<ThreadStack {...props} expandedRoots={new Set(["a"])} threads={threads} />);
    fireEvent.click(getByText("Reopened here"));
    expect(props.onCollapse).not.toHaveBeenCalled();
    fireEvent.click(getByText("Resolved here"));
    expect(props.onCollapse).toHaveBeenCalledExactlyOnceWith("a");
  });

  test("preserves a reply draft when another thread opens", () => {
    function Harness() {
      const [expanded, setExpanded] = useState(new Set(["a"]));
      const [drafts, setDrafts] = useState<Record<string, string>>({});
      return <ThreadStack {...props}
        threads={[entry("a", false, 1), entry("b", false, 2)]}
        expandedRoots={expanded}
        onExpand={(name) => setExpanded((previous) => new Set(previous).add(name))}
        onCollapse={(name) => setExpanded((previous) => {
          const next = new Set(previous);
          next.delete(name);
          return next;
        })}
        replyDrafts={drafts}
        onReplyDraftChange={(name, draft) => setDrafts((previous) => ({...previous, [name]: typeof draft === "function" ? draft(previous[name] ?? "") : draft}))}
      />;
    }
    const { container } = render(<Harness />);
    const fold = (name: string) => fireEvent.click(container.querySelector(`[data-thread-name="${name}"] button`)!);
    fireEvent.change(container.querySelector('[data-thread-name="a"] input')!, { target: { value: "unfinished reply" } });
    fold("b");
    expect(container.querySelectorAll("input")).toHaveLength(2);
    expect((container.querySelector('[data-thread-name="a"] input') as HTMLInputElement).value).toBe("unfinished reply");
    fold("a");
    expect(container.querySelector('[data-thread-name="a"] input')).toBeNull();
    expect(container.querySelector('[data-thread-name="b"] input')).not.toBeNull();
    fold("a");
    expect((container.querySelector('[data-thread-name="a"] input') as HTMLInputElement).value).toBe("unfinished reply");
  });

  test("shows new threads and removes missing ones without remounting retained rows", () => {
    const { container, rerender } = render(<ThreadStack {...props} threads={[entry("a", false, 1), entry("b", true, 2)]} />);
    const first = container.querySelector('[data-thread-name="a"]');
    rerender(<ThreadStack {...props} threads={[entry("a", true, 1), entry("c", false, 3), entry("d", true, 4)]} />);
    expect(order(container)).toEqual(["a", "c", "d"]);
    expect(container.querySelector('[data-thread-name="a"]')).toBe(first);
    expect(container.textContent).not.toContain("second line");
  });
});


test("draws a folded thread as its card, titled by the lines it is on", () => {
  const { container } = render(
    <ThreadStack {...props} threads={[entry("one", false, 1), entry("range", true, 2, { startLine: 1, endLine: 7 })]} />
  );
  const cards = Array.from(container.querySelectorAll("[data-thread-name]"));
  expect(cards.map((card) => card.getAttribute("data-folded"))).toEqual(["true", "true"]);
  expect(cards.map((card) => card.getAttribute("data-title"))).toEqual([
    "plan.review.thread.comment-title:2",
    "plan.review.thread.comment-title:1–7",
  ]);
  expect(container.querySelector("[data-testid='thread-row']")).toBeNull();
});

test("the bar's fold control unfolds a folded thread and folds an unfolded one", () => {
  const { container } = render(
    <ThreadStack {...props} expandedRoots={new Set(["open"])} threads={[entry("open", false, 1), entry("folded", false, 2)]} />
  );
  fireEvent.click(container.querySelector('[data-thread-name="folded"] button')!);
  expect(props.onExpand).toHaveBeenCalledExactlyOnceWith("folded");
  expect(props.onCollapse).not.toHaveBeenCalled();
  fireEvent.click(container.querySelector('[data-thread-name="open"] button')!);
  expect(props.onCollapse).toHaveBeenCalledExactlyOnceWith("open");
});

test("hands each expanded card the spec of the statement it sits on", () => {
  const { container } = render(<ThreadStack {...props} expandedRoots={new Set(["a", "b"])} threads={[entry("a", false, 1), entry("b", true, 2)]} />);
  const cards = Array.from(container.querySelectorAll("[data-thread-name]"));
  expect(cards.map((card) => card.getAttribute("data-spec"))).toEqual(["spec-1", "spec-1"]);
});

test("exposes the expanded card for editor reveal", () => {
  const ref = createRef<HTMLDivElement>();
  const threads = [entry("a", false, 1), entry("b", false, 2)];
  const {rerender} = render(<ThreadStack {...props} threads={threads} expandedRoots={new Set(["a"])} expandedThreadRef={ref} revealRoot="a" />);
  expect(ref.current?.querySelector('[data-thread-name="a"]')).not.toBeNull();
  rerender(<ThreadStack {...props} threads={threads} expandedRoots={new Set(["a", "b"])} expandedThreadRef={ref} revealRoot="b" />);
  expect(ref.current?.querySelector('[data-thread-name="b"]')).not.toBeNull();
});

test("passes the card no classes, so its own frame stands in the Changes section", () => {
  const { container } = render(<ThreadStack {...props} expandedRoots={new Set(["a"])} threads={[entry("a", false, 1), entry("b", true, 2)]} />);
  const cards = Array.from(container.querySelectorAll("[data-thread-name]"));
  expect(cards.map((card) => card.getAttribute("data-class-name"))).toEqual(["", ""]);
});
