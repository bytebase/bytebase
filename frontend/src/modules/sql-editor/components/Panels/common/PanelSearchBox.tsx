import { Search, X } from "lucide-react";
import type { ChangeEvent } from "react";
import { useTranslation } from "react-i18next";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

interface PanelSearchBoxProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
}

/**
 * Search box for the SQL Editor panels, with a leading search icon and a
 * clear button. Width grows to fill the flex container, capped at 18rem.
 */
export function PanelSearchBox({
  value,
  onChange,
  placeholder,
  className,
}: PanelSearchBoxProps) {
  const { t } = useTranslation();
  const handleChange = (event: ChangeEvent<HTMLInputElement>) => {
    onChange(event.target.value);
  };
  return (
    <div className={cn("relative flex-1 max-w-72", className)}>
      <Search className="absolute left-2 top-1/2 -translate-y-1/2 size-3.5 text-control-placeholder pointer-events-none" />
      <Input
        size="sm"
        style={{ paddingInlineStart: "1.75rem", paddingInlineEnd: "1.75rem" }}
        value={value}
        placeholder={placeholder ?? t("common.search")}
        onChange={handleChange}
      />
      {value ? (
        <button
          type="button"
          aria-label={t("common.clear")}
          className="absolute right-1.5 top-1/2 -translate-y-1/2 size-5 inline-flex items-center justify-center rounded-xs text-control-placeholder hover:text-control hover:bg-control-bg"
          onClick={() => onChange("")}
        >
          <X className="size-3.5" />
        </button>
      ) : null}
    </div>
  );
}
