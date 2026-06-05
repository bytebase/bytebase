import { Input } from "@/react/components/ui/input";
import { cn } from "@/react/lib/utils";

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
    <Input
      className={cn("w-64 max-w-full", className)}
      onChange={(event) =>
        onChange(parseDatetimeLocalValue(event.target.value))
      }
      placeholder={placeholder}
      type="datetime-local"
      value={formatDatetimeLocalValue(value)}
    />
  );
}

export function formatDatetimeLocalValue(value?: number) {
  if (value === undefined) {
    return "";
  }
  const date = new Date(value);
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

export function parseDatetimeLocalValue(value: string) {
  return value ? new Date(value).getTime() : undefined;
}
