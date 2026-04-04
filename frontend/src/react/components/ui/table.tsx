import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";

function Table({ className, ref, ...props }: ComponentProps<"table">) {
  return (
    <table ref={ref} className={cn("w-full text-sm", className)} {...props} />
  );
}

function TableHeader({ className, ref, ...props }: ComponentProps<"thead">) {
  return <thead ref={ref} className={cn("", className)} {...props} />;
}

function TableBody({ className, ref, ...props }: ComponentProps<"tbody">) {
  return <tbody ref={ref} className={cn("", className)} {...props} />;
}

function TableRow({ className, ref, ...props }: ComponentProps<"tr">) {
  return (
    <tr
      ref={ref}
      className={cn("border-b last:border-b-0", className)}
      {...props}
    />
  );
}

function TableHead({
  className,
  resizable = false,
  onResizeStart,
  ref,
  ...props
}: ComponentProps<"th"> & {
  resizable?: boolean;
  onResizeStart?: (e: React.MouseEvent) => void;
}) {
  return (
    <th
      ref={ref}
      className={cn(
        "relative px-4 py-2 text-left font-medium whitespace-nowrap",
        className
      )}
      {...props}
    >
      {props.children}
      {resizable && onResizeStart && (
        <div
          className="absolute right-0 top-1/4 h-1/2 w-[3px] cursor-col-resize rounded-full bg-gray-200 hover:bg-accent/60 active:bg-accent transition-colors"
          onMouseDown={onResizeStart}
        />
      )}
    </th>
  );
}

function TableCell({ className, ref, ...props }: ComponentProps<"td">) {
  return <td ref={ref} className={cn("px-4 py-2", className)} {...props} />;
}

export { Table, TableHeader, TableBody, TableRow, TableHead, TableCell };
