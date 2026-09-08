import {
  createContext,
  type ReactNode,
  type RefObject,
  useCallback,
  useLayoutEffect,
  useMemo,
  useRef,
} from "react";
import { createPortal } from "react-dom";
import type { IStandaloneCodeEditor } from "./types";

export const MonacoViewZoneRevealContext = createContext<
  ((target: HTMLElement) => void) | undefined
>(undefined);

// Hosts interactive React content between two editor lines the way VS Code's
// ZoneWidget does: a view zone reserves the vertical space (the zone layer
// sits under the text layer, so it cannot take clicks), and an overlay widget
// pinned to the zone's top carries the real content. The content wrapper is
// measured with a ResizeObserver and the zone re-laid out to match.
export function MonacoViewZone({
  afterLineNumber,
  children,
  editor,
  revealKey,
  revealTarget,
}: {
  afterLineNumber: number;
  children: ReactNode;
  editor: IStandaloneCodeEditor;
  // Reveal once on opening/changing content, never on every resize or keystroke.
  revealKey?: unknown;
  revealTarget?: RefObject<HTMLElement | null>;
}) {
  const widgetNode = useMemo(() => {
    const node = document.createElement("div");
    node.style.pointerEvents = "auto";
    return node;
  }, []);
  const contentRef = useRef<HTMLDivElement>(null);
  const revealFrame = useRef<number | undefined>(undefined);

  useLayoutEffect(() => {
    const content = contentRef.current;
    const zoneNode = document.createElement("div");
    const zone = {
      afterLineNumber,
      domNode: zoneNode,
      heightInPx: 1,
      onDomNodeTop: (top: number) => {
        widgetNode.style.top = `${top}px`;
      },
    };
    let zoneId = "";
    editor.changeViewZones((accessor) => {
      zoneId = accessor.addZone(zone);
    });
    const widget = {
      getId: () => `bytebase.view-zone.${zoneId}`,
      getDomNode: () => widgetNode,
      getPosition: () => null,
    };
    editor.addOverlayWidget(widget);

    // Span the content column: after the gutter, before the scrollbar.
    const layout = () => {
      const info = editor.getLayoutInfo();
      widgetNode.style.left = `${info.contentLeft}px`;
      widgetNode.style.width = `${Math.max(
        0,
        info.width -
          info.contentLeft -
          info.verticalScrollbarWidth -
          info.minimap.minimapWidth
      )}px`;
    };
    layout();
    const layoutSubscription = editor.onDidLayoutChange(layout);

    const fit = () => {
      const next = content?.offsetHeight ?? 0;
      if (!next || next === zone.heightInPx) return;
      zone.heightInPx = next;
      editor.changeViewZones((accessor) => accessor.layoutZone(zoneId));
    };
    // The content is only measurable once the widget is in the editor's DOM
    // and has its column width. Size the zone now, before the editor's next
    // render, so the first frame never shows the content over the lines
    // below it at a placeholder height.
    fit();
    const observer = new ResizeObserver(fit);
    if (content) observer.observe(content);

    return () => {
      observer.disconnect();
      layoutSubscription.dispose();
      // The editor may already be disposed when the host unmounts.
      if (!editor.getModel()) return;
      editor.removeOverlayWidget(widget);
      editor.changeViewZones((accessor) => accessor.removeZone(zoneId));
    };
  }, [afterLineNumber, editor, widgetNode]);

  const reveal = useCallback(
    (target: HTMLElement) => {
      if (revealFrame.current !== undefined)
        cancelAnimationFrame(revealFrame.current);
      // Wait for the zone height to include newly opened forms.
      revealFrame.current = requestAnimationFrame(() => {
        revealFrame.current = requestAnimationFrame(() => {
          const host = editor.getDomNode();
          if (!host || !editor.getModel()) return;
          const bounds = target.getBoundingClientRect();
          const viewport = host.getBoundingClientRect();
          if (!bounds.height || !viewport.height) return;
          const top = viewport.top + 8;
          const bottom = viewport.bottom - 8;
          const delta =
            bounds.height > bottom - top || bounds.top < top
              ? bounds.top - top
              : Math.max(0, bounds.bottom - bottom);
          if (delta) editor.setScrollTop(editor.getScrollTop() + delta);
        });
      });
    },
    [editor]
  );

  useLayoutEffect(() => {
    const target = revealTarget?.current ?? contentRef.current;
    if (revealKey !== undefined && target) reveal(target);
    return () => {
      if (revealFrame.current !== undefined)
        cancelAnimationFrame(revealFrame.current);
    };
  }, [reveal, revealKey, revealTarget]);

  return createPortal(
    <div data-testid="monaco-view-zone" ref={contentRef}>
      <MonacoViewZoneRevealContext value={reveal}>
        {children}
      </MonacoViewZoneRevealContext>
    </div>,
    widgetNode
  );
}
