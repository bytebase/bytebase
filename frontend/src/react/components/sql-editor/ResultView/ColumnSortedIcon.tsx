import { ArrowDownWideNarrow, ArrowUpWideNarrow } from "lucide-react";
import { cn } from "@/react/lib/utils";
import type { SortDirection } from "./types";

interface ColumnSortedIconProps {
  isSorted: SortDirection;
}

export function ColumnSortedIcon({ isSorted }: ColumnSortedIconProps) {
  const showAsc = isSorted === "asc";
  // Explicit `size-3.5` on both wrapper and icon so Lucide's intrinsic
  // `width="24" height="24"` SVG attributes can't beat the CSS in any
  // browser. Matches the Vue version's compact 14px sort glyph.
  return (
    <span className="inline-flex size-3.5 opacity-80 shrink-0">
      {showAsc ? (
        <ArrowUpWideNarrow className="size-3.5 text-accent" />
      ) : (
        <ArrowDownWideNarrow
          className={cn(
            "size-3.5",
            isSorted === "desc" ? "text-accent" : "text-control-placeholder"
          )}
        />
      )}
    </span>
  );
}
