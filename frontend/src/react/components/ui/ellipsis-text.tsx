import { Tooltip as BaseTooltip } from "@base-ui/react/tooltip";
import type { ReactNode } from "react";
import { useEffect, useRef, useState } from "react";
import { cn } from "@/react/lib/utils";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "./layer";

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
    <BaseTooltip.Provider delay={300}>
      <BaseTooltip.Root open={isTruncated && open} onOpenChange={setOpen}>
        <BaseTooltip.Trigger
          render={
            <span ref={ref} className={cn("block truncate", className)} />
          }
        >
          {children ?? text}
        </BaseTooltip.Trigger>
        <BaseTooltip.Portal container={getLayerRoot("overlay")}>
          <BaseTooltip.Positioner
            side="top"
            sideOffset={4}
            className={LAYER_SURFACE_CLASS}
          >
            <BaseTooltip.Popup className="rounded-sm bg-main px-2.5 py-1.5 text-xs text-main-text shadow-md whitespace-nowrap">
              {text}
              <BaseTooltip.Arrow className="fill-main" />
            </BaseTooltip.Popup>
          </BaseTooltip.Positioner>
        </BaseTooltip.Portal>
      </BaseTooltip.Root>
    </BaseTooltip.Provider>
  );
}
