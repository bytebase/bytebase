import { act, useContext, useEffect } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { fakeEditorLayout, lineTop } from "@/test-utils/monacoLayout";
import { MonacoViewZone, MonacoViewZoneRevealContext } from "./MonacoViewZone";
import type { IStandaloneCodeEditor } from "./types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

type FakeZone = {
  afterLineNumber: number;
  domNode: HTMLElement;
  heightInPx: number;
  onDomNodeTop?: (top: number) => void;
};

function fakeEditor(layoutOptions: { scrollTop?: number } = {}) {
  const zones = new Map<string, FakeZone>();
  let nextId = 0;
  const widgets = new Set<{ getDomNode: () => HTMLElement }>();
  const layout = fakeEditorLayout(
    () =>
      Array.from(zones, ([id, zone]) => ({
        id,
        afterLineNumber: zone.afterLineNumber,
        height: zone.heightInPx,
      })),
    layoutOptions
  );
  const editor = {
    ...layout,
    addOverlayWidget: (widget: { getDomNode: () => HTMLElement }) => {
      widgets.add(widget);
    },
    removeOverlayWidget: (widget: { getDomNode: () => HTMLElement }) => {
      widgets.delete(widget);
    },
    onDidLayoutChange: () => ({ dispose: vi.fn() }),
    getModel: () => ({}),
    changeViewZones: (
      callback: (accessor: {
        addZone: (zone: FakeZone) => string;
        removeZone: (id: string) => void;
        layoutZone: (id: string) => void;
      }) => void
    ) =>
      callback({
        addZone: (zone) => {
          const id = `zone-${nextId++}`;
          zones.set(id, zone);
          return id;
        },
        removeZone: (id) => {
          zones.delete(id);
        },
        layoutZone: vi.fn(),
      }),
  };
  // A sibling zone Monaco stacks before the component's, after the same line.
  const addZoneBefore = (afterLineNumber: number, height: number) =>
    zones.set(`sibling-${nextId++}`, {
      afterLineNumber,
      domNode: document.createElement("div"),
      heightInPx: height,
    });
  return {
    addZoneBefore,
    editor: editor as unknown as IStandaloneCodeEditor,
    scroll: layout.setScrollTop,
    widgets,
    zones,
  };
}

describe("MonacoViewZone", () => {
  test("reserves a zone after the line, hosts the content in an overlay widget, and removes both on unmount", () => {
    const { editor, widgets, zones } = fakeEditor();
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(
        <MonacoViewZone afterLineNumber={4} editor={editor}>
          <span data-testid="content">hello</span>
        </MonacoViewZone>
      );
    });
    expect(zones.size).toBe(1);
    const [zone] = Array.from(zones.values());
    expect(zone.afterLineNumber).toBe(4);
    // The zone only reserves space; the content lives in the overlay widget.
    expect(zone.domNode.querySelector("[data-testid='content']")).toBeNull();
    expect(widgets.size).toBe(1);
    const widgetNode = Array.from(widgets)[0].getDomNode();
    expect(widgetNode.querySelector("[data-testid='content']")?.textContent).toBe(
      "hello"
    );
    expect(widgetNode.style.left).toBe("52px");
    expect(widgetNode.style.width).toBe("734px");
    zone.onDomNodeTop?.(120);
    expect(widgetNode.style.top).toBe("120px");

    act(() => {
      root.render(
        <MonacoViewZone afterLineNumber={9} editor={editor}>
          <span data-testid="content">hello</span>
        </MonacoViewZone>
      );
    });
    expect(zones.size).toBe(1);
    expect(Array.from(zones.values())[0].afterLineNumber).toBe(9);

    act(() => root.unmount());
    expect(zones.size).toBe(0);
    expect(widgets.size).toBe(0);
  });
});

// Monaco parks the widget of a zone outside the rendered viewport far above
// the page, so the reveal must not depend on where the widget is.
const WIDGET_TOP = -1_000_000;

afterEach(() => vi.restoreAllMocks());

// Positions the widget off screen and the target `offset` px into it.
function stubRects(target: HTMLElement, offset: number, height: number) {
  vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(
    function (this: HTMLElement) {
      return this === target
        ? new DOMRect(0, WIDGET_TOP + offset, 700, height)
        : new DOMRect(0, WIDGET_TOP, 800, 0);
    }
  );
}

