import { Tooltip as BaseTooltip } from "@base-ui/react/tooltip";
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
  const [isTruncated, setIsTruncated] = useState(false);

  const checkTruncation = useCallback(() => {
    const el = textRef.current;
    if (el) {
      setIsTruncated(el.scrollWidth > el.clientWidth);
    }
  }, []);

  return (
    <BaseTooltip.Provider delay={100}>
      <BaseTooltip.Root open={isTruncated ? undefined : false}>
        <BaseTooltip.Trigger
          render={<span className={`block truncate ${className ?? ""}`} />}
          ref={textRef}
          onMouseEnter={checkTruncation}
        >
          {text}
        </BaseTooltip.Trigger>
        <BaseTooltip.Portal>
          <BaseTooltip.Positioner side="top" sideOffset={4}>
            <BaseTooltip.Popup className="z-50 rounded-sm bg-gray-900 px-2.5 py-1.5 text-xs text-white shadow-md max-w-80">
              {text}
              <BaseTooltip.Arrow className="fill-gray-900" />
            </BaseTooltip.Popup>
          </BaseTooltip.Positioner>
        </BaseTooltip.Portal>
      </BaseTooltip.Root>
    </BaseTooltip.Provider>
  );
}
