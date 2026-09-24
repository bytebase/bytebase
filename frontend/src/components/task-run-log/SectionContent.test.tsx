import { CheckCircle2 } from "lucide-react";
import { act, createElement, useState } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { TaskRunLogEntry_Type } from "@/types/proto-es/v1/rollout_service_pb";
import { SectionContent } from "./SectionContent";
import type { DisplayItem, Section } from "./types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  writeTextToClipboard: vi.fn(async (_text: string) => true),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/lib/clipboard", () => ({
  writeTextToClipboard: mocks.writeTextToClipboard,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => ({ notify: vi.fn() }) },
}));

const SHOW = "task-run.log-detail.show-full-statement";
const HIDE = "task-run.log-detail.hide-full-statement";

// jsdom has no layout, so the geometry the component reads is stubbed: a line
// is as wide as its text, its box is `lineWidth`, and rows stack at ROW_HEIGHT.
const CHAR_WIDTH = 7;
const ROW_HEIGHT = 28;
const layout = { lineWidth: 900 };

const isLine = (element: HTMLElement) => element.dataset.logPayload === "line";

const layoutStubs: Record<string, (this: HTMLElement) => number> = {
  scrollWidth() {
    return isLine(this) ? (this.textContent?.length ?? 0) * CHAR_WIDTH : 0;
  },
  clientWidth() {
    return isLine(this) ? layout.lineWidth : 0;
  },
  clientHeight() {
    return 10 * ROW_HEIGHT;
  },
  offsetHeight() {
    return ROW_HEIGHT;
  },
  offsetTop() {
    const siblings = Array.from(this.parentElement?.children ?? []);
    return siblings.indexOf(this) * ROW_HEIGHT;
  },
};

const resizeObservers: Array<() => void> = [];
class FakeResizeObserver {
  constructor(callback: ResizeObserverCallback) {
    resizeObservers.push(() =>
      callback([], this as unknown as ResizeObserver)
    );
  }
  observe() {}
  unobserve() {}
  disconnect() {}
}

const resizeTo = (lineWidth: number) => {
  layout.lineWidth = lineWidth;
  act(() => {
    for (const notify of resizeObservers) notify();
  });
};

let restoreLayout: Array<() => void> = [];

beforeEach(() => {
  layout.lineWidth = 900;
  resizeObservers.length = 0;
  mocks.writeTextToClipboard.mockClear();
  vi.stubGlobal("ResizeObserver", FakeResizeObserver);
  restoreLayout = Object.entries(layoutStubs).map(([property, get]) => {
    const original = Object.getOwnPropertyDescriptor(
      HTMLElement.prototype,
      property
    );
    Object.defineProperty(HTMLElement.prototype, property, {
      configurable: true,
      get,
    });
    return () => {
      if (original) {
        Object.defineProperty(HTMLElement.prototype, property, original);
      } else {
        delete (HTMLElement.prototype as unknown as Record<string, unknown>)[
          property
        ];
      }
    };
  });
});

afterEach(() => {
  for (const restore of restoreLayout) restore();
  vi.unstubAllGlobals();
  window.getSelection()?.removeAllRanges();
});

const baseItem = (key: string): DisplayItem => ({
  key,
  time: "12:00:00.000",
  timeMs: undefined,
  relativeTime: "",
  levelIndicator: "✓",
  levelClass: "text-success",
  detail: "",
  detailClass: "text-control",
});

const ran = (
  key: string,
  statement: string,
  extra: Partial<DisplayItem> = {}
): DisplayItem => ({
  ...baseItem(key),
  statement,
  detail: statement.trim().replace(/\s+/g, " "),
  ...extra,
});

const failed = (
  key: string,
  statement: string | undefined,
  error: string,
  extra: Partial<DisplayItem> = {}
): DisplayItem => ({
  ...baseItem(key),
  levelIndicator: "✗",
  levelClass: "text-error",
  detailClass: "text-error",
  statement,
  error,
  detail: error,
  ...extra,
});

const status = (key: string, detail: string): DisplayItem => ({
  ...baseItem(key),
  detail,
});

const sectionOf = (items: DisplayItem[]): Section => ({
  id: "section-0",
  type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
  label: "Command Execute",
  status: "success",
  statusIcon: CheckCircle2,
  statusClass: "text-success",
  duration: "2s",
  entryCount: items.length,
  items,
});

interface HarnessProps {
  items: DisplayItem[];
  datasetKey?: string;
  // Folds the reader made before this SectionContent mounted.
  initialOverrides?: ReadonlyMap<string, boolean>;
}

// The fold overrides live above SectionContent in the product, so the harness
// owns them the same way.
const Harness = ({
  items,
  datasetKey = "runs/1",
  initialOverrides,
}: HarnessProps) => {
  const [overrides, setOverrides] = useState<ReadonlyMap<string, boolean>>(
    () => initialOverrides ?? new Map()
  );
  return createElement(SectionContent, {
    section: sectionOf(items),
    datasetKey,
    foldOverrides: overrides,
    onFoldChange: (key: string, open: boolean) =>
      setOverrides((previous) => new Map(previous).set(key, open)),
  });
};

