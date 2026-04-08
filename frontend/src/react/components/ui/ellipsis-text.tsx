import { useCallback, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";

interface EllipsisTextProps {
  readonly text: string;
  readonly className?: string;
}

/**
 * Renders text with CSS truncation. Shows a tooltip with the full text only
 * when the text is actually overflowing (i.e. truncated with ellipsis).
 * The tooltip is portaled to document.body to avoid clipping by overflow:hidden.
 */
export function EllipsisText({ text, className }: EllipsisTextProps) {
  const textRef = useRef<HTMLSpanElement>(null);
  const [showTooltip, setShowTooltip] = useState(false);
  const [pos, setPos] = useState({ x: 0, y: 0 });

  const handleMouseEnter = useCallback(() => {
    const el = textRef.current;
    if (el && el.scrollWidth > el.clientWidth) {
      const rect = el.getBoundingClientRect();
      setPos({
        x: rect.left + rect.width / 2,
        y: rect.top,
      });
      setShowTooltip(true);
    }
  }, []);

  const handleMouseLeave = useCallback(() => {
    setShowTooltip(false);
  }, []);

  useEffect(() => {
    if (!showTooltip) return;
    const onScroll = () => setShowTooltip(false);
    window.addEventListener("scroll", onScroll, true);
    return () => window.removeEventListener("scroll", onScroll, true);
  }, [showTooltip]);

  return (
    <span
      ref={textRef}
      className={`block truncate ${className ?? ""}`}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      {text}
      {showTooltip &&
        createPortal(
          <span
            role="tooltip"
            style={{
              position: "fixed",
              left: pos.x,
              top: pos.y,
              transform: "translate(-50%, -100%) translateY(-6px)",
            }}
            className="z-50 rounded-sm bg-gray-900 px-2.5 py-1.5 text-xs font-normal text-white shadow-md max-w-80 whitespace-normal pointer-events-none"
          >
            {text}
          </span>,
          document.body
        )}
    </span>
  );
}
