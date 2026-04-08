import { useCallback, useRef, useState } from "react";

interface EllipsisTextProps {
  readonly text: string;
  readonly className?: string;
}

/**
 * Renders text with CSS truncation. Shows a tooltip with the full text only
 * when the text is actually overflowing (i.e. truncated with ellipsis).
 */
export function EllipsisText({ text, className }: EllipsisTextProps) {
  const textRef = useRef<HTMLSpanElement>(null);
  const [showTooltip, setShowTooltip] = useState(false);

  const handleMouseEnter = useCallback(() => {
    const el = textRef.current;
    if (el && el.scrollWidth > el.clientWidth) {
      setShowTooltip(true);
    }
  }, []);

  const handleMouseLeave = useCallback(() => {
    setShowTooltip(false);
  }, []);

  return (
    <span
      ref={textRef}
      className={`relative block truncate ${className ?? ""}`}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      {text}
      {showTooltip && (
        <span
          role="tooltip"
          className="absolute left-1/2 bottom-full mb-1.5 -translate-x-1/2 z-50 rounded-sm bg-gray-900 px-2.5 py-1.5 text-xs font-normal text-white shadow-md max-w-80 whitespace-normal pointer-events-none"
        >
          {text}
        </span>
      )}
    </span>
  );
}
