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
    let transientWidth: number | undefined;
    let pointer: { id: number; x: number; width: number } | undefined;
    const settle = host.settlePaneRatio;
    function releaseSlot() {
      resizer.hidden = true;
      transientWidth = undefined;
      pointer = undefined;
      slot.style.width = originalWidth;
      slot.style.flexShrink = originalShrink;
    }
    const measure = () => {
      const available = mainColumn.getBoundingClientRect().width;
      resizer.hidden = !settle || available < DOCK_MIN_COLUMN_WIDTH;
      const maxWidth = Math.floor(
        Math.min(
          DOCK_MAX_WIDTH,
          available - EDITOR_MIN_WIDTH - resizer.getBoundingClientRect().width
        )
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
        Math.max(
          MIN_WIDTH,
          transientWidth ?? Math.round(available * host.paneRatio)
        )
      );
      resizer.setAttribute("aria-valuemin", String(MIN_WIDTH));
      resizer.setAttribute("aria-valuemax", String(maxWidth));
      resizer.setAttribute("aria-valuenow", String(width));
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
    const commit = () => {
      const width = slot.getBoundingClientRect().width;
      const available = mainColumn.getBoundingClientRect().width;
      if (available && !resizer.hidden) settle?.(width / available);
    };
    const down = (event: PointerEvent) => {
      if (resizer.hidden || event.button !== 0 || pointer) return;
      pointer = {
        id: event.pointerId,
        x: event.clientX,
        width: slot.getBoundingClientRect().width,
      };
      resizer.setPointerCapture(event.pointerId);
    };
    const move = (event: PointerEvent) => {
      if (pointer?.id !== event.pointerId || !pointer) return;
      transientWidth = Math.round(pointer.width + pointer.x - event.clientX);
      measure();
    };
    const end = (event: PointerEvent) => {
      if (!pointer || pointer.id !== event.pointerId) return;
      if (event.type === "pointerup" && pointer.x !== event.clientX) {
        move(event);
        commit();
      }
      pointer = undefined;
      transientWidth = undefined;
      measure();
    };
    const key = (event: KeyboardEvent) => {
      if (resizer.hidden || pointer) return;
      const width = slot.getBoundingClientRect().width;
      switch (event.key) {
        case "ArrowLeft":
          transientWidth = width + 16;
          break;
        case "ArrowRight":
          transientWidth = width - 16;
          break;
        case "Home":
          transientWidth = MIN_WIDTH;
          break;
        case "End":
          transientWidth = DOCK_MAX_WIDTH;
          break;
        default:
          return;
      }
      event.preventDefault();
      measure();
      commit();
      transientWidth = undefined;
    };
    resizer.addEventListener("pointerdown", down);
    resizer.addEventListener("pointermove", move);
    resizer.addEventListener("pointerup", end);
    resizer.addEventListener("pointercancel", end);
    resizer.addEventListener("lostpointercapture", end);
    resizer.addEventListener("keydown", key);
    measure();
    const observer = new ResizeObserver(measure);
    for (const element of [mainColumn, resizer, slot])
      observer.observe(element);
    window.addEventListener("resize", measure);
    window.addEventListener("scroll", measure, true);
    return () => {
      resizer.removeEventListener("pointerdown", down);
      resizer.removeEventListener("pointermove", move);
      resizer.removeEventListener("pointerup", end);
      resizer.removeEventListener("pointercancel", end);
      resizer.removeEventListener("lostpointercapture", end);
      resizer.removeEventListener("keydown", key);
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
