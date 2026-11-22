import { Input as BaseInput } from "@base-ui/react/input";
import { XIcon } from "lucide-react";
import { type KeyboardEvent, useId, useState } from "react";
import { cn } from "@/lib/utils";

interface TagInputProps {
  readonly value: string[];
  readonly onChange: (value: string[]) => void;
  readonly id?: string;
  readonly placeholder?: string;
  readonly disabled?: boolean;
  // Shows the values as chips in a grey box, with no way to change them.
  readonly readOnly?: boolean;
  readonly maxLength?: number;
  readonly "aria-invalid"?: boolean;
  // Names each value's remove button, e.g. `Remove ${value}`.
  readonly removeLabel: (value: string) => string;
}

// A free-text list: Enter or a comma adds what was typed, × removes a value,
// and Backspace in an empty input removes the last one. Duplicates and blank
// values are ignored.
function TagInput({
  value,
  onChange,
  id,
  placeholder,
  disabled,
  readOnly,
  maxLength,
  removeLabel,
  ...aria
}: TagInputProps) {
  const fallbackId = useId();
  const inputId = id ?? fallbackId;
  const [draft, setDraft] = useState("");

  const commit = () => {
    const next = draft.trim();
    setDraft("");
    if (!next || value.includes(next)) return;
    onChange([...value, next]);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter" || event.key === ",") {
      event.preventDefault();
      commit();
      return;
    }
    if (event.key === "Backspace" && draft === "" && value.length > 0) {
      onChange(value.slice(0, -1));
    }
  };

  return (
    <div
      data-slot="tag-input"
      className={cn(
        "flex min-h-9 w-full flex-wrap items-center gap-1.5 rounded-xs border border-control-border px-1.5 py-1",
        readOnly ? "bg-control-bg" : "focus-within:border-accent",
        aria["aria-invalid"] && "border-error",
        disabled && "cursor-not-allowed bg-control-bg opacity-50"
      )}
    >
      {value.map((item) => (
        <span
          key={item}
          className="inline-flex items-center gap-x-1 rounded-xs border border-control-border bg-background px-2 py-0.5 font-mono text-xs text-main"
        >
          {item}
          {!disabled && !readOnly && (
            <button
              type="button"
              aria-label={removeLabel(item)}
              onClick={() => onChange(value.filter((v) => v !== item))}
              className="rounded-xs text-control-light hover:text-main focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-accent"
            >
              <XIcon className="size-3.5" />
            </button>
          )}
        </span>
      ))}
      {!readOnly && (
        <BaseInput
          id={inputId}
          value={draft}
          disabled={disabled}
          maxLength={maxLength}
          placeholder={placeholder}
          autoComplete="off"
          onChange={(event) => setDraft(event.target.value)}
          onKeyDown={handleKeyDown}
          onBlur={commit}
          className={cn(
            "h-6 min-w-24 flex-1 border-0 bg-transparent px-1 text-sm text-main outline-hidden",
            "placeholder:text-control-placeholder disabled:cursor-not-allowed"
          )}
          {...aria}
        />
      )}
    </div>
  );
}

export type { TagInputProps };
export { TagInput };
