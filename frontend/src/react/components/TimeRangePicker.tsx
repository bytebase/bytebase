import dayjs from "dayjs";
import { ArrowRight, Calendar } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { SearchParams } from "./AdvancedSearch";
import { Button } from "./ui/button";
import { DateTimePicker } from "./ui/date-time-picker";
import { Popover, PopoverContent, PopoverTrigger } from "./ui/popover";

interface TimeRangePickerProps {
  params: SearchParams;
  onParamsChange: (p: SearchParams) => void;
}

const DISPLAY_FORMAT = "YYYY-MM-DD HH:mm";

export function TimeRangePicker({
  params,
  onParamsChange,
}: TimeRangePickerProps) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);

  const createdScope = params.scopes.find((s) => s.id === "created");
  const [fromTs, toTs] = useMemo(() => {
    if (!createdScope) return [undefined, undefined];
    const parts = createdScope.value.split(",");
    if (parts.length !== 2) return [undefined, undefined];
    return [parseInt(parts[0], 10), parseInt(parts[1], 10)];
  }, [createdScope]);

  const hasRange = fromTs !== undefined && toTs !== undefined;

  // Local draft state for the pickers inside the dropdown.
  const [draftFrom, setDraftFrom] = useState<Date | undefined>(undefined);
  const [draftTo, setDraftTo] = useState<Date | undefined>(undefined);

  // Sync drafts when the external range changes.
  useEffect(() => {
    setDraftFrom(fromTs ? new Date(fromTs) : undefined);
    setDraftTo(toTs ? new Date(toTs) : undefined);
  }, [fromTs, toTs]);

  const displayFrom = fromTs ? dayjs(fromTs).format(DISPLAY_FORMAT) : "";
  const displayTo = toTs ? dayjs(toTs).format(DISPLAY_FORMAT) : "";

  const applyRange = useCallback(() => {
    const fromVal = draftFrom ? draftFrom.getTime() : undefined;
    const toVal = draftTo ? draftTo.getTime() : undefined;
    const scopes = params.scopes.filter((s) => s.id !== "created");
    if (fromVal !== undefined && toVal !== undefined) {
      scopes.push({
        id: "created",
        value: `${fromVal},${toVal}`,
        readonly: true,
      });
    }
    onParamsChange({ ...params, scopes });
    setOpen(false);
  }, [draftFrom, draftTo, params, onParamsChange]);

  const clearRange = useCallback(() => {
    const scopes = params.scopes.filter((s) => s.id !== "created");
    onParamsChange({ ...params, scopes });
    setDraftFrom(undefined);
    setDraftTo(undefined);
    setOpen(false);
  }, [params, onParamsChange]);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger className="h-9 flex items-center gap-x-2 border border-control-border rounded-xs px-3 text-sm hover:bg-control-bg whitespace-nowrap shrink-0">
        {displayFrom && displayTo ? (
          <>
            <span>{displayFrom}</span>
            <ArrowRight className="size-3.5 text-control-light shrink-0" />
            <span>{displayTo}</span>
          </>
        ) : (
          <span className="text-control-placeholder">{t("common.select")}</span>
        )}
        <Calendar className="size-4 text-control-light ml-1 shrink-0" />
      </PopoverTrigger>
      <PopoverContent
        align="end"
        className="flex flex-col gap-y-2 min-w-[300px]"
      >
        <RangeEndPicker
          label={t("common.from")}
          value={draftFrom}
          maxDate={draftTo}
          onChange={setDraftFrom}
        />
        <RangeEndPicker
          label={t("common.to")}
          value={draftTo}
          minDate={draftFrom}
          onChange={setDraftTo}
        />
        <div className="flex items-center gap-x-2 mt-1">
          <Button
            size="sm"
            onClick={applyRange}
            disabled={!draftFrom || !draftTo}
          >
            {t("common.confirm")}
          </Button>
          {hasRange && (
            <Button variant="ghost" size="sm" onClick={clearRange}>
              {t("common.clear")}
            </Button>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}

function RangeEndPicker({
  label,
  value,
  minDate,
  maxDate,
  onChange,
}: {
  label: string;
  value: Date | undefined;
  minDate?: Date;
  maxDate?: Date;
  onChange: (value: Date | undefined) => void;
}) {
  return (
    <div className="flex items-center gap-x-2">
      <label className="text-sm text-control-light whitespace-nowrap w-10">
        {label}
      </label>
      <DateTimePicker
        className="flex-1 w-full min-w-0"
        value={value}
        minDate={minDate}
        maxDate={maxDate}
        onChange={onChange}
      />
    </div>
  );
}
