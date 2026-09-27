import { type ReactNode, useId, useLayoutEffect, useMemo } from "react";
import { createPortal } from "react-dom";
import type { IStandaloneCodeEditor } from "./types";

// Inset from the editor's top edge and from the scrollbar lane, in px.
const CORNER_INSET = 8;

// Hosts React content in the editor's top-right corner through Monaco's
// overlay-widget slot, so it stays put while the editor scrolls. Positioned
// here rather than by Monaco's corner preference so it sits right beside
// the scrollbar and above the view-zone widgets added after it.
export function MonacoOverlayWidget({
  children,
  editor,
}: {
  children: ReactNode;
  editor: IStandaloneCodeEditor;
}) {
  const id = useId();
  const domNode = useMemo(() => {
    const node = document.createElement("div");
    node.style.pointerEvents = "auto";
    // Above the view-zone widgets, which Monaco stacks by insertion order.
    node.style.zIndex = "10";
    return node;
  }, []);

  useLayoutEffect(() => {
    const widget = {
      getId: () => `bytebase.overlay-widget.${id}`,
      getDomNode: () => domNode,
      getPosition: () => ({ preference: null }),
    };
    const layout = () => {
      const info = editor.getLayoutInfo();
      domNode.style.top = `${CORNER_INSET}px`;
      domNode.style.right = `${info.verticalScrollbarWidth + CORNER_INSET}px`;
    };
    layout();
    editor.addOverlayWidget(widget);
    const subscription = editor.onDidLayoutChange(layout);
    return () => {
      subscription.dispose();
      // The editor may already be disposed when the host unmounts.
      if (!editor.getModel()) return;
      editor.removeOverlayWidget(widget);
    };
  }, [domNode, editor, id]);

  return createPortal(children, domNode);
}
