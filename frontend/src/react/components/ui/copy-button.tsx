import { Check, Copy } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button, type ButtonProps } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { writeTextToClipboard } from "@/react/lib/clipboard";
import { cn } from "@/react/lib/utils";
import { useAppStore } from "@/react/stores/app";

interface CopyButtonProps {
  // The text to copy. Pass a function to defer resolution until click.
  readonly content: string | (() => string);
  readonly size?: ButtonProps["size"];
  readonly variant?: ButtonProps["variant"];
  readonly disabled?: boolean;
  readonly className?: string;
}

export function CopyButton({
  content,
  size = "xs",
  variant = "ghost",
  disabled,
  className,
}: CopyButtonProps) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => () => clearTimeout(timerRef.current), []);

  const handleCopy = useCallback(async () => {
    const text = typeof content === "function" ? content() : content;
    if (!text) return;
    const ok = await writeTextToClipboard(text);
    useAppStore.getState().notify({
      module: "bytebase",
      style: ok ? "SUCCESS" : "CRITICAL",
      title: ok ? t("common.copied") : t("common.copy-failed"),
    });
    if (!ok) return;
    setCopied(true);
    clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => setCopied(false), 2000);
  }, [content, t]);

  return (
    <Tooltip content={t("common.copy")}>
      <Button
        type="button"
        variant={variant}
        size={size}
        disabled={disabled}
        onClick={handleCopy}
        aria-label={t("common.copy")}
        className={cn("px-1", className)}
      >
        {copied ? (
          <Check className="size-3.5 text-success" />
        ) : (
          <Copy className="size-3.5" />
        )}
      </Button>
    </Tooltip>
  );
}
