import { Bot } from "lucide-react";
import {
  type PointerEvent as ReactPointerEvent,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { WINDOW_MARGIN } from "../window";

export interface AgentLauncherPosition {
  right: number;
  bottom: number;
}

export function AgentMinimizedLauncher({
  initialPosition,
  onPositionChange,
  onOpen,
}: {
  initialPosition?: AgentLauncherPosition;
  onPositionChange?: (position: AgentLauncherPosition) => void;
  onOpen: () => void;
}) {
  const { t } = useTranslation();
  const buttonRef = useRef<HTMLButtonElement>(null);
  const [position, setPosition] = useState(
    initialPosition ?? { right: WINDOW_MARGIN, bottom: WINDOW_MARGIN }
  );
  const desiredPositionRef = useRef(position);
  const updatePositionRef = useRef(() => {});
  const dragRef = useRef<{
    pointerId: number;
    x: number;
    y: number;
    right: number;
    bottom: number;
  } | null>(null);
  const suppressClickRef = useRef(false);

  const stopDragging = (event: ReactPointerEvent<HTMLButtonElement>) => {
    if (dragRef.current?.pointerId !== event.pointerId) return;
    dragRef.current = null;
    event.currentTarget.releasePointerCapture(event.pointerId);
  };

  useLayoutEffect(() => {
    const button = buttonRef.current;
    if (!button) return;
    let footers: Element[] = [];
    const updatePosition = () => {
      const { width, height } = button.getBoundingClientRect();
      const viewportHeight = window.innerHeight;
      const rightOffset = Math.min(
        Math.max(WINDOW_MARGIN, window.innerWidth - WINDOW_MARGIN - width),
        Math.max(WINDOW_MARGIN, desiredPositionRef.current.right)
      );
      const right = window.innerWidth - rightOffset;
      const left = right - width;
      const maximumBottom = Math.max(
        WINDOW_MARGIN,
        viewportHeight - WINDOW_MARGIN - height
      );
      let nextBottom = Math.min(
        maximumBottom,
        Math.max(WINDOW_MARGIN, desiredPositionRef.current.bottom)
      );
      const footerRects = footers
        .filter((footer) => {
          const style = getComputedStyle(footer);
          return style.display !== "none" && style.visibility !== "hidden";
        })
        .map((footer) => footer.getBoundingClientRect())
        .filter((rect) => rect.width > 0 && rect.height > 0)
        .sort((a, b) => b.top - a.top);

      // Measure collisions from the resting position, not the displaced button.
      for (const rect of footerRects) {
        const buttonBottom = viewportHeight - nextBottom;
        if (
          rect.left < right &&
          rect.right > left &&
          rect.top < buttonBottom &&
          rect.bottom > buttonBottom - height
        ) {
          nextBottom = Math.min(
            maximumBottom,
            viewportHeight - rect.top + WINDOW_MARGIN
          );
        }
      }
      setPosition((current) =>
        current.right === rightOffset && current.bottom === nextBottom
          ? current
          : { right: rightOffset, bottom: nextBottom }
      );
    };
    updatePositionRef.current = updatePosition;
    const resizeObserver = new ResizeObserver(updatePosition);
    const syncFooters = () => {
      footers = [
        ...document.querySelectorAll('[data-slot="sticky-action-footer"]'),
      ];
      resizeObserver.disconnect();
      resizeObserver.observe(button);
      for (const footer of footers) resizeObserver.observe(footer);
      updatePosition();
    };
    const mutationObserver = new MutationObserver(syncFooters);
    mutationObserver.observe(document.body, { childList: true, subtree: true });
    window.addEventListener("resize", updatePosition);
    window.addEventListener("scroll", updatePosition, true);
    syncFooters();
    return () => {
      updatePositionRef.current = () => {};
      resizeObserver.disconnect();
      mutationObserver.disconnect();
      window.removeEventListener("resize", updatePosition);
      window.removeEventListener("scroll", updatePosition, true);
    };
  }, []);

  return (
    <Button
      ref={buttonRef}
      type="button"
      size="lg"
      data-agent-window
      aria-label={t("agent.self")}
      title={t("agent.self")}
      className="fixed w-10 cursor-grab touch-none select-none rounded-full px-0 shadow-lg active:cursor-grabbing"
      style={position}
      onPointerDown={(event) => {
        if (event.button !== 0 || !event.isPrimary || dragRef.current) return;
        const rect = event.currentTarget.getBoundingClientRect();
        dragRef.current = {
          pointerId: event.pointerId,
          x: event.clientX,
          y: event.clientY,
          right: window.innerWidth - rect.right,
          bottom: window.innerHeight - rect.bottom,
        };
        suppressClickRef.current = false;
        event.currentTarget.setPointerCapture(event.pointerId);
      }}
      onPointerMove={(event) => {
        const drag = dragRef.current;
        if (!drag || drag.pointerId !== event.pointerId) return;
        const deltaX = event.clientX - drag.x;
        const deltaY = event.clientY - drag.y;
        if (!suppressClickRef.current && Math.hypot(deltaX, deltaY) < 4) return;
        suppressClickRef.current = true;
        desiredPositionRef.current = {
          right: drag.right - deltaX,
          bottom: drag.bottom - deltaY,
        };
        onPositionChange?.(desiredPositionRef.current);
        updatePositionRef.current();
      }}
      onPointerUp={stopDragging}
      onPointerCancel={stopDragging}
      onLostPointerCapture={(event) => {
        if (dragRef.current?.pointerId === event.pointerId) {
          dragRef.current = null;
        }
      }}
      onClick={(event) => {
        if (suppressClickRef.current && event.detail !== 0) {
          suppressClickRef.current = false;
          event.preventDefault();
          return;
        }
        suppressClickRef.current = false;
        onOpen();
      }}
    >
      <Bot className="size-5" aria-hidden="true" />
    </Button>
  );
}