const mount = (props: HarnessProps, host: HTMLElement = document.body) => {
  const container = document.createElement("div");
  host.appendChild(container);
  const root = createRoot(container);
  const render = (next: HarnessProps) =>
    act(() => {
      root.render(createElement(Harness, next));
    });
  render(props);

  const rows = () =>
    Array.from(
      container.querySelectorAll<HTMLElement>(
        '[data-testid="task-run-log-row"]'
      )
    );
  const rowOf = (text: string) => {
    const row = rows().find((candidate) =>
      candidate.textContent?.includes(text)
    );
    if (!row) throw new Error(`no row containing "${text}"`);
    return row;
  };
  const within = (row: HTMLElement) => ({
    fold: () =>
      row.querySelector<HTMLButtonElement>(
        `button[aria-label="${SHOW}"], button[aria-label="${HIDE}"]`
      ),
    copy: () =>
      Array.from(
        row.querySelectorAll<HTMLButtonElement>(
          'button[aria-label="common.copy"]'
        )
      ),
    payload: (kind: "line" | "error" | "block") =>
      row.querySelector<HTMLElement>(`[data-log-payload="${kind}"]`),
  });

  return {
    container,
    render,
    rows,
    rowOf,
    within,
    scrollBox: () => rows()[0]?.parentElement as HTMLElement,
    unmount: () => {
      act(() => root.unmount());
      container.remove();
    },
  };
};

const click = (element: Element | null | undefined) => {
  if (!element) throw new Error("nothing to click");
  act(() => {
    element.dispatchEvent(
      new MouseEvent("click", { bubbles: true, cancelable: true })
    );
  });
};

const MULTI_LINE = "CREATE TABLE t (\n  id bigint,\n  name text\n);";
const WIDE = `CREATE INDEX idx ON public.loan_application (${"customer_id, ".repeat(12)}id);`;

describe("SectionContent folding", () => {
  test("a foldable row toggles and reports aria-expanded", () => {
    const view = mount({ items: [ran("a", MULTI_LINE)] });
    const row = () => view.within(view.rowOf("CREATE TABLE"));

    expect(row().fold()?.getAttribute("aria-expanded")).toBe("false");
    expect(row().fold()?.getAttribute("aria-label")).toBe(SHOW);
    expect(row().payload("block")).toBeNull();

    click(row().fold());
    expect(row().fold()?.getAttribute("aria-expanded")).toBe("true");
    expect(row().fold()?.getAttribute("aria-label")).toBe(HIDE);
    expect(row().payload("block")?.textContent).toContain("  id bigint,\n");
    // The block takes the line's place: one statement on screen at a time.
    expect(row().payload("line")).toBeNull();

    click(row().fold());
    expect(row().fold()?.getAttribute("aria-expanded")).toBe("false");
    expect(row().payload("block")).toBeNull();
    expect(row().payload("line")).not.toBeNull();

    view.unmount();
  });

  test("a marked failure starts folded like any other row", () => {
    const view = mount({
      items: [failed("a", MULTI_LINE, "ERROR: exists", { marked: true })],
    });
    const row = () => view.within(view.rowOf("ERROR: exists"));

    expect(row().fold()?.getAttribute("aria-expanded")).toBe("false");
    expect(row().payload("block")).toBeNull();
    expect(view.scrollBox().style.maxHeight).toBe("280px");

    // Unfolded, both payloads are the point: the error keeps the line and the
    // block sits beneath it.
    click(row().fold());
    expect(row().payload("error")?.textContent).toBe("ERROR: exists");
    expect(row().payload("block")?.textContent).toContain("CREATE TABLE t (");

    view.unmount();
  });

  test("the block shows the statement without the blank lines a split segment begins with", () => {
    const view = mount({ items: [ran("a", `\n\n${MULTI_LINE}\n`)] });
    const row = () => view.within(view.rowOf("CREATE TABLE"));

    click(row().fold());
    const block = row().payload("block");
    expect(block?.textContent?.startsWith("CREATE TABLE t (\n  id bigint,")).toBe(
      true
    );

    view.unmount();
  });
});

