import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { MonacoViewZone } from "./MonacoViewZone";
import type { IStandaloneCodeEditor } from "./types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

function fakeEditor() {
  const zones = new Map<
    string,
    {
      afterLineNumber: number;
      domNode: HTMLElement;
      onDomNodeTop?: (top: number) => void;
    }
  >();
  let nextId = 0;
  const widgets = new Set<{ getDomNode: () => HTMLElement }>();
  const editor = {
    addOverlayWidget: (widget: { getDomNode: () => HTMLElement }) => {
      widgets.add(widget);
    },
    removeOverlayWidget: (widget: { getDomNode: () => HTMLElement }) => {
      widgets.delete(widget);
    },
    getLayoutInfo: () => ({
      contentLeft: 52,
      width: 800,
      verticalScrollbarWidth: 14,
      minimap: { minimapWidth: 0 },
    }),
    onDidLayoutChange: () => ({ dispose: vi.fn() }),
    getModel: () => ({}),
    changeViewZones: (
      callback: (accessor: {
        addZone: (zone: {
          afterLineNumber: number;
          domNode: HTMLElement;
          onDomNodeTop?: (top: number) => void;
        }) => string;
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
  return { editor: editor as unknown as IStandaloneCodeEditor, widgets, zones };
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


afterEach(() => vi.unstubAllGlobals());

test.each([
  { name: "already visible", top: 150, height: 100, expected: undefined },
  { name: "below the viewport", top: 480, height: 120, expected: 208 },
  { name: "above the viewport", top: 50, height: 120, expected: 42 },
  { name: "taller than the viewport", top: 450, height: 600, expected: 442 },
])("reveals $name with minimal editor scrolling", ({top, height, expected}) => {
  const frames = new Map<number, FrameRequestCallback>();
  let nextFrame = 0;
  vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => { frames.set(++nextFrame, callback); return nextFrame; });
  vi.stubGlobal("cancelAnimationFrame", (id: number) => frames.delete(id));
  const runFrame = () => { const queued = [...frames.values()]; frames.clear(); act(() => queued.forEach(callback => callback(0))); };
  const host = document.createElement("div");
  host.getBoundingClientRect = () => new DOMRect(0, 100, 800, 400);
  const target = document.createElement("div");
  target.getBoundingClientRect = () => new DOMRect(0, top, 700, height);
  const revealTarget = {current: target};
  const {editor} = fakeEditor();
  const scroll = vi.fn();
  Object.assign(editor, {getDomNode: () => host, getScrollTop: () => 100, setScrollTop: scroll});
  const root = createRoot(document.createElement("div"));
  const render = (key: string, text = "comment") => act(() => root.render(<MonacoViewZone afterLineNumber={4} editor={editor} revealKey={key} revealTarget={revealTarget}>{text}</MonacoViewZone>));
  render("first");
  runFrame();
  expect(scroll).not.toHaveBeenCalled();
  runFrame();
  if (expected === undefined) expect(scroll).not.toHaveBeenCalled();
  else expect(scroll).toHaveBeenCalledExactlyOnceWith(expected);
  // Typing/rerendering the same comment must not pull the reader back.
  scroll.mockClear();
  render("first", "longer comment");
  runFrame(); runFrame();
  expect(scroll).not.toHaveBeenCalled();
  render("second");
  runFrame();
  act(() => root.unmount());
  runFrame();
  expect(scroll).not.toHaveBeenCalled();
});
