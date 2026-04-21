import { first, last } from "lodash-es";
import { ChevronRight } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { cn } from "@/react/lib/utils";
import { minmax } from "@/utils";

type MaxRowCountSelectProps = {
  readonly value: number;
  readonly onChange: (value: number) => void;
  readonly maximum: number;
  readonly quaternary?: boolean;
  readonly className?: string;
};

export function MaxRowCountSelect({
  value,
  onChange,
  maximum,
  quaternary = false,
  className,
}: MaxRowCountSelectProps) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);

  const rowCountOptions = useMemo(() => {
    const list = [1, 100, 500, 1000, 5000, 10000, 100000].filter(
      (num) => num <= maximum
    );
    if (maximum !== Number.MAX_VALUE && !list.includes(maximum)) {
      list.push(maximum);
    }
    return list;
  }, [maximum]);

  const handlePresetClick = (n: number) => {
    onChange(n);
    setOpen(false);
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const rawValue = Number(e.target.value);
    const clamped = minmax(
      Number.isNaN(rawValue) ? 0 : rawValue,
      first(rowCountOptions) ?? 1,
      last(rowCountOptions) ?? 100000
    );
    onChange(clamped);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            variant={quaternary ? "ghost" : "outline"}
            size="sm"
            className={cn("justify-start", className)}
          />
        }
      >
        {t("sql-editor.result-limit.self")}{" "}
        {t("common.rows.n-rows", { n: value })}
        <ChevronRight className="size-4 ml-1" />
      </PopoverTrigger>
      <PopoverContent side="bottom" align="start" className="p-1 min-w-32">
        <div className="flex flex-col">
          {rowCountOptions.map((n) => (
            <button
              key={n}
              type="button"
              className={cn(
                "px-3 py-1.5 text-sm text-left rounded-xs cursor-pointer",
                "hover:bg-control-bg",
                n === value && "bg-control-bg font-medium"
              )}
              onClick={() => handlePresetClick(n)}
            >
              {t("common.rows.n-rows", { n })}
            </button>
          ))}
          <div className="flex items-center gap-1 px-3 py-1.5 border-t mt-1">
            <input
              type="number"
              className="w-20 border border-control-border rounded-xs px-2 py-0.5 text-sm"
              value={value}
              min={first(rowCountOptions) ?? 1}
              max={last(rowCountOptions) ?? 100000}
              onChange={handleInputChange}
            />
            <span className="text-sm">{t("common.rows.self")}</span>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
