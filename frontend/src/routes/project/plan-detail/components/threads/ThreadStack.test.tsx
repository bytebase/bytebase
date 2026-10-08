import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { cleanup, fireEvent, render } from "@testing-library/react";
import { createRef, useState } from "react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { IssueCommentSchema, IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import type { EditorThread } from "./threadModel";
import { ThreadStack } from "./ThreadStack";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/components/HumanizeTs", async () => ({
  ...(await import("@/test-utils/humanizeTs")).humanizeTsStub(),
}));
vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({ getUserByIdentifier: () => ({ email: "a@example.com", title: "Aurora" }) }),
}));
vi.mock("./CommentThreadCard", () => ({
  CommentThreadCard: ({ thread, onClose, replyDraft, onReplyDraftChange }: {
    thread: EditorThread["thread"];
    onClose: () => void;
    replyDraft: string;
    onReplyDraftChange: (draft: string) => void;
  }) => (
    <div data-thread-name={thread.root.name} data-resolved={String(thread.resolved)}>
      <input aria-label="Reply draft" value={replyDraft} onChange={(e) => onReplyDraftChange(e.target.value)} />
      <button onClick={onClose}>Close</button>
    </div>
  ),
}));

afterEach(() => { cleanup(); vi.clearAllMocks(); });

const entry = (name: string, resolved: boolean, seconds: number): EditorThread => ({
  range: { startLine: 2, endLine: 2 },
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

  test("keeps multiple threads expanded and closes only the chosen one", () => {
    const threads = [entry("a", false, 1), entry("b", false, 2)];
    const { container } = render(
      <ThreadStack {...props} expandedRoots={new Set(["a", "b"])} threads={threads} />
    );
    expect(container.querySelectorAll("input")).toHaveLength(2);
    fireEvent.click(container.querySelector('[data-thread-name="a"] button')!);
    expect(props.onCollapse).toHaveBeenCalledWith("a");
    expect(props.onExpand).not.toHaveBeenCalled();
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
    fireEvent.change(container.querySelector('[data-thread-name="a"] input')!, { target: { value: "unfinished reply" } });
    fireEvent.click(container.querySelector('[data-thread-name="b"]')!);
    expect(container.querySelectorAll("input")).toHaveLength(2);
    expect((container.querySelector('[data-thread-name="a"] input') as HTMLInputElement).value).toBe("unfinished reply");
    fireEvent.click(container.querySelector('[data-thread-name="a"] button')!);
    expect(container.querySelector('[data-thread-name="b"] input')).not.toBeNull();
    fireEvent.click(container.querySelector('[data-thread-name="a"]')!);
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


test("exposes the expanded card for editor reveal", () => {
  const ref = createRef<HTMLDivElement>();
  const threads = [entry("a", false, 1), entry("b", false, 2)];
  const {rerender} = render(<ThreadStack {...props} threads={threads} expandedRoots={new Set(["a"])} expandedThreadRef={ref} revealRoot="a" />);
  expect(ref.current?.querySelector('[data-thread-name="a"]')).not.toBeNull();
  rerender(<ThreadStack {...props} threads={threads} expandedRoots={new Set(["a", "b"])} expandedThreadRef={ref} revealRoot="b" />);
  expect(ref.current?.querySelector('[data-thread-name="b"]')).not.toBeNull();
});
