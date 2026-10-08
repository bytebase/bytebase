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

// Breathing room kept between a revealed box and the viewport edges, in px.
const REVEAL_INSET = 8;

// Hosts interactive React content between two editor lines the way VS Code's
// ZoneWidget does: a view zone reserves the vertical space (the zone layer
// sits under the text layer, so it cannot take clicks), and an overlay widget
// pinned to the zone's top carries the real content. The content wrapper is
// measured with a ResizeObserver and the zone re-laid out to match.
export function MonacoViewZone({
  afterLineNumber,
  children,
  editor,
  revealFromLine,
  revealKey,
  revealTarget,
}: {
  afterLineNumber: number;
  children: ReactNode;
  editor: IStandaloneCodeEditor;
  // The first line to keep in view with the revealed content, for content
  // that discusses the lines above it.
  revealFromLine?: number;
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
  const zoneRef = useRef<{
    id: string;
    afterLineNumber: number;
    fit: () => void;
  }>(undefined);
  const revealFromLineRef = useRef(revealFromLine);
  useLayoutEffect(() => {
    revealFromLineRef.current = revealFromLine;
  }, [revealFromLine]);

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
    zoneRef.current = { id: zoneId, afterLineNumber, fit };
    const observer = new ResizeObserver(fit);
    if (content) observer.observe(content);

    return () => {
      observer.disconnect();
      layoutSubscription.dispose();
      zoneRef.current = undefined;
      // The editor may already be disposed when the host unmounts.
      if (!editor.getModel()) return;
      editor.removeOverlayWidget(widget);
      editor.changeViewZones((accessor) => accessor.removeZone(zoneId));
    };
  }, [afterLineNumber, editor, widgetNode]);

  // Scrolls the editor once, synchronously, so the target (and the lines from
  // `fromLine` above it) is in view. Positions come from the editor's layout
  // rather than the DOM: Monaco places a zone's widget only when the zone is
  // inside the rendered viewport and only on its next render, and the scroll
  // must land before that render to avoid a visible jump.
  // Content revealing a part of itself, such as a reply form, passes no line.
  const reveal = useCallback(
    (target: HTMLElement, fromLine?: number) => {
      const zone = zoneRef.current;
      if (!zone || !editor.getModel()) return;
      // The zone may not have caught up with content that just grew.
      zone.fit();
      const rect = target.getBoundingClientRect();
      const viewportHeight = editor.getLayoutInfo().height;
      if (!rect.height || !viewportHeight) return;

      const targetTop =
        zoneTopOffset(editor, zone.afterLineNumber, zone.id) +
        rect.top -
        widgetNode.getBoundingClientRect().top;
      const targetBottom = targetTop + rect.height;
      const boxTop =
        (fromLine === undefined
          ? targetTop
          : Math.min(targetTop, editor.getTopForLineNumber(fromLine))) -
        REVEAL_INSET;
      const boxBottom = targetBottom + REVEAL_INSET;

      const scrollTop = editor.getScrollTop();
      const viewportBottom = scrollTop + viewportHeight;
      if (boxTop >= scrollTop && boxBottom <= viewportBottom) return;
      if (boxBottom - boxTop > viewportHeight) {
        // Anchor the top of the box, but keep enough of the target itself in
        // view that a long range does not push it off the bottom.
        const peek = Math.min(rect.height, viewportHeight / 3);
        editor.setScrollTop(
          Math.max(boxTop, targetTop + peek + REVEAL_INSET - viewportHeight)
        );
        return;
      }
      editor.setScrollTop(
        boxTop < scrollTop ? boxTop : boxBottom - viewportHeight
      );
    },
    [editor, widgetNode]
  );

  useLayoutEffect(() => {
    const target = revealTarget?.current ?? contentRef.current;
    if (revealKey !== undefined && target)
      reveal(target, revealFromLineRef.current);
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

// The editor widget lists its zones in layout order, but the method is not
// on the public editor interface.
type WhitespaceLister = IStandaloneCodeEditor & {
  getWhitespaces: () => {
    id: string;
    afterLineNumber: number;
    height: number;
  }[];
};

// The top of the zone within the editor's scrollable content: the bottom of
// its line, plus the zones Monaco stacks before it after that same line.
function zoneTopOffset(
  editor: IStandaloneCodeEditor,
  afterLineNumber: number,
  zoneId: string
): number {
  let top = editor.getBottomForLineNumber(afterLineNumber);
  for (const whitespace of (editor as WhitespaceLister).getWhitespaces()) {
    if (whitespace.afterLineNumber !== afterLineNumber) continue;
    if (whitespace.id === zoneId) break;
    top += whitespace.height;
  }
  return top;
}
