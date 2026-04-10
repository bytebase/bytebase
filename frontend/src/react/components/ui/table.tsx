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

function TableHead({ className, ...props }: ComponentPropsWithoutRef<"th">) {
  return (
    <th
      className={cn(
        "h-10 px-4 py-2 text-left align-middle font-medium text-control-light",
        className
      )}
      {...props}
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