describe("SectionContent foldability", () => {
  test("a single-line statement that fits has no fold control until its container clamps it", () => {
    const view = mount({ items: [ran("a", "SET statement_timeout TO '3600s';")] });
    const row = () => view.within(view.rowOf("SET statement_timeout"));

    expect(row().fold()).toBeNull();

    resizeTo(100);
    expect(row().fold()).not.toBeNull();

    resizeTo(900);
    expect(row().fold()).toBeNull();

    view.unmount();
  });

  test("a multi-line statement is foldable at any width", () => {
    const view = mount({ items: [ran("a", MULTI_LINE)] });

    expect(view.within(view.rowOf("CREATE TABLE")).fold()).not.toBeNull();

    view.unmount();
  });

  test("whitespace around a one-line statement is nothing to unfold", () => {
    const view = mount({ items: [ran("a", "\nSELECT 1;\n")] });

    expect(view.within(view.rowOf("SELECT 1;")).fold()).toBeNull();

    view.unmount();
  });

  test("a failed row is foldable however short its statement", () => {
    const view = mount({ items: [failed("a", "SELECT 1", "ERROR: boom")] });
    const row = () => view.within(view.rowOf("ERROR: boom"));

    expect(row().fold()).not.toBeNull();
    expect(row().copy()).toHaveLength(0);

    click(row().fold());
    expect(row().payload("block")?.textContent).toContain("SELECT 1");
    expect(row().copy()).toHaveLength(1);

    click(row().fold());
    expect(row().payload("block")).toBeNull();
    expect(row().fold()).not.toBeNull();

    view.unmount();
  });

  test("a clamped one-line statement keeps its chevron while it is open", () => {
    layout.lineWidth = 100;
    const view = mount({ items: [ran("a", WIDE)] });
    const row = () => view.within(view.rowOf("CREATE INDEX"));

    click(row().fold());
    expect(row().payload("line")).toBeNull();

    // Opening raises the cap, which resizes the box, which fires the observer
    // while the line it would measure is gone.
    resizeTo(100);
    expect(row().fold()?.getAttribute("aria-expanded")).toBe("true");

    click(row().fold());
    expect(row().payload("block")).toBeNull();
    expect(row().fold()?.getAttribute("aria-expanded")).toBe("false");

    view.unmount();
  });

  test("the fold control survives closing, so focus is not dropped", () => {
    layout.lineWidth = 100;
    const view = mount({ items: [ran("a", WIDE)] });
    const row = () => view.within(view.rowOf("CREATE INDEX"));

    click(row().fold());
    const openControl = row().fold();
    click(openControl);

    expect(row().fold()).toBe(openControl);

    view.unmount();
  });

  test("a row already open when the section remounts can still be closed", () => {
    // Opened while its container clamped it; the section then collapsed and
    // reopened wide, so nothing is left to measure the verdict from.
    const view = mount({
      items: [ran("a", "SELECT 1;")],
      initialOverrides: new Map([["a", true]]),
    });
    const row = () => view.within(view.rowOf("SELECT 1;"));

    expect(row().payload("block")).not.toBeNull();
    expect(row().fold()?.getAttribute("aria-expanded")).toBe("true");

    click(row().fold());
    expect(row().payload("block")).toBeNull();
    // Closed, it fits its line, so there is nothing left to unfold.
    expect(row().fold()).toBeNull();

    view.unmount();
  });

  test("a marked failure with no recoverable statement has no chevron and no row toggle", () => {
    const view = mount({
      items: [failed("a", undefined, "ERROR: boom", { marked: true })],
    });
    const row = view.rowOf("ERROR: boom");

    expect(view.within(row).fold()).toBeNull();
    expect(view.within(row).copy()).toHaveLength(0);
    expect(view.scrollBox().style.maxHeight).toBe("280px");

    click(row.firstElementChild);
    expect(view.within(view.rowOf("ERROR: boom")).payload("block")).toBeNull();

    view.unmount();
  });

  test("rows that carry status words get neither control but keep the slot", () => {
    const view = mount({ items: [status("a", "BEGIN"), ran("b", MULTI_LINE)] });
    const begin = view.rowOf("BEGIN");
    const create = view.rowOf("CREATE TABLE");

    expect(begin.querySelectorAll("button")).toHaveLength(0);
    expect(begin.children).toHaveLength(create.children.length);

    view.unmount();
  });

  test("a status row that failed gets neither control", () => {
    const view = mount({
      items: [
        {
          ...status("a", "ROLLBACK: connection reset"),
          levelIndicator: "✗",
          detailClass: "text-error",
        },
      ],
    });
    const row = view.rowOf("ROLLBACK");

    expect(row.querySelectorAll("button")).toHaveLength(0);
    click(row.firstElementChild);
    expect(view.within(view.rowOf("ROLLBACK")).payload("block")).toBeNull();

    view.unmount();
  });

  test("rows appended by a poll are measured without a resize", () => {
    layout.lineWidth = 100;
    const first = ran("a", `${WIDE} -- first`);
    const view = mount({ items: [first] });
    expect(view.within(view.rowOf("-- first")).fold()).not.toBeNull();

    // A poll grows the capped box's scroll height, never its border box, so no
    // observer fires for the new row.
    view.render({ items: [first, ran("b", `${WIDE} -- second`)] });

    expect(view.within(view.rowOf("-- second")).fold()).not.toBeNull();

    view.unmount();
  });

  test("rows revealed by load more are measured too", () => {
    layout.lineWidth = 100;
    const view = mount({
      items: Array.from({ length: 60 }, (_, index) =>
        ran(`item-${index}`, `${WIDE} -- ${index}`)
      ),
    });

    const loadMore = Array.from(view.container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("common.load-more")
    );
    click(loadMore);

    expect(view.rows()).toHaveLength(60);
    expect(view.within(view.rowOf("-- 59")).fold()).not.toBeNull();

    view.unmount();
  });
});

