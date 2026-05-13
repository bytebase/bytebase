import dayjs from "dayjs";
import { ArrowRight, Calendar } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { SearchParams } from "./AdvancedSearch";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { LAYER_SURFACE_CLASS } from "./ui/layer";

interface TimeRangePickerProps {
  params: SearchParams;
  onParamsChange: (p: SearchParams) => void;
}

export function TimeRangePicker({
  params,
  onParamsChange,
}: TimeRangePickerProps) {
  const { t } = useTranslation();
  const [showPicker, setShowPicker] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const createdScope = params.scopes.find((s) => s.id === "created");
  const [fromTs, toTs] = useMemo(() => {
    if (!createdScope) return [undefined, undefined];
    const parts = createdScope.value.split(",");
    if (parts.length !== 2) return [undefined, undefined];
    return [parseInt(parts[0], 10), parseInt(parts[1], 10)];
  }, [createdScope]);

  const hasRange = fromTs !== undefined && toTs !== undefined;

  // Local draft state for the inputs inside the dropdown.
  const [draftFrom, setDraftFrom] = useState("");
  const [draftTo, setDraftTo] = useState("");

  // Sync drafts when the dropdown opens or the external range changes.
  useEffect(() => {
    if (fromTs) {
      setDraftFrom(dayjs(fromTs).format("YYYY-MM-DDTHH:mm:ss"));
    } else {
      setDraftFrom("");
    }
    if (toTs) {
      setDraftTo(dayjs(toTs).format("YYYY-MM-DDTHH:mm:ss"));
    } else {
      setDraftTo("");
    }
  }, [fromTs, toTs]);

  const displayFrom = fromTs ? dayjs(fromTs).format("YYYY-MM-DD HH:mm:ss") : "";
  const displayTo = toTs ? dayjs(toTs).format("YYYY-MM-DD HH:mm:ss") : "";

  const applyRange = useCallback(() => {
    const fromVal = draftFrom ? dayjs(draftFrom).valueOf() : undefined;
    const toVal = draftTo ? dayjs(draftTo).valueOf() : undefined;
    const scopes = params.scopes.filter((s) => s.id !== "created");
    if (fromVal !== undefined && toVal !== undefined) {
      scopes.push({
        id: "created",
        value: `${fromVal},${toVal}`,
        readonly: true,
      });
    }
    onParamsChange({ ...params, scopes });
    setShowPicker(false);
  }, [draftFrom, draftTo, params, onParamsChange]);

  const clearRange = useCallback(() => {
    const scopes = params.scopes.filter((s) => s.id !== "created");
    onParamsChange({ ...params, scopes });
    setDraftFrom("");
    setDraftTo("");
    setShowPicker(false);
  }, [params, onParamsChange]);

  // Close on click outside.
  useEffect(() => {
    if (!showPicker) return;
    const handler = (e: MouseEvent) => {
      if (
        containerRef.current &&
        !containerRef.current.contains(e.target as Node) &&
        !containerRef.current.contains(document.activeElement)
      ) {
        setShowPicker(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [showPicker]);

  return (
    <div ref={containerRef} className="relative shrink-0">
      <button
        type="button"
        className="h-9 flex items-center gap-x-2 border border-control-border rounded-xs px-3 text-sm hover:bg-control-bg whitespace-nowrap"
        onClick={() => setShowPicker(!showPicker)}
      >
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
      </button>
      {showPicker && (
        <div
          className={`absolute right-0 top-[42px] bg-background border border-control-border rounded-sm shadow-lg p-3 flex flex-col gap-y-2 min-w-[300px] ${LAYER_SURFACE_CLASS}`}
        >
          <div className="flex items-center gap-x-2">
            <label className="text-sm text-control-light whitespace-nowrap w-10">
              {t("common.from")}
            </label>
            <Input
              type="datetime-local"
              step="1"
              className="flex-1 accent-accent"
              value={draftFrom}
              onChange={(e) => setDraftFrom(e.target.value)}
            />
          </div>
          <div className="flex items-center gap-x-2">
            <label className="text-sm text-control-light whitespace-nowrap w-10">
              {t("common.to")}
            </label>
            <Input
              type="datetime-local"
              step="1"
              className="flex-1 accent-accent"
              value={draftTo}
              onChange={(e) => setDraftTo(e.target.value)}
            />
          </div>
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
        </div>
      )}
    </div>
  );
}
