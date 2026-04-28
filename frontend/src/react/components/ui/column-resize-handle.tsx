import { cn } from "@/react/lib/utils";

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
        "absolute right-0 top-1/4 h-1/2 w-[3px] cursor-col-resize rounded-full bg-control-bg-hover hover:bg-accent/60 active:bg-accent transition-colors",
        className
      )}
      onMouseDown={onMouseDown}
    />
  );
}