describe("SectionContent copy", () => {
  test("copy takes the verbatim statement, never the line", async () => {
    const statement = `\n${MULTI_LINE}\n`;
    const view = mount({ items: [ran("a", statement)] });
    const row = () => view.within(view.rowOf("CREATE TABLE"));

    expect(row().copy()).toHaveLength(1);
    await act(async () => {
      row().copy()[0]?.click();
    });
    expect(mocks.writeTextToClipboard).toHaveBeenLastCalledWith(statement);

    // Unfolded, the cluster's button gives way to the block's.
    click(row().fold());
    expect(row().copy()).toHaveLength(1);
    expect(row().payload("block")?.contains(row().copy()[0] ?? null)).toBe(true);
    await act(async () => {
      row().copy()[0]?.click();
    });
    expect(mocks.writeTextToClipboard).toHaveBeenLastCalledWith(statement);
    expect(mocks.writeTextToClipboard).toHaveBeenCalledTimes(2);

    view.unmount();
  });

  test("a statement that fits its line still offers copy", async () => {
    const view = mount({ items: [ran("a", "SELECT 1;")] });
    const row = view.within(view.rowOf("SELECT 1;"));

    expect(row.fold()).toBeNull();
    expect(row.copy()).toHaveLength(1);
    await act(async () => {
      row.copy()[0]?.click();
    });
    expect(mocks.writeTextToClipboard).toHaveBeenLastCalledWith("SELECT 1;");

    view.unmount();
  });

  test("a failed row has no copy on its error line and one inside its block", async () => {
    const view = mount({ items: [failed("a", MULTI_LINE, "ERROR: exists")] });
    const row = () => view.within(view.rowOf("ERROR: exists"));

    expect(row().copy()).toHaveLength(0);

    click(row().fold());
    expect(row().copy()).toHaveLength(1);
    expect(row().payload("block")?.contains(row().copy()[0] ?? null)).toBe(true);
    await act(async () => {
      row().copy()[0]?.click();
    });
    expect(mocks.writeTextToClipboard).toHaveBeenLastCalledWith(MULTI_LINE);

    view.unmount();
  });

  test("a row with no recoverable statement has no copy at all", () => {
    const view = mount({
      items: [status("a", "-"), failed("b", undefined, "ERROR: boom")],
    });

    expect(
      view.container.querySelectorAll('button[aria-label="common.copy"]')
    ).toHaveLength(0);

    view.unmount();
  });
});

describe("SectionContent row clicks", () => {
  const openBlock = (view: ReturnType<typeof mount>) =>
    view.within(view.rowOf("CREATE TABLE")).payload("block");

  test("the index, the timestamp and the row's empty space toggle", () => {
    const view = mount({
      items: [ran("a", MULTI_LINE, { relativeTime: "+12ms" })],
    });
    const row = () => view.rowOf("CREATE TABLE");
    const cell = (text: string) =>
      Array.from(row().querySelectorAll("span")).find(
        (span) => span.textContent === text
      );

    click(cell("1"));
    expect(openBlock(view)).not.toBeNull();
    click(cell("12:00:00.000"));
    expect(openBlock(view)).toBeNull();
    click(cell("+12ms"));
    expect(openBlock(view)).not.toBeNull();
    click(row());
    expect(openBlock(view)).toBeNull();

    view.unmount();
  });

  test("a click on the folded statement line unfolds the row", () => {
    const view = mount({ items: [ran("a", MULTI_LINE)] });
    const row = () => view.within(view.rowOf("CREATE TABLE"));

    click(row().payload("line"));

    expect(row().payload("block")?.textContent).toContain("  id bigint,\n");
    expect(row().fold()?.getAttribute("aria-expanded")).toBe("true");

    view.unmount();
  });

  test("a click inside the unfolded block does not fold it", () => {
    const view = mount({ items: [ran("a", MULTI_LINE)] });
    const row = () => view.within(view.rowOf("CREATE TABLE"));

    click(row().fold());
    click(row().payload("block"));

    expect(row().payload("block")).not.toBeNull();

    view.unmount();
  });

  test("two quick clicks on the clamped line leave the row open", () => {
    const view = mount({ items: [ran("a", MULTI_LINE)] });
    const row = () => view.within(view.rowOf("CREATE TABLE"));

    // A double-click: the first click opens the row, which swaps the line for
    // the block, so the second lands on the block.
    click(row().payload("line"));
    click(row().payload("block"));

    expect(row().payload("block")).not.toBeNull();

    view.unmount();
  });

  test("a drag across the clamped line does not unfold it", () => {
    const view = mount({ items: [ran("a", MULTI_LINE)] });
    const row = () => view.within(view.rowOf("CREATE TABLE"));

    const range = document.createRange();
    range.selectNodeContents(row().payload("line") as HTMLElement);
    window.getSelection()?.addRange(range);
    click(row().payload("line"));

    expect(row().payload("block")).toBeNull();

    view.unmount();
  });

  test("the line of a row with nothing to unfold ignores clicks", () => {
    const view = mount({ items: [ran("a", "SELECT 1;")] });
    const row = () => view.within(view.rowOf("SELECT 1;"));

    click(row().payload("line"));

    expect(row().payload("block")).toBeNull();

    view.unmount();
  });

  test("the statement line is mouse convenience, not a control", () => {
    const view = mount({ items: [ran("a", MULTI_LINE)] });
    const row = view.rowOf("CREATE TABLE");
    const line = view.within(row).payload("line");

    expect(line?.getAttribute("role")).toBeNull();
    expect(line?.getAttribute("tabindex")).toBeNull();
    // Assistive technology reaches the row through its two buttons alone.
    expect(
      Array.from(row.querySelectorAll("[role], [tabindex], button")).map(
        (element) => element.getAttribute("aria-label")
      )
    ).toEqual([SHOW, "common.copy"]);

    view.unmount();
  });

  test("a click on the error line unfolds and folds a failed row", () => {
    const view = mount({ items: [failed("a", MULTI_LINE, "ERROR: exists")] });
    const row = () => view.within(view.rowOf("ERROR: exists"));

    click(row().payload("error"));
    expect(row().payload("block")?.textContent).toContain("CREATE TABLE t (");
    expect(row().fold()?.getAttribute("aria-expanded")).toBe("true");
    // The error keeps the line either way.
    expect(row().payload("error")?.textContent).toBe("ERROR: exists");

    click(row().payload("error"));
    expect(row().payload("block")).toBeNull();

    view.unmount();
  });

  test("a drag across the error line does not unfold the row", () => {
    const view = mount({ items: [failed("a", MULTI_LINE, "ERROR: exists")] });
    const row = () => view.within(view.rowOf("ERROR: exists"));

    const range = document.createRange();
    range.selectNodeContents(row().payload("error") as HTMLElement);
    window.getSelection()?.addRange(range);
    click(row().payload("error"));

    expect(row().payload("block")).toBeNull();

    view.unmount();
  });

  test("the error of a failure with no recoverable statement ignores clicks", () => {
    const view = mount({
      items: [failed("a", undefined, "ERROR: boom", { marked: true })],
    });
    const row = () => view.within(view.rowOf("ERROR: boom"));

    click(row().payload("error"));

    expect(row().payload("block")).toBeNull();
    expect(row().fold()).toBeNull();

    view.unmount();
  });

  test("copying does not toggle the row", async () => {
    const view = mount({ items: [ran("a", MULTI_LINE)] });
    const row = () => view.within(view.rowOf("CREATE TABLE"));

    await act(async () => {
      row().copy()[0]?.click();
    });
    expect(row().payload("block")).toBeNull();

    view.unmount();
  });

  test("a click that ends a selection does not toggle", () => {
    const view = mount({ items: [ran("a", MULTI_LINE)] });
    const row = view.rowOf("CREATE TABLE");
    const line = view.within(row).payload("line");

    const range = document.createRange();
    range.selectNodeContents(line as HTMLElement);
    window.getSelection()?.addRange(range);

    click(row.firstElementChild);
    expect(view.within(view.rowOf("CREATE TABLE")).payload("block")).toBeNull();

    view.unmount();
  });
});