test.each([
  { name: "already visible", line: 4, height: 100, scrollTop: 50, expected: undefined },
  // Minimal scrolling: the box lands against the nearer edge, 8px in.
  { name: "below the viewport", line: 20, height: 120, scrollTop: 50, expected: lineTop(21) + 120 + 8 - 400 },
  { name: "above the viewport", line: 4, height: 120, scrollTop: 300, expected: lineTop(5) - 8 },
  { name: "taller than the viewport", line: 4, height: 600, scrollTop: 300, expected: lineTop(5) - 8 },
])("reveals a zone $name with one editor scroll", ({line, height, scrollTop, expected}) => {
  const {editor, scroll} = fakeEditor({scrollTop});
  const target = document.createElement("div");
  stubRects(target, 0, height);
  const revealTarget = {current: target};
  const root = createRoot(document.createElement("div"));
  const render = (key: string, text = "comment") => act(() => root.render(<MonacoViewZone afterLineNumber={line} editor={editor} revealKey={key} revealTarget={revealTarget}>{text}</MonacoViewZone>));
  render("first");
  if (expected === undefined) expect(scroll).not.toHaveBeenCalled();
  else expect(scroll).toHaveBeenCalledExactlyOnceWith(expected);
  // Typing/rerendering the same comment must not pull the reader back.
  scroll.mockClear();
  render("first", "longer comment");
  expect(scroll).not.toHaveBeenCalled();
  act(() => root.unmount());
  expect(scroll).not.toHaveBeenCalled();
});

test("reveals from the first line of the range the content is about", () => {
  // Line 12's zone is in view; lines 9-11 above it are scrolled past.
  const {editor, scroll} = fakeEditor({scrollTop: 250});
  const target = document.createElement("div");
  stubRects(target, 0, 100);
  const root = createRoot(document.createElement("div"));
  act(() => root.render(<MonacoViewZone afterLineNumber={12} editor={editor} revealKey="a" revealTarget={{current: target}}>comment</MonacoViewZone>));
  expect(scroll).not.toHaveBeenCalled();
  act(() => root.render(<MonacoViewZone afterLineNumber={12} editor={editor} revealFromLine={9} revealKey="b" revealTarget={{current: target}}>comment</MonacoViewZone>));
  expect(scroll).toHaveBeenCalledExactlyOnceWith(lineTop(9) - 8);
  act(() => root.unmount());
});

test("content revealing a part of itself scrolls to that part, not to the range", () => {
  const {editor, scroll} = fakeEditor({scrollTop: 250});
  const footer = document.createElement("div");
  stubRects(footer, 320, 40);
  function Footer() {
    const reveal = useContext(MonacoViewZoneRevealContext);
    useEffect(() => reveal?.(footer), [reveal]);
    return null;
  }
  const root = createRoot(document.createElement("div"));
  act(() => root.render(<MonacoViewZone afterLineNumber={12} editor={editor} revealFromLine={9}><Footer /></MonacoViewZone>));
  // The footer (296 + 320 → 616..656) crosses the 250..650 viewport's bottom.
  expect(scroll).toHaveBeenCalledExactlyOnceWith(lineTop(13) + 320 + 40 + 8 - 400);
  act(() => root.unmount());
});

test("a range taller than the viewport keeps its last line and the top of the content in view", () => {
  const {editor, scroll} = fakeEditor();
  const target = document.createElement("div");
  stubRects(target, 0, 100);
  const root = createRoot(document.createElement("div"));
  act(() => root.render(<MonacoViewZone afterLineNumber={40} editor={editor} revealFromLine={1} revealKey="a" revealTarget={{current: target}}>comment</MonacoViewZone>));
  // The whole content (100px) peeks above the bottom inset.
  expect(scroll).toHaveBeenCalledExactlyOnceWith(lineTop(41) + 100 + 8 - 400);
  act(() => root.unmount());
});

test("accounts for sibling zones stacked before it and the target's offset in the content", () => {
  const {addZoneBefore, editor, scroll} = fakeEditor({scrollTop: 600});
  addZoneBefore(4, 50);
  const target = document.createElement("div");
  stubRects(target, 30, 100);
  const root = createRoot(document.createElement("div"));
  act(() => root.render(<MonacoViewZone afterLineNumber={4} editor={editor} revealKey="a" revealTarget={{current: target}}>comment</MonacoViewZone>));
  expect(scroll).toHaveBeenCalledExactlyOnceWith(lineTop(5) + 50 + 30 - 8);
  act(() => root.unmount());
});
