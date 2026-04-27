import { useCallback, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { HighlightLabelText } from "@/react/components/HighlightLabelText";
import { getLayerRoot } from "@/react/components/ui/layer";
import { cn } from "@/react/lib/utils";

interface EllipsisCellProps {
  readonly content: string;
  readonly keyword?: string;
  readonly tooltip?: string;
  readonly className?: string;
}

/**
 * React port of `frontend/src/views/sql-editor/EditorPanel/common/EllipsisCell.vue`.
 * Renders truncated text with keyword highlighting; on hover, when the
 * text is actually overflowing, a tooltip portals out the full content
 * (also highlighted, or a custom `tooltip` string when provided).
 */
export function EllipsisCell({
  content,
  keyword,
  tooltip,
  className,
}: EllipsisCellProps) {
  const ref = useRef<HTMLSpanElement>(null);
  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState({ x: 0, y: 0 });

  const handleMouseEnter = useCallback(() => {
    const el = ref.current;
    if (el && el.scrollWidth > el.clientWidth) {
      const rect = el.getBoundingClientRect();
      setPos({ x: rect.left + rect.width / 2, y: rect.top });
      setOpen(true);
    }
  }, []);

  const handleMouseLeave = useCallback(() => {
    setOpen(false);
  }, []);

  useEffect(() => {
    if (!open) return;
    const close = () => setOpen(false);
    window.addEventListener("scroll", close, true);
    return () => window.removeEventListener("scroll", close, true);
  }, [open]);

  return (
    <span
      ref={ref}
      className={cn("block truncate", className)}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      <HighlightLabelText text={content} keyword={keyword} />
      {open &&
        createPortal(
          <span
            role="tooltip"
            style={{
              position: "fixed",
              left: pos.x,
              top: pos.y,
              transform: "translate(-50%, -100%) translateY(-6px)",
              maxWidth: "min(33vw, 320px)",
            }}
            className="rounded-sm bg-gray-900 px-2.5 py-1.5 text-xs font-normal text-white shadow-md whitespace-pre-wrap break-all pointer-events-none"
          >
            <HighlightLabelText text={tooltip ?? content} keyword={keyword} />
          </span>,
          getLayerRoot("overlay")
        )}
    </span>
  );
}