describe("SectionContent section height", () => {
  test("the box keeps its ten-row cap while a block is on screen", () => {
    const view = mount({ items: [ran("a", MULTI_LINE), ran("b", MULTI_LINE)] });
    const folds = () => view.rows().map((row) => view.within(row).fold());

    expect(view.scrollBox().style.maxHeight).toBe("280px");
    click(folds()[0]);
    click(folds()[1]);
    expect(view.scrollBox().style.maxHeight).toBe("280px");
    click(folds()[0]);
    click(folds()[1]);
    expect(view.scrollBox().style.maxHeight).toBe("280px");

    view.unmount();
  });

  test("the block's copy button is a sticky sibling of the SQL, not an overlay on it", () => {
    const view = mount({ items: [ran("a", MULTI_LINE)] });
    const row = () => view.within(view.rowOf("CREATE TABLE"));

    click(row().fold());
    const block = row().payload("block") as HTMLElement;
    const copy = row().copy()[0] as HTMLElement;
    const holder = copy.closest("[data-log-copy]") as HTMLElement;

    expect(holder.parentElement).toBe(block);
    expect(holder.className).toContain("sticky");
    expect(holder.className).not.toContain("absolute");
    // The SQL is its own child, so the two never share a box.
    expect(block.querySelector("[data-log-sql]")?.textContent).toContain(
      "CREATE TABLE t ("
    );

    view.unmount();
  });
});

describe("SectionContent render window", () => {
  const sixty = (last: DisplayItem) => [
    ...Array.from({ length: 59 }, (_, index) =>
      ran(`item-${index}`, `SELECT ${index};`)
    ),
    last,
  ];

  test("renders a load more action for large sections", () => {
    const items = sixty(ran("item-59", "SELECT 59;"));
    const view = mount({ items });

    expect(view.container.textContent).toContain("common.load-more");
    expect(view.container.textContent).toContain("(10)");
    expect(view.container.textContent).toContain("SELECT 0;");
    expect(view.container.textContent).not.toContain("SELECT 59;");

    const loadMore = Array.from(view.container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("common.load-more")
    );
    click(loadMore);
    expect(view.container.textContent).toContain("SELECT 59;");

    view.render({ items, datasetKey: "runs/2" });
    expect(view.container.textContent).toContain("common.load-more");
    expect(view.container.textContent).not.toContain("SELECT 59;");

    view.unmount();
  });

  test("a marked failure past the window is rendered, numbered and scrolled to", () => {
    const view = mount({
      items: sixty(
        failed("item-59", "ALTER TABLE t ADD c int;", "ERROR: exists", {
          marked: true,
        })
      ),
    });

    const marked = view.rowOf("ERROR: exists");
    // Surfaced, not opened.
    expect(view.within(marked).payload("block")).toBeNull();
    expect(view.rows()).toHaveLength(51);
    expect(view.container.textContent).toContain("(9)");
    // Its index in the section, not its position in the sparse window.
    expect(marked.firstElementChild?.textContent).toBe("60");
    // 50 rows and the load-more button sit above it.
    expect(view.scrollBox().scrollTop).toBe(51 * ROW_HEIGHT);

    const loadMore = Array.from(view.container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("common.load-more")
    );
    click(loadMore);
    expect(view.rows()).toHaveLength(60);
    expect(view.rowOf("ERROR: exists").firstElementChild?.textContent).toBe(
      "60"
    );
    expect(view.rows().at(-1)).toBe(view.rowOf("ERROR: exists"));

    view.unmount();
  });

  test("a marked row inside the window is rendered once, in place", () => {
    const items = sixty(ran("item-59", "SELECT 59;"));
    items[10] = failed("item-10", "SELECT 10;", "ERROR: boom", {
      marked: true,
    });
    const view = mount({ items });

    expect(view.rows()).toHaveLength(50);
    expect(view.container.textContent).toContain("(10)");
    const marked = view.rows().filter((row) =>
      row.textContent?.includes("ERROR: boom")
    );
    expect(marked).toHaveLength(1);
    expect(marked[0]?.firstElementChild?.textContent).toBe("11");

    view.unmount();
  });

  test("a later poll does not scroll the reader back to the marked row", () => {
    const items = sixty(
      failed("item-59", "ALTER TABLE t ADD c int;", "ERROR: exists", {
        marked: true,
      })
    );
    const view = mount({ items });
    expect(view.scrollBox().scrollTop).toBeGreaterThan(0);

    view.scrollBox().scrollTop = 0;
    view.render({ items: items.map((item) => ({ ...item })) });
    expect(view.scrollBox().scrollTop).toBe(0);

    view.unmount();
  });
});

