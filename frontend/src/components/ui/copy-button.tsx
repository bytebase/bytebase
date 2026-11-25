import { Check, Copy } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button, type ButtonProps } from "@/components/ui/button";
import type { LayerFamily } from "@/components/ui/layer";
import { Tooltip } from "@/components/ui/tooltip";
import { writeTextToClipboard } from "@/lib/clipboard";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";

interface CopyButtonProps {
  // The text to copy. Pass a function to defer resolution until click.
  readonly content: string | (() => string);
  // Accessible name. Name what is copied when several copy buttons share a view.
  readonly label?: string;
  readonly size?: ButtonProps["size"];
  readonly variant?: ButtonProps["variant"];
  readonly appearance?: ButtonProps["appearance"];
  readonly disabled?: boolean;
  readonly className?: string;
  readonly tooltipLayer?: LayerFamily;
  // Visible text beside the icon. With it the button needs no tooltip; without
  // it the button is icon-only.
  readonly children?: React.ReactNode;
}

export function CopyButton({
  content,
  label,
  size = "xs",
  variant = "default",
  appearance = "secondary",
  disabled,
  className,
  tooltipLayer,
  children,
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

  const hasText = children !== undefined;
  const button = (
    <Button
      type="button"
      variant={variant}
      appearance={appearance}
      size={size}
      disabled={disabled}
      onClick={handleCopy}
      aria-label={hasText ? label : (label ?? t("common.copy"))}
      className={cn(!hasText && "px-1", className)}
    >
      {copied ? (
        // On a solid button the check takes the button's own text color.
        <Check
          className={cn("size-3.5", appearance !== "solid" && "text-success")}
        />
      ) : (
        <Copy className="size-3.5" />
      )}
      {children}
    </Button>
  );
  if (hasText) return button;
  return (
    <Tooltip content={t("common.copy")} layer={tooltipLayer}>
      {button}
    </Tooltip>
  );
}
