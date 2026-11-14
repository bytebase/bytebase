import type { AriaAttributes, ReactNode } from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";

export type NeonOption = { value: string; label: ReactNode };

// A single-value Select over string options. A value outside the options
// shows the placeholder, which is how a required choice reads as unmade.
export function NeonOptionSelect({
  id,
  value,
  options,
  placeholder,
  size = "md",
  invalid = false,
  disabled = false,
  className,
  onChange,
  ...aria
}: AriaAttributes & {
  id?: string;
  value: string;
  options: readonly NeonOption[];
  placeholder?: string;
  size?: "sm" | "md";
  invalid?: boolean;
  disabled?: boolean;
  className?: string;
  onChange: (value: string) => void;
}) {
  const selected = options.find((option) => option.value === value);
  return (
    <Select
      value={selected ? value : null}
      disabled={disabled}
      onValueChange={(next) => {
        if (typeof next === "string") onChange(next);
      }}
    >
      <SelectTrigger
        {...aria}
        id={id}
        size={size}
        aria-invalid={invalid || undefined}
        className={cn("w-full min-w-0", invalid && "border-error", className)}
      >
        <SelectValue>
          {selected ? (
            <span className="truncate text-main">{selected.label}</span>
          ) : (
            <span className="truncate text-control-placeholder">
              {placeholder}
            </span>
          )}
        </SelectValue>
      </SelectTrigger>
      <SelectContent>
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