describe("SectionContent live updates", () => {
  const block = (view: ReturnType<typeof mount>, text: string) =>
    view.within(view.rowOf(text)).payload("block");

  test("a newly marked failure is brought into view, not opened, without remounting", () => {
    const commands = Array.from({ length: 59 }, (_, index) =>
      ran(`item-${index}`, `SELECT ${index};`)
    );
    const view = mount({ items: [...commands, ran("last", "SELECT 59;")] });
    const firstRow = view.rows()[0];
    expect(view.scrollBox().scrollTop).toBe(0);

    view.render({
      items: [
        ...commands,
        failed("last", "SELECT 59;", "ERROR: boom", { marked: true }),
      ],
    });

    expect(view.rows()[0]).toBe(firstRow);
    expect(view.scrollBox().scrollTop).toBe(51 * ROW_HEIGHT);
    expect(block(view, "ERROR: boom")).toBeNull();

    view.unmount();
  });

  test("a row the reader unfolded stays unfolded when the next poll arrives", () => {
    const item = () => failed("cmd", MULTI_LINE, "ERROR: exists", { marked: true });
    const view = mount({ items: [item()] });

    click(view.within(view.rowOf("ERROR: exists")).fold());
    view.render({ items: [item(), status("next", "ROLLBACK")] });

    expect(block(view, "ERROR: exists")).not.toBeNull();

    view.unmount();
  });

  test("a row the reader unfolded stays unfolded when its mark moves away", () => {
    const view = mount({
      items: [failed("cmd", MULTI_LINE, "ERROR: exists", { marked: true })],
    });

    click(view.within(view.rowOf("ERROR: exists")).fold());
    view.render({
      items: [
        failed("cmd", MULTI_LINE, "ERROR: exists"),
        status("retry", "attempt 1/3"),
      ],
    });

    expect(block(view, "ERROR: exists")).not.toBeNull();

    view.unmount();
  });

  test("a poll never opens a row", () => {
    const view = mount({ items: [ran("cmd", MULTI_LINE)] });

    view.render({
      items: [failed("cmd", MULTI_LINE, "ERROR: exists", { marked: true })],
    });

    expect(block(view, "ERROR: exists")).toBeNull();

    view.unmount();
  });
});

describe("SectionContent columns", () => {
  const widths = (view: ReturnType<typeof mount>) => ({
    index: view.scrollBox().style.getPropertyValue("--task-log-index-width"),
    time: view.scrollBox().style.getPropertyValue("--task-log-time-width"),
  });

  test("a short section keeps the floors", () => {
    const view = mount({
      items: [ran("a", "SELECT 1;"), ran("b", "SELECT 2;", { relativeTime: "+12ms" })],
    });

    expect(widths(view)).toEqual({ index: "max(24px, 1ch)", time: "7ch" });

    view.unmount();
  });

  test("the widest value sets the column for every row, including in the sparse window", () => {
    const view = mount({
      items: [
        ...Array.from({ length: 999 }, (_, index) =>
          ran(`item-${index}`, `SELECT ${index};`, { relativeTime: "+12ms" })
        ),
        failed("item-999", "SELECT 999;", "ERROR: boom", {
          marked: true,
          relativeTime: "+1234.56s",
        }),
      ],
    });

    // Rows 1-50 sit directly above row 1000.
    expect(view.rows()).toHaveLength(51);
    expect(widths(view)).toEqual({ index: "max(24px, 4ch)", time: "9ch" });

    // No row carries a width of its own, so none can disagree with the rest.
    for (const row of view.rows()) {
      const [index, , relativeTime] = Array.from(row.children) as HTMLElement[];
      expect(index?.className).toContain("w-(--task-log-index-width)");
      expect(relativeTime?.className).toContain("w-(--task-log-time-width)");
      expect(index?.getAttribute("style")).toBeNull();
      expect(relativeTime?.getAttribute("style")).toBeNull();
    }

    view.unmount();
  });

  test("the relative-time column is kept when a row has no value", () => {
    const view = mount({
      items: [ran("a", "SELECT 1;"), ran("b", "SELECT 2;", { relativeTime: "+12ms" })],
    });
    const [first, second] = view.rows();

    expect(first?.children).toHaveLength(second?.children.length ?? -1);

    view.unmount();
  });
});

