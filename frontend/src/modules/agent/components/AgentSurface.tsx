import {
  type CSSProperties,
  type ReactNode,
  type Ref,
  useLayoutEffect,
  useState,
} from "react";
import { createPortal } from "react-dom";
import { getLayerRoot } from "@/components/ui/layer";
import type { AgentSqlEditorHost } from "../logic/sqlEditorHost";
import { MIN_WIDTH } from "../window";

type Presentation = "floating" | "sql-editor-docked" | "sql-editor-overlay";

type DockBounds = Pick<DOMRect, "left" | "top" | "width" | "height">;
const DOCK_MIN_COLUMN_WIDTH = 960;
const EDITOR_MIN_WIDTH = 480;
const DOCK_MAX_WIDTH = 640;

export function useAgentDockBounds(host: AgentSqlEditorHost | undefined) {
  const [dock, setDock] = useState<{
    host: AgentSqlEditorHost;
    bounds: DockBounds;
  }>();

  useLayoutEffect(() => {
    if (!host) return;
    const { mainColumn, resizer, slot } = host;
    const originalWidth = slot.style.width;
    const originalShrink = slot.style.flexShrink;
    function releaseSlot() {
      slot.style.width = originalWidth;
      slot.style.flexShrink = originalShrink;
    }
    const measure = () => {
      const available = mainColumn.getBoundingClientRect().width;
      const maxWidth = Math.min(
        DOCK_MAX_WIDTH,
        available - EDITOR_MIN_WIDTH - resizer.getBoundingClientRect().width
      );
      if (
        !slot.isConnected ||
        available < DOCK_MIN_COLUMN_WIDTH ||
        maxWidth < MIN_WIDTH
      ) {
        releaseSlot();
        setDock(undefined);
        return;
      }
      const width = Math.min(
        maxWidth,
        Math.max(MIN_WIDTH, Math.round(available * host.paneRatio))
      );
      slot.style.width = `${width}px`;
      slot.style.flexShrink = "0";
      const {
        left,
        top,
        width: actualWidth,
        height,
      } = slot.getBoundingClientRect();
      setDock((prev) =>
        prev?.host === host &&
        prev.bounds.left === left &&
        prev.bounds.top === top &&
        prev.bounds.width === actualWidth &&
        prev.bounds.height === height
          ? prev
          : { host, bounds: { left, top, width: actualWidth, height } }
      );
    };
    measure();
    const observer = new ResizeObserver(measure);
    for (const element of [mainColumn, resizer, slot])
      observer.observe(element);
    window.addEventListener("resize", measure);
    window.addEventListener("scroll", measure, true);
    return () => {
      observer.disconnect();
      window.removeEventListener("resize", measure);
      window.removeEventListener("scroll", measure, true);
      releaseSlot();
    };
  }, [host]);

  return dock && dock.host === host ? dock.bounds : undefined;
}

export function AgentSurface({
  presentation,
  style,
  ref,
  children,
}: {
  presentation: Presentation;
  style?: CSSProperties;
  ref?: Ref<HTMLDivElement>;
  children: ReactNode;
}) {
  // The SQL slot reserves space, but normal dialogs can inert its ancestors.
  // Keep the frame in the Agent family, below only critical authentication.
  return createPortal(
    <div
      ref={ref}
      data-agent-window
      data-agent-presentation={presentation}
      className="fixed overflow-visible"
      style={style}
    >
      {children}
    </div>,
    getLayerRoot("agent")
  );
}
