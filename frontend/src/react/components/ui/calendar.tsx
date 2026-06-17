import { ChevronLeft, ChevronRight } from "lucide-react";
import type { ComponentProps } from "react";
import { DayPicker } from "react-day-picker";
import { buttonVariants } from "@/react/components/ui/button";
import { cn } from "@/react/lib/utils";

export type CalendarProps = ComponentProps<typeof DayPicker>;

/**
 * Calendar — shadcn-style wrapper around react-day-picker, themed with app
 * tokens (no react-day-picker default CSS).
 */
function Calendar({
  className,
  classNames,
  showOutsideDays = true,
  ...props
}: CalendarProps) {
  return (
    <DayPicker
      showOutsideDays={showOutsideDays}
      className={cn("p-0", className)}
      classNames={{
        months: "flex flex-col gap-4",
        month: "flex flex-col gap-4",
        month_caption: "flex h-7 items-center justify-center relative",
        caption_label: "text-sm font-medium text-control",
        nav: "flex items-center justify-between absolute inset-x-0 top-0",
        button_previous: cn(
          buttonVariants({ variant: "ghost" }),
          "size-7 p-0 text-control-light"
        ),
        button_next: cn(
          buttonVariants({ variant: "ghost" }),
          "size-7 p-0 text-control-light"
        ),
        month_grid: "w-full border-collapse",
        weekdays: "flex",
        weekday: "text-control-light w-8 font-normal text-[0.8rem]",
        week: "flex w-full mt-1",
        day: "relative size-8 p-0 text-center text-sm",
        day_button: cn(
          buttonVariants({ variant: "ghost" }),
          "size-8 p-0 font-normal text-control"
        ),
        today:
          "[&>button]:bg-control-bg [&>button]:font-semibold [&>button]:text-accent",
        selected:
          "[&>button]:bg-accent [&>button]:text-accent-text [&>button]:hover:bg-accent-hover [&>button]:hover:text-accent-text",
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
