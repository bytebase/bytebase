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
}: {
  value: number;
  items: number[];
  onChange: (value: number) => void;
}) {
  return (
    <Select value={String(value)} onValueChange={(v) => onChange(Number(v))}>
      <SelectTrigger size="sm" className="w-16 justify-center">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {items.map((item) => (
          <SelectItem key={item} value={String(item)}>
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

  const handleSelectDate = (day: Date | undefined) => {
    if (!day) {
      onChange(undefined);
      return;
    }
    onChange(
      dayjs(day).hour(hour).minute(minute).second(0).millisecond(0).toDate()
    );
  };

  const handleTimeChange = (h: number, m: number) => {
    onChange(
      dayjs(value ?? new Date())
        .hour(h)
        .minute(m)
        .second(0)
        .millisecond(0)
        .toDate()
    );
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
          />
          <span className="text-control-light">:</span>
          <TimeSelect
            value={minute}
            items={MINUTES}
            onChange={(m) => handleTimeChange(hour, m)}
          />
        </div>
      </PopoverContent>
    </Popover>
  );
}
