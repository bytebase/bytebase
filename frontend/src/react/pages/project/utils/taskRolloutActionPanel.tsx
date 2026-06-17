import { DateTimePicker } from "@/react/components/ui/date-time-picker";

export const TASK_ROLLOUT_ACTION_SHEET_WIDTH = "standard";

export function ScheduledRunTimeInput({
  className,
  onChange,
  placeholder,
  value,
}: {
  className?: string;
  onChange: (value: number | undefined) => void;
  placeholder: string;
  value: number | undefined;
}) {
  return (
    <DateTimePicker
      className={className}
      minDate={new Date()}
      onChange={(date) => onChange(date ? date.getTime() : undefined)}
      placeholder={placeholder}
      value={value === undefined ? undefined : new Date(value)}
    />
  );
}
