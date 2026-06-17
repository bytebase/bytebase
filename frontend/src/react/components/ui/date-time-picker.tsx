import dayjs from "dayjs";
import { Calendar as CalendarIcon } from "lucide-react";
import { useState } from "react";
import { Calendar } from "@/react/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/react/components/ui/select";
import { cn } from "@/react/lib/utils";

const HOURS = Array.from({ length: 24 }, (_, i) => i);
const MINUTES = Array.from({ length: 60 }, (_, i) => i);
const pad = (n: number) => String(n).padStart(2, "0");

function TimeSelect({
  value,
  items,
  onChange,
  isDisabled,
}: {
  value: number;
  items: number[];
  onChange: (value: number) => void;
  isDisabled?: (value: number) => boolean;
}) {
  return (
    <Select value={String(value)} onValueChange={(v) => onChange(Number(v))}>
      <SelectTrigger size="sm" className="w-16 justify-center">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {items.map((item) => (
          <SelectItem
            key={item}
            value={String(item)}
            disabled={isDisabled?.(item)}
          >
            {pad(item)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

/**
 * DateTimePicker — a styled trigger that opens a Popover containing an
 * app-themed calendar plus hour/minute selects. Replaces native
 * `<input type="datetime-local">` for consistent look and a full-width trigger.
 */
export function DateTimePicker({
  value,
  onChange,
  placeholder,
  disabled,
  minDate,
  maxDate,
  className,
  displayFormat = "YYYY-MM-DD HH:mm",
}: {
  value: Date | undefined;
  onChange: (value: Date | undefined) => void;
  placeholder?: string;
  disabled?: boolean;
  minDate?: Date;
  maxDate?: Date;
  className?: string;
  displayFormat?: string;
}) {
  const [open, setOpen] = useState(false);

  const hour = value ? value.getHours() : 0;
  const minute = value ? value.getMinutes() : 0;

  // The picker is minute-granular, so floor the bounds to the minute and clamp
  // every emitted value into [minTs, maxTs]. The calendar's day matchers only
  // disable whole days, so without this the time selects could emit a value
  // outside the bound on the boundary day.
  const minTs = minDate
    ? dayjs(minDate).second(0).millisecond(0).valueOf()
    : undefined;
  const maxTs = maxDate
    ? dayjs(maxDate).second(0).millisecond(0).valueOf()
    : undefined;

  const clamp = (date: Date) => {
    const ts = date.getTime();
    if (minTs !== undefined && ts < minTs) return new Date(minTs);
    if (maxTs !== undefined && ts > maxTs) return new Date(maxTs);
    return date;
  };

  const handleSelectDate = (day: Date | undefined) => {
    if (!day) {
      onChange(undefined);
      return;
    }
    onChange(
      clamp(
        dayjs(day).hour(hour).minute(minute).second(0).millisecond(0).toDate()
      )
    );
  };

  const handleTimeChange = (h: number, m: number) => {
    onChange(
      clamp(
        dayjs(value ?? new Date())
          .hour(h)
          .minute(m)
          .second(0)
          .millisecond(0)
          .toDate()
      )
    );
  };

  // Disable hour/minute options that fall outside the bounds on the selected
  // day, so the UI reflects the same range the value is clamped to.
  const isHourDisabled = (h: number) => {
    if (!value) return false;
    const latest = dayjs(value).hour(h).minute(59).second(0).millisecond(0);
    const earliest = dayjs(value).hour(h).minute(0).second(0).millisecond(0);
    if (minTs !== undefined && latest.valueOf() < minTs) return true;
    if (maxTs !== undefined && earliest.valueOf() > maxTs) return true;
    return false;
  };
  const isMinuteDisabled = (m: number) => {
    if (!value) return false;
    const ts = dayjs(value)
      .hour(hour)
      .minute(m)
      .second(0)
      .millisecond(0)
      .valueOf();
    if (minTs !== undefined && ts < minTs) return true;
    if (maxTs !== undefined && ts > maxTs) return true;
    return false;
  };

  const disabledMatchers = [
    ...(minDate ? [{ before: minDate }] : []),
    ...(maxDate ? [{ after: maxDate }] : []),
  ];

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        disabled={disabled}
        className={cn(
          "inline-flex h-9 w-64 max-w-full items-center justify-between gap-2 rounded-xs border border-control-border bg-background px-3 text-sm text-control",
          "cursor-pointer hover:bg-control-bg focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent",
          "disabled:cursor-not-allowed disabled:opacity-50",
          className
        )}
      >
        <span className={cn("truncate", !value && "text-control-placeholder")}>
          {value ? dayjs(value).format(displayFormat) : (placeholder ?? "")}
        </span>
        <CalendarIcon className="size-4 shrink-0 text-control-light" />
      </PopoverTrigger>
      <PopoverContent align="start" className="w-auto p-3">
        <Calendar
          mode="single"
          autoFocus
          selected={value}
          defaultMonth={value ?? minDate}
          disabled={disabledMatchers.length ? disabledMatchers : undefined}
          onSelect={handleSelectDate}
        />
        <div className="mt-2 flex items-center justify-center gap-1 border-t border-control-border pt-3">
          <TimeSelect
            value={hour}
            items={HOURS}
            onChange={(h) => handleTimeChange(h, minute)}
            isDisabled={isHourDisabled}
          />
          <span className="text-control-light">:</span>
          <TimeSelect
            value={minute}
            items={MINUTES}
            onChange={(m) => handleTimeChange(hour, m)}
            isDisabled={isMinuteDisabled}
          />
        </div>
      </PopoverContent>
    </Popover>
  );
}
