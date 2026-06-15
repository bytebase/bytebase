import type { ComponentProps } from "react";
import { Button } from "@/react/components/ui/button";
import { cn } from "@/react/lib/utils";

/**
 * Trigger button shared by the SQL Editor connection controls
 * (`DatabaseChooser` + `ConnectChooser`), which render side by side as one
 * joined, accent-outlined segmented group. Wraps the shared <Button> so both
 * inherit its Base UI button semantics (focus/disabled handling) while keeping
 * the segmented-group look — no Button variant matches it, so the accent
 * border/text and group rounding come through className (merged last).
 */
export function ConnectionChooserButton({
  className,
  ...props
}: ComponentProps<typeof Button>) {
  return (
    <Button
      type="button"
      variant="outline"
      className={cn(
        "h-8 justify-end gap-1 px-2",
        "border-accent text-accent hover:bg-accent/5 focus:bg-accent/5",
        // Joined group: square inner edges, only the group's outer corners
        // round, and inner borders collapse so adjacent segments share one.
        "rounded-none first:rounded-l-xs last:rounded-r-xs",
        "[&:not(:last-child)]:border-r-0",
        // Keep the group flat — suppress Button's offset focus ring, which
        // would otherwise draw around a single segment.
        "focus-visible:ring-0",
        className
      )}
      {...props}
    />
  );
}
