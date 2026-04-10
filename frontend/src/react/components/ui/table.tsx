import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";
import type { ComponentPropsWithoutRef } from "react";
import { cn } from "@/react/lib/utils";

function Table({ className, ...props }: ComponentPropsWithoutRef<"table">) {
  return (
    <table
      className={cn("w-full caption-bottom text-sm", className)}
      {...props}
    />
  );
}

function TableHeader({
  className,
  ...props
}: ComponentPropsWithoutRef<"thead">) {
  return <thead className={cn("[&_tr]:border-b", className)} {...props} />;
}

function TableBody({ className, ...props }: ComponentPropsWithoutRef<"tbody">) {
  return (
    <tbody className={cn("[&_tr:last-child]:border-0", className)} {...props} />
  );
}

function TableRow({ className, ...props }: ComponentPropsWithoutRef<"tr">) {
  return (
    <tr
      className={cn(
        "border-b border-block-border transition-colors hover:bg-control-bg/60 data-[state=selected]:bg-control-bg",
        className
      )}
      {...props}
    />
  );
}

export type TableHeadSortDirection = "asc" | "desc";

interface TableHeadProps extends ComponentPropsWithoutRef<"th"> {
  /** Show a sort indicator and make the header clickable. */
  sortable?: boolean;
  /** Whether this column is the currently active sort column. */
  sortActive?: boolean;
  /** Current direction when `sortActive` is true. */
  sortDir?: TableHeadSortDirection;
  /** Called when the user clicks to toggle sort. */
  onSort?: () => void;
}

function TableHead({
  className,
  children,
  sortable,
  sortActive,
  sortDir,
  onSort,
  onClick,
  ...props
}: TableHeadProps) {
  return (
    <th
      className={cn(
        "h-10 px-4 py-2 text-left align-middle font-medium text-control-light",
        sortable && "cursor-pointer select-none hover:text-control",
        className
      )}
      onClick={(e) => {
        onClick?.(e);
        if (sortable) onSort?.();
      }}
      {...props}
    >
      {sortable ? (
        <span className="inline-flex items-center gap-x-1">
          {children}
          <SortIndicator active={!!sortActive} dir={sortDir} />
        </span>
      ) : (
        children
      )}
    </th>
  );
}

function SortIndicator({
  active,
  dir,
}: {
  active: boolean;
  dir?: TableHeadSortDirection;
}) {
  const Icon = active ? (dir === "asc" ? ArrowUp : ArrowDown) : ArrowUpDown;
  return (
    <Icon
      className={cn(
        "w-3.5 h-3.5",
        active ? "text-accent" : "text-control-placeholder"
      )}
    />
  );
}

function TableCell({ className, ...props }: ComponentPropsWithoutRef<"td">) {
  return (
    <td
      className={cn("px-4 py-3 align-middle text-sm text-control", className)}
      {...props}
    />
  );
}

export { Table, TableBody, TableCell, TableHead, TableHeader, TableRow };
