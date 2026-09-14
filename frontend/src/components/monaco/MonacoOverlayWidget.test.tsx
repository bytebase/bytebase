import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, test, vi } from "vitest";
import { MonacoOverlayWidget } from "./MonacoOverlayWidget";
import type { IStandaloneCodeEditor } from "./types";

type Widget = { getDomNode: () => HTMLElement };

function createEditor() {
  const widgets = new Set<Widget>();
  let relayout = () => {};
  const layout = { verticalScrollbarWidth: 14 };
  const editor = {
    addOverlayWidget: (widget: Widget) => widgets.add(widget),
    removeOverlayWidget: (widget: Widget) => widgets.delete(widget),
    getLayoutInfo: () => layout,
    onDidLayoutChange: (listener: () => void) => {
      relayout = listener;
      return { dispose: vi.fn() };
    },
    getModel: () => ({}),
  } as unknown as IStandaloneCodeEditor;
  return { editor, layout, widgets, relayout: () => relayout() };
}

test("sits in the top-right corner past the scrollbar and follows layout changes", () => {
  const fake = createEditor();
  const container = document.createElement("div");
  const root = createRoot(container);
  act(() =>
    root.render(
      <MonacoOverlayWidget editor={fake.editor}>
        <span data-testid="content" />
      </MonacoOverlayWidget>
    )
  );
  expect(fake.widgets.size).toBe(1);
  const [widget] = fake.widgets;
  const node = widget.getDomNode();
  expect(node.querySelector("[data-testid='content']")).not.toBeNull();
  expect(node.style.zIndex).toBe("10");
  expect([node.style.top, node.style.right]).toEqual(["8px", "22px"]);

  fake.layout.verticalScrollbarWidth = 20;
  fake.relayout();
  expect(node.style.right).toBe("28px");

  act(() => root.unmount());
  expect(fake.widgets.size).toBe(0);
});

test("skips removal once the editor is disposed", () => {
  const fake = createEditor();
  (fake.editor as unknown as { getModel: () => null }).getModel = () => null;
  const removeOverlayWidget = vi.spyOn(fake.editor, "removeOverlayWidget");
  const container = document.createElement("div");
  const root = createRoot(container);
  act(() =>
    root.render(<MonacoOverlayWidget editor={fake.editor}>x</MonacoOverlayWidget>)
  );
  expect(fake.widgets.size).toBe(1);
  act(() => root.unmount());
  expect(removeOverlayWidget).not.toHaveBeenCalled();
});
