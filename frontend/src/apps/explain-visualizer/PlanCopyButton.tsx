import { Check, Copy } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { writeTextToClipboard } from "@/lib/clipboard";
import { cn } from "@/lib/utils";

/**
 * A copy control that reports the result on itself.
 *
 * The shared `CopyButton` confirms through `useTranslation` and the app store's
 * notifications; this entry initializes neither, so a local button is the whole
 * dependency rather than the app's stores.
 */
interface Props {
  /** What to put on the clipboard. */
  readonly content: string;
  /** Names what is being copied, such as "Copy plan". */
  readonly label: string;
  readonly disabled?: boolean;
}

/** How long the outcome stays on the button before it offers the copy again. */
const CONFIRMATION_MS = 2000;

type CopyState = "idle" | "copied" | "failed";

const MESSAGE: Record<Exclude<CopyState, "idle">, string> = {
  copied: "Copied",
  failed: "Copy failed",
};

export function PlanCopyButton({ content, label, disabled }: Props) {
  const [state, setState] = useState<CopyState>("idle");
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => () => clearTimeout(timerRef.current), []);

  const copy = useCallback(async () => {
    const ok = await writeTextToClipboard(content);
    setState(ok ? "copied" : "failed");
    clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => setState("idle"), CONFIRMATION_MS);
  }, [content]);

  const outcome = state === "idle" ? undefined : MESSAGE[state];

  return (
    <span className="inline-flex items-center">
      <Button
        size="sm"
        appearance="outline"
        disabled={disabled}
        data-testid="plan-copy-button"
        onClick={copy}
        className={cn("bg-background", state === "failed" && "text-error")}
      >
        {state === "copied" ? (
          <Check aria-hidden="true" className="size-3.5 text-success" />
        ) : (
          <Copy aria-hidden="true" className="size-3.5" />
        )}
        {outcome ?? label}
      </Button>
      {/* The button's own text already changed; this announces it to a reader
          whose focus is elsewhere on the page. */}
      <span aria-live="polite" className="sr-only">
        {outcome ?? ""}
      </span>
    </span>
  );
}
