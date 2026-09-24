import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";

interface SwitchRowProps {
  title: string;
  description?: ReactNode;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  disabled?: boolean;
  // Shows the state as text instead of a switch, for a setting made elsewhere.
  readOnly?: boolean;
  className?: string;
}

// A setting whose text takes the row and whose control trails it.
export function SwitchRow({
  title,
  description,
  checked,
  onCheckedChange,
  disabled = false,
  readOnly = false,
  className,
}: SwitchRowProps) {
  const { t } = useTranslation();
  return (
    <div className={cn("flex items-center justify-between gap-x-4", className)}>
      <div className="flex min-w-0 flex-col">
        <div className="text-sm font-medium text-main">{title}</div>
        {description !== undefined && (
          <div className="text-sm text-control-light">{description}</div>
        )}
      </div>
      {readOnly ? (
        <span
          className={cn(
            "inline-flex shrink-0 items-center gap-x-1.5 text-xs",
            checked ? "text-main" : "text-control-light"
          )}
        >
          <span
            className={cn(
              "size-1.5 rounded-full",
              checked ? "bg-accent" : "bg-control-placeholder"
            )}
          />
          {checked
            ? t("sql-review.standard-rules.on")
            : t("sql-review.standard-rules.off")}
        </span>
      ) : (
        <Switch
          className="shrink-0"
          checked={checked}
          onCheckedChange={onCheckedChange}
          disabled={disabled}
          aria-label={title}
        />
      )}
    </div>
  );
}