describe("SectionContent revealing what a toggle opened", () => {
  // jsdom lays nothing out, so this models the box as a 280px scroller over a
  // stack of 28px rows: an open row is its line plus `blockHeight`, and the
  // box's scroll range grows and shrinks with it, clamping like a browser's.
  // The real motion is checked in a browser and in the e2e journey.
  const BOX_HEIGHT = 280;
  const BOX_TOP = 100;
  const PAGE_START = 1000;
  const geometry = { blockHeight: 0 };
  let page: HTMLElement;
  let restoreRect: () => void;

  const pageDelta = () => page.scrollTop - PAGE_START;
  const rect = (top: number, height: number) =>
    ({ top, bottom: top + height, height }) as DOMRect;
  const rowsOf = (box: HTMLElement) =>
    Array.from(box.querySelectorAll<HTMLElement>('[data-testid="task-run-log-row"]'));
  const rowHeight = (row: HTMLElement) =>
    ROW_HEIGHT + (row.querySelector('[data-log-payload="block"]') ? geometry.blockHeight : 0);
  const rowOffset = (row: HTMLElement) =>
    rowsOf(row.parentElement as HTMLElement)
      .slice(0, rowsOf(row.parentElement as HTMLElement).indexOf(row))
      .reduce((sum, sibling) => sum + rowHeight(sibling), 0);
  const contentHeight = (box: HTMLElement) =>
    rowsOf(box).reduce((sum, row) => sum + rowHeight(row), 0);

  const clampedScroll = (element: HTMLElement, max: () => number) => {
    let value = 0;
    Object.defineProperty(element, "scrollTop", {
      configurable: true,
      get: () => Math.max(0, Math.min(max(), value)),
      set: (next: number) => {
        value = Math.max(0, Math.min(max(), next));
      },
    });
  };

  beforeEach(() => {
    geometry.blockHeight = 0;
    page = document.createElement("div");
    page.style.overflowY = "auto";
    document.body.appendChild(page);
    Object.defineProperty(page, "scrollHeight", { get: () => 5000 });
    clampedScroll(page, () => 4000);
    page.scrollTop = PAGE_START;

    const original = HTMLElement.prototype.getBoundingClientRect;
    HTMLElement.prototype.getBoundingClientRect = function (this: HTMLElement) {
      if (this.dataset.testid === "task-run-log-row") {
        const box = this.parentElement as HTMLElement;
        return rect(
          BOX_TOP + rowOffset(this) - box.scrollTop - pageDelta(),
          rowHeight(this)
        );
      }
      if (this.firstElementChild?.getAttribute("data-testid") === "task-run-log-row") {
        return rect(
          BOX_TOP - pageDelta(),
          Math.min(BOX_HEIGHT, contentHeight(this))
        );
      }
      return rect(0, 0);
    };
    restoreRect = () => {
      HTMLElement.prototype.getBoundingClientRect = original;
    };
  });

  afterEach(() => {
    restoreRect();
    page.remove();
  });

  // Twelve rows, the failure last: a full box with the marked row parked at
  // its bottom, which is the shape a failed log arrives in.
  const mountFullBox = (blockHeight: number) => {
    geometry.blockHeight = blockHeight;
    const view = mount(
      {
        items: [
          ...Array.from({ length: 11 }, (_, index) =>
            ran(`ok-${index}`, `SELECT ${index};`)
          ),
          failed("last", MULTI_LINE, "ERROR: exists", { marked: true }),
        ],
      },
      page
    );
    const box = view.scrollBox();
    clampedScroll(box, () =>
      Math.max(0, contentHeight(box) - Math.min(BOX_HEIGHT, contentHeight(box)))
    );
    // D13 parks the marked row: the last row, so at the bottom of the box.
    box.scrollTop = 12 * ROW_HEIGHT;
    return view;
  };
  const lineTop = (view: ReturnType<typeof mount>) =>
    view.rowOf("ERROR: exists").getBoundingClientRect().top;
  const boxTop = (view: ReturnType<typeof mount>) =>
    view.scrollBox().getBoundingClientRect().top;
  const toggleByError = (view: ReturnType<typeof mount>) =>
    click(view.within(view.rowOf("ERROR: exists")).payload("error"));

  test("unfolding a row whose block already fits scrolls nothing", () => {
    geometry.blockHeight = 60;
    const view = mount(
      { items: [ran("a", "SELECT 1;"), failed("b", MULTI_LINE, "ERROR: exists")] },
      page
    );
    clampedScroll(view.scrollBox(), () => 0);
    const before = lineTop(view);

    toggleByError(view);

    expect(view.scrollBox().scrollTop).toBe(0);
    expect(lineTop(view)).toBe(before);
    expect(page.scrollTop).toBe(PAGE_START);

    view.unmount();
  });

  test("unfolding the bottom row scrolls the section by exactly the block's overflow", () => {
    const view = mountFullBox(100);
    const before = lineTop(view);
    expect(before).toBe(boxTop(view) + 9 * ROW_HEIGHT);

    const parked = 12 * ROW_HEIGHT - BOX_HEIGHT;
    expect(view.scrollBox().scrollTop).toBe(parked);

    toggleByError(view);

    expect(view.scrollBox().scrollTop).toBe(parked + 100);
    expect(lineTop(view)).toBe(before - 100);
    expect(page.scrollTop).toBe(PAGE_START);

    view.unmount();
  });

  test("unfolding a row taller than the box puts its line at the box's top", () => {
    const view = mountFullBox(640);

    toggleByError(view);

    expect(lineTop(view)).toBe(boxTop(view));
    expect(page.scrollTop).toBe(PAGE_START);

    view.unmount();
  });

  test("folding scrolls nothing itself: the line comes back to where it was clicked from", () => {
    const view = mountFullBox(640);
    const before = lineTop(view);

    toggleByError(view);
    expect(lineTop(view)).toBe(boxTop(view));
    toggleByError(view);

    expect(view.within(view.rowOf("ERROR: exists")).payload("block")).toBeNull();
    expect(lineTop(view)).toBe(before);
    expect(page.scrollTop).toBe(PAGE_START);

    view.unmount();
  });

  test("a reveal in one section leaves another section's scroll alone", () => {
    geometry.blockHeight = 640;
    const TwoSections = () => {
      const [overrides, setOverrides] = useState<ReadonlyMap<string, boolean>>(
        () => new Map()
      );
      const onFoldChange = (key: string, open: boolean) =>
        setOverrides((previous) => new Map(previous).set(key, open));
      const items = (prefix: string) => [
        ...Array.from({ length: 11 }, (_, index) =>
          ran(`${prefix}-${index}`, `SELECT ${index};`)
        ),
        failed(`${prefix}-last`, MULTI_LINE, `ERROR: ${prefix}`),
      ];
      return createElement(
        "div",
        null,
        createElement(SectionContent, {
          section: { ...sectionOf(items("first")), id: "section-0" },
          foldOverrides: overrides,
          onFoldChange,
        }),
        createElement(SectionContent, {
          section: { ...sectionOf(items("second")), id: "section-1" },
          foldOverrides: overrides,
          onFoldChange,
        })
      );
    };
    const container = document.createElement("div");
    page.appendChild(container);
    const root = createRoot(container);
    act(() => root.render(createElement(TwoSections)));
    const rowOf = (text: string) =>
      Array.from(
        container.querySelectorAll<HTMLElement>('[data-testid="task-run-log-row"]')
      ).find((row) => row.textContent?.includes(text)) as HTMLElement;
    const firstBox = rowOf("ERROR: first").parentElement as HTMLElement;
    clampedScroll(firstBox, () => contentHeight(firstBox) - BOX_HEIGHT);
    firstBox.scrollTop = 40;

    click(rowOf("ERROR: second").querySelector('[data-log-payload="error"]'));
    expect(firstBox.scrollTop).toBe(40);

    // The first section's own unfold is revealed once. If the reader then
    // scrolls that row back out of view, a toggle elsewhere must not reveal it
    // again.
    click(rowOf("ERROR: first").querySelector('[data-log-payload="error"]'));
    const revealed = firstBox.scrollTop;
    expect(revealed).toBeGreaterThan(40);
    firstBox.scrollTop = 40;
    click(rowOf("ERROR: second").querySelector('[data-log-payload="error"]'));
    expect(firstBox.scrollTop).toBe(40);

    act(() => root.unmount());
    container.remove();
  });
});

