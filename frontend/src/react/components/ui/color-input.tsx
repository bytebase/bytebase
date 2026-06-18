import { throttle } from "lodash-es";
import { useEffect, useMemo, useRef, useState } from "react";
import { Input } from "@/react/components/ui/input";
import { cn } from "@/react/lib/utils";

const HEX_RE = /^#?([0-9a-fA-F]{6})$/;

/** Normalize free text to `#rrggbb`, or null when it isn't a 6-digit hex. */
function normalizeHex(raw: string): string | null {
  const match = HEX_RE.exec(raw.trim());
  return match ? `#${match[1].toLowerCase()}` : null;
}

interface ColorInputProps {
  /** Current color as `#rrggbb`. */
  value: string;
  /** Called with a normalized `#rrggbb` whenever a valid color is entered. */
  onChange: (value: string) => void;
  id?: string;
  disabled?: boolean;
  ariaLabel?: string;
}

/**
 * A color control pairing the native swatch picker with an editable hex text
 * field, so a color can be picked visually OR typed as `#081C56`. The text
 * field accepts free input while typing and commits only a valid 6-digit hex
 * (reverting to the last good value on blur); the swatch always commits.
 */
export function ColorInput({
  value,
  onChange,
  id,
  disabled,
  ariaLabel,
}: Readonly<ColorInputProps>) {
  const [draft, setDraft] = useState(value);

  // The native swatch fires `input` events continuously while dragging. Throttle
  // them (via a ref so the throttled fn isn't recreated each render) so expensive
  // downstream work — re-deriving the theme + re-rendering the preview — runs at
  // a steady rate instead of on every tick. Trailing keeps the final color.
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const emitSwatch = useMemo(
    () => throttle((next: string) => onChangeRef.current(next), 40),
    []
  );
  useEffect(() => () => emitSwatch.cancel(), [emitSwatch]);

  // Adopt external changes (swatch, preset switch, revert) but keep the user's
  // raw text while they are typing an equivalent value.
  useEffect(() => {
    setDraft((prev) => (normalizeHex(prev) === value ? prev : value));
  }, [value]);

  return (
    <>
      <input
        id={id}
        type="color"
        className={cn(
          // Strip the native swatch chrome (`appearance-none` + `p-0` + the
          // ::*-color-swatch pseudo-elements) so only our single outer border
          // shows — the browser otherwise draws its own inner swatch border.
          "size-8 shrink-0 appearance-none rounded-xs border border-control-border bg-transparent p-0",
          "[&::-webkit-color-swatch-wrapper]:p-0 [&::-webkit-color-swatch]:rounded-xs [&::-webkit-color-swatch]:border-0 [&::-moz-color-swatch]:rounded-xs [&::-moz-color-swatch]:border-0",
          disabled ? "cursor-not-allowed opacity-50" : "cursor-pointer"
        )}
        value={value}
        disabled={disabled}
        aria-label={ariaLabel}
        onChange={(e) => emitSwatch(e.target.value)}
      />
      <Input
        size="sm"
        className="h-8 w-28 font-mono"
        value={draft}
        disabled={disabled}
        aria-label={ariaLabel}
        spellCheck={false}
        autoComplete="off"
        onChange={(e) => {
          const raw = e.target.value;
          setDraft(raw);
          const normalized = normalizeHex(raw);
          if (normalized) onChange(normalized);
        }}
        onBlur={() => {
          if (!normalizeHex(draft)) setDraft(value);
        }}
      />
    </>
  );
}
