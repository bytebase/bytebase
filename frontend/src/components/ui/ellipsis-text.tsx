import type { ReactNode } from "react";
import { useEffect, useRef, useState } from "react";
import { BlockTooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

interface EllipsisTextProps {
  readonly text: string;
  readonly className?: string;
  readonly children?: ReactNode;
}

/**
 * Renders text (or `children`) with CSS truncation. Shows a tooltip with the
 * full `text` only when the rendered content is actually overflowing.
 *
 * `text` is the source-of-truth string used as both the default rendered
 * content and the tooltip body. Pass `children` to render a richer node
 * (e.g. <HighlightLabelText>) while still tooltip-ing the plain text.
 */
export function EllipsisText({ text, className, children }: EllipsisTextProps) {
  const ref = useRef<HTMLSpanElement>(null);
  const [isTruncated, setIsTruncated] = useState(false);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const check = () => {
      setIsTruncated(el.scrollWidth > el.clientWidth);
    };
    check();
    const ro = new ResizeObserver(check);
    ro.observe(el);
    return () => ro.disconnect();
  }, [text, children]);

  return (
    <BlockTooltip
      content={text}
      delayDuration={300}
      open={isTruncated && open}
      onOpenChange={setOpen}
      popupClassName="max-w-none whitespace-nowrap"
      render={<span ref={ref} className={cn("block truncate", className)} />}
    >
      {children ?? text}
    </BlockTooltip>
  );
}