describe("SectionContent time cell", () => {
  const cell = (view: ReturnType<typeof mount>, text: string) =>
    Array.from(view.container.querySelectorAll("span")).find(
      (span) => span.textContent === text
    );

  test("offers the full date only for a line that has an instant", () => {
    const view = mount({
      items: [
        { ...ran("timed", "SELECT 1;"), time: "12:00:00.000", timeMs: Date.UTC(2026, 2, 2, 12) },
        { ...ran("untimed", "SELECT 2;"), time: "--:--:--.---", timeMs: undefined },
        { ...ran("impossible", "SELECT 3;"), time: "12:00:02.000", timeMs: 8.64e15 + 1 },
      ],
    });

    // A tooltip with nothing to say renders its children bare, so the trigger
    // it wraps them in is what says whether the full date is on offer.
    expect(cell(view, "12:00:00.000")?.firstElementChild).not.toBeNull();
    expect(cell(view, "--:--:--.---")?.firstElementChild).toBeNull();
    // Finite, and past the last instant there is: no reading, so no tooltip.
    expect(cell(view, "12:00:02.000")?.firstElementChild).toBeNull();

    view.unmount();
  });

  test("the offered date is the line's own instant", () => {
    // Which instant, not merely that there is one: a tooltip built from the
    // wrong time reads as plausibly as the right one. The zone is pinned to
    // Asia/Shanghai in vitest.config.ts, so noon UTC reads as 8pm.
    vi.useFakeTimers();
    const view = mount({
      items: [
        { ...ran("timed", "SELECT 1;"), time: "20:00:00.000", timeMs: Date.UTC(2026, 2, 2, 12) },
      ],
    });

    act(() => {
      cell(view, "20:00:00.000")?.firstElementChild?.dispatchEvent(
        new FocusEvent("focusin", { bubbles: true })
      );
      vi.advanceTimersByTime(200);
    });

    const overlay = document.getElementById("bb-react-layer-overlay");
    expect(overlay?.textContent).toContain("Mar 2, 2026");
    expect(overlay?.textContent).toContain("8:00:00");

    view.unmount();
    vi.useRealTimers();
  });
});
