import { useCallback, useRef } from "react";
import { cn } from "@/react/lib/utils";

interface OtpInputProps {
  value: string[];
  onChange: (value: string[]) => void;
  onFinish?: (value: string[]) => void;
  length?: number;
  className?: string;
}

export function OtpInput({
  value,
  onChange,
  onFinish,
  length = 6,
  className,
}: OtpInputProps) {
  const inputRefs = useRef<(HTMLInputElement | null)[]>([]);

  const handleChange = useCallback(
    (index: number, raw: string) => {
      const digits = raw.replace(/\D/g, "");
      const next = [...value];
      while (next.length < length) next.push("");
      if (!digits) {
        next[index] = "";
        onChange(next);
        return;
      }
      // Accept multi-digit input here so 2FA autofill extensions that set
      // `input.value = "123456"` on the first cell get distributed across
      // cells instead of rejected.
      for (let i = 0; i < digits.length && index + i < length; i++) {
        next[index + i] = digits[i];
      }
      onChange(next);
      const focusIndex = Math.min(index + digits.length, length - 1);
      inputRefs.current[focusIndex]?.focus();
      if (next.filter((v) => v).length === length) {
        onFinish?.(next);
      }
    },
    [value, onChange, onFinish, length]
  );

  const handleKeyDown = useCallback(
    (index: number, e: React.KeyboardEvent) => {
      if (e.key === "Backspace" && !value[index] && index > 0) {
        inputRefs.current[index - 1]?.focus();
      }
    },
    [value]
  );

  const handlePaste = useCallback(
    (e: React.ClipboardEvent) => {
      e.preventDefault();
      const pasted = e.clipboardData
        .getData("text")
        .replace(/\D/g, "")
        .slice(0, length);
      if (!pasted) return;
      const next = [...value];
      while (next.length < length) next.push("");
      for (let i = 0; i < pasted.length; i++) {
        next[i] = pasted[i];
      }
      onChange(next);
      const focusIndex = Math.min(pasted.length, length - 1);
      inputRefs.current[focusIndex]?.focus();
      if (next.filter((v) => v).length === length) {
        onFinish?.(next);
      }
    },
    [value, onChange, onFinish, length]
  );

  return (
    <div className={cn("flex gap-2", className)}>
      {Array.from({ length }, (_, i) => (
        <input
          key={i}
          ref={(el) => {
            inputRefs.current[i] = el;
          }}
          type="text"
          inputMode="numeric"
          name={i === 0 ? "otp" : undefined}
          maxLength={i === 0 ? length : 1}
          value={value[i] || ""}
          onChange={(e) => handleChange(i, e.target.value)}
          onKeyDown={(e) => handleKeyDown(i, e)}
          onPaste={handlePaste}
          className="flex-1 min-w-0 h-12 text-center text-lg font-mono border border-control-border rounded-xs focus:outline-none focus:ring-2 focus:ring-accent focus:border-accent"
          autoComplete="one-time-code"
        />
      ))}
    </div>
  );
}
