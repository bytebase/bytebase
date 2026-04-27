import { Search, X } from "lucide-react";
import type { ChangeEvent } from "react";
import { useTranslation } from "react-i18next";
import { Input } from "@/react/components/ui/input";
import { cn } from "@/react/lib/utils";

interface PanelSearchBoxProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
}

/**
 * 1:1 React port of `frontend/src/components/v2/Form/SearchBox.vue` for
 * the SQL Editor panels. Naive UI's NInput at `size="small"` renders at
 * 28px tall with a leading prefix icon and a clearable affordance —
 * this wrapper recreates that exactly so the panel toolbar matches the
 * pre-migration look.
 *
 * Width grows to fill the flex container, capped at 18rem (matching
 * the Vue SearchBox's `max-width: 18rem; flex: 1 1 0%`).
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
        className="pl-7 pr-7"
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
