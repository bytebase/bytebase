import { cn } from "@/lib/utils";

interface ColumnResizeHandleProps {
  onMouseDown: (e: React.MouseEvent) => void;
  className?: string;
}

/**
 * Drag affordance for resizing a table column. Designed to be absolutely
 * positioned inside a `relative` `<th>`.
 */
export function ColumnResizeHandle({
  onMouseDown,
  className,
}: ColumnResizeHandleProps) {
  return (
    <div
      className={cn(
        "group absolute right-0 top-0 z-10 h-full w-3 cursor-col-resize",
        className
      )}
      onMouseDown={onMouseDown}
      onClick={(e) => e.stopPropagation()}
    >
      <span className="pointer-events-none absolute right-0 top-1/4 h-1/2 w-0.5 rounded-full bg-control-bg-hover transition-colors group-hover:bg-accent/60 group-active:bg-accent" />
    </div>
  );
}
