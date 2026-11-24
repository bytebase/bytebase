import { ChevronLeft, ChevronRight } from "lucide-react";
import type { ComponentProps } from "react";
import { DayPicker } from "react-day-picker";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type CalendarSize = "md" | "lg";

export type CalendarProps = ComponentProps<typeof DayPicker> & {
  /** Day cell size: `md` is 32px, `lg` is 40px for touch. Defaults to `md`. */
  size?: CalendarSize;
};

const cellClass: Record<CalendarSize, string> = {
  md: "size-8",
  lg: "size-10",
};

const weekdayClass: Record<CalendarSize, string> = {
  md: "w-8",
  lg: "w-10",
};

const selectedDayClass =
  "[&>button]:bg-accent [&>button]:text-accent-text [&>button]:hover:bg-accent-hover [&>button]:hover:text-accent-text";

const isSameDay = (a: Date, b: Date) =>
  a.getFullYear() === b.getFullYear() &&
  a.getMonth() === b.getMonth() &&
  a.getDate() === b.getDate();

/**
 * Calendar — shadcn-style wrapper around react-day-picker, themed with app
 * tokens (no react-day-picker default CSS).
 */
function Calendar({
  className,
  classNames,
  showOutsideDays = true,
  size = "md",
  ...props
}: CalendarProps) {
  const isRange = props.mode === "range";
  // A one-day range has no band to join, so its cell keeps a plain background.
  const spansDays =
    props.mode === "range" &&
    !!props.selected?.from &&
    !!props.selected.to &&
    !isSameDay(props.selected.from, props.selected.to);

  return (
    <DayPicker
      showOutsideDays={showOutsideDays}
      className={cn("p-0", className)}
      classNames={{
        // `relative` keeps the absolutely placed `nav` on the caption row. The
        // caption stays unpositioned so it cannot cover the `nav` buttons.
        months: "relative flex flex-col gap-4",
        month: "flex flex-col gap-4",
        month_caption: "flex h-7 items-center justify-center",
        caption_label: "text-sm font-medium text-control",
        nav: "flex items-center justify-between absolute inset-x-0 top-0",
        button_previous: cn(
          buttonVariants({ appearance: "secondary" }),
          "size-7 p-0 text-control-light"
        ),
        button_next: cn(
          buttonVariants({ appearance: "secondary" }),
          "size-7 p-0 text-control-light"
        ),
        month_grid: "w-full border-collapse",
        weekdays: "flex",
        weekday: cn(
          "text-control-light font-normal text-xs",
          weekdayClass[size]
        ),
        week: "flex w-full mt-1",
        day: cn("relative p-0 text-center text-sm", cellClass[size]),
        day_button: cn(
          buttonVariants({ appearance: "secondary" }),
          "p-0 font-normal text-control tabular-nums",
          cellClass[size]
        ),
        // Today's colors yield to a selection, which colors the same button.
        today:
          "[&>button]:font-semibold [&:not([data-selected])>button]:bg-control-bg [&:not([data-selected])>button]:text-accent",
        // In range mode every day of the range is `selected`; only its two
        // ends are filled, and the days between carry the band.
        selected: isRange ? "" : selectedDayClass,
        range_start: cn(
          selectedDayClass,
          spansDays &&
            "bg-linear-to-r from-transparent from-50% to-accent/10 to-50%"
        ),
        range_end: cn(
          selectedDayClass,
          spansDays &&
            "bg-linear-to-l from-transparent from-50% to-accent/10 to-50%"
        ),
        range_middle: "bg-accent/10",
        outside: "[&>button]:text-control-light [&>button]:opacity-50",
        disabled:
          "[&>button]:text-control-light [&>button]:opacity-40 [&>button]:cursor-not-allowed",
        hidden: "invisible",
        ...classNames,
      }}
      components={{
        Chevron: ({ orientation, className: chevronClassName, ...rest }) =>
          orientation === "left" ? (
            <ChevronLeft className={cn("size-4", chevronClassName)} {...rest} />
          ) : (
            <ChevronRight
              className={cn("size-4", chevronClassName)}
              {...rest}
            />
          ),
      }}
      {...props}
    />
  );
}

export { Calendar };
