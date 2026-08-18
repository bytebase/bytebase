import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

/**
 * A bordered group of settings rows.
 *
 * Grouping into a card does the work that section headings and whitespace were
 * doing badly: the border says where a group starts and ends, so the rows
 * inside can sit close together without reading as one undifferentiated list.
 */
export function SettingsCard({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "divide-y divide-control-border rounded-md border border-control-border bg-background",
        className
      )}
    >
      {children}
    </div>
  );
}

/**
 * One setting: what it is on the left, its value or control on the right.
 *
 * The label column carries the reading weight and the control column stays
 * narrow, so a text input does not stretch to the width of the page just
 * because the page is wide.
 */
export function SettingsRow({
  label,
  description,
  children,
  align = "center",
}: {
  label: ReactNode;
  description?: ReactNode;
  children?: ReactNode;
  /** "start" when the control is taller than one line. */
  align?: "center" | "start";
}) {
  return (
    <div
      className={cn(
        "flex flex-col gap-y-2 px-4 py-3.5 sm:flex-row sm:gap-x-6",
        align === "center" ? "sm:items-center" : "sm:items-start"
      )}
    >
      <div className="flex min-w-0 flex-1 flex-col gap-y-0.5">
        <span className="text-sm font-medium text-main">{label}</span>
        {description && (
          <span className="text-xs leading-relaxed text-control-light">
            {description}
          </span>
        )}
      </div>
      {children && (
        <div className="flex shrink-0 items-center gap-x-2 sm:justify-end">
          {children}
        </div>
      )}
    </div>
  );
}
