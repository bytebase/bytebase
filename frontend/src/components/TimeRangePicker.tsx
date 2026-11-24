import dayjs from "dayjs";
import type { TFunction } from "i18next";
import {
  ArrowRight,
  Calendar as CalendarIcon,
  Check,
  CircleAlert,
  X,
} from "lucide-react";
import {
  type ComponentProps,
  type SubmitEvent,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import type { SearchParams } from "./AdvancedSearch";
import {
  END_OF_DAY,
  formatDay,
  formatRangeScope,
  isWholeDays,
  matchQuickRange,
  parseRangeScope,
  parseTimeOfDay,
  QUICK_RANGE_IDS,
  type QuickRangeId,
  quickRange,
  rangeEnd,
  rangeStart,
  START_OF_DAY,
  type TimeRange,
  timeOfDay,
} from "./time-range";
import { Button } from "./ui/button";
import { Calendar } from "./ui/calendar";
import { Input } from "./ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "./ui/popover";
import { Separator } from "./ui/separator";

interface TimeRangePickerProps {
  params: SearchParams;
  onParamsChange: (p: SearchParams) => void;
  /** Names the time the range filters. Shown on the trigger. */
  label: string;
}

type RangeEnd = "from" | "to";
const RANGE_ENDS: readonly RangeEnd[] = ["from", "to"];

interface Draft {
  fromDay?: Date;
  toDay?: Date;
  fromTime: string;
  toTime: string;
  /** The end the next calendar click sets. */
  activeEnd: RangeEnd;
  /** Compact layout only: whether the time fields are shown. */
  showTime: boolean;
}

// Not a range: picking it removes the filter.
const ALL_TIME = "all-time";
type QuickOption = QuickRangeId | typeof ALL_TIME;
const QUICK_OPTIONS: readonly QuickOption[] = [...QUICK_RANGE_IDS, ALL_TIME];

const RANGE_SCOPE_ID = "created";
const COMPACT_MEDIA_QUERY = "(max-width: 639px)";
// Extends a control's tap area to 44px without changing how it looks.
const TAP_AREA_MD =
  "relative before:absolute before:inset-x-0 before:-inset-y-1";
const TAP_AREA_SM =
  "relative before:absolute before:inset-x-0 before:-inset-y-2";

const toDraft = (range: TimeRange | undefined): Draft => ({
  fromDay: range ? dayjs(range.from).startOf("day").toDate() : undefined,
  toDay: range ? dayjs(range.to).startOf("day").toDate() : undefined,
  fromTime: range ? timeOfDay(range.from) : START_OF_DAY,
  toTime: range ? timeOfDay(range.to) : END_OF_DAY,
  activeEnd: "from",
  showTime: !!range && !isWholeDays(range),
});

const quickRangeLabel = (t: TFunction, id: QuickRangeId): string => {
  switch (id) {
    case "today":
      return t("time-range.today");
    case "last-7-days":
      return t("time-range.last-7-days");
    case "last-30-days":
      return t("time-range.last-30-days");
    case "last-3-months":
      return t("time-range.last-3-months");
    case "last-6-months":
      return t("time-range.last-6-months");
    case "last-12-months":
      return t("time-range.last-12-months");
  }
};

export function TimeRangePicker({
  params,
  onParamsChange,
  label,
}: TimeRangePickerProps) {
  const { t } = useTranslation();
  const compact = useMediaQuery(COMPACT_MEDIA_QUERY);
  const anchorRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState<Draft>(() => toDraft(undefined));
  const [invalidTime, setInvalidTime] = useState({ from: false, to: false });

  const scopeValue = params.scopes.find((s) => s.id === RANGE_SCOPE_ID)?.value;
  const applied = useMemo(() => parseRangeScope(scopeValue), [scopeValue]);

  const draftRange: TimeRange | undefined =
    draft.fromDay && draft.toDay
      ? {
          from: rangeStart(draft.fromDay, draft.fromTime),
          to: rangeEnd(draft.toDay, draft.toTime),
        }
      : undefined;
  const error =
    invalidTime.from || invalidTime.to
      ? t("time-range.invalid-time")
      : draftRange && draftRange.to < draftRange.from
        ? t("time-range.to-before-from")
        : "";

  const handleOpenChange = (next: boolean) => {
    if (next) {
      setDraft(toDraft(applied));
      setInvalidTime({ from: false, to: false });
    }
    setOpen(next);
  };

  const commit = (range: TimeRange | undefined) => {
    const scopes = params.scopes.filter((s) => s.id !== RANGE_SCOPE_ID);
    if (range) {
      scopes.push({
        id: RANGE_SCOPE_ID,
        value: formatRangeScope(range),
        readonly: true,
      });
    }
    onParamsChange({ ...params, scopes });
    setOpen(false);
  };

  const handleSubmit = (event: SubmitEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (draftRange && !error) commit(draftRange);
  };

  const pickDay = (picked: Date) => {
    const day = dayjs(picked).startOf("day").toDate();
    setDraft((d) => {
      const startsRange =
        d.activeEnd === "from" || (!!d.fromDay && day < d.fromDay);
      if (startsRange) {
        return {
          ...d,
          fromDay: day,
          toDay: d.toDay && d.toDay < day ? undefined : d.toDay,
          activeEnd: "to",
        };
      }
      return { ...d, toDay: day, activeEnd: "from" };
    });
  };

  const toggleTime = () => {
    setInvalidTime({ from: false, to: false });
    setDraft((d) =>
      d.showTime
        ? { ...d, showTime: false, fromTime: START_OF_DAY, toTime: END_OF_DAY }
        : { ...d, showTime: true }
    );
  };

  const endLabel = (end: RangeEnd) =>
    end === "from" ? t("common.from") : t("common.to");

  const dayButton = (end: RangeEnd) => {
    const day = end === "from" ? draft.fromDay : draft.toDay;
    const text = day
      ? formatDay(day.getTime())
      : end === "from"
        ? t("time-range.start-day")
        : t("time-range.end-day");
    return (
      <Button
        type="button"
        appearance="outline"
        aria-label={`${endLabel(end)}: ${text}`}
        aria-pressed={draft.activeEnd === end}
        onClick={() => setDraft((d) => ({ ...d, activeEnd: end }))}
        className={cn(
          "min-w-0 flex-1 justify-start font-normal tabular-nums",
          !compact && "px-2",
          day ? "text-main" : "text-control-placeholder",
          draft.activeEnd === end && "border-accent ring-1 ring-accent",
          compact && TAP_AREA_MD
        )}
      >
        <span className="truncate">{text}</span>
      </Button>
    );
  };

  const timeField = (end: RangeEnd, className: string) => (
    <TimeField
      className={className}
      aria-label={
        end === "from" ? t("time-range.from-time") : t("time-range.to-time")
      }
      value={end === "from" ? draft.fromTime : draft.toTime}
      invalid={invalidTime[end]}
      onInvalidChange={(invalid) =>
        setInvalidTime((current) => ({ ...current, [end]: invalid }))
      }
      onChange={(time) =>
        setDraft((d) =>
          end === "from" ? { ...d, fromTime: time } : { ...d, toTime: time }
        )
      }
    />
  );

  const errorLine = error && (
    <p role="alert" className="flex items-start gap-x-1.5 text-xs text-error">
      <CircleAlert className="mt-px size-3.5 shrink-0" />
      <span>{error}</span>
    </p>
  );

  const calendar = (
    <Calendar
      mode="range"
      size={compact ? "lg" : "md"}
      fixedWeeks
      // The month name belongs to its grid, so it sits closer than the 12px
      // between the popover's sections.
      classNames={{ month: "flex flex-col gap-2" }}
      defaultMonth={draft.fromDay ?? new Date()}
      selected={{ from: draft.fromDay, to: draft.toDay }}
      onDayClick={pickDay}
    />
  );

  const applyButton = (
    <Button
      type="submit"
      disabled={!draftRange || !!error}
      className={cn("ml-auto", compact && TAP_AREA_MD)}
    >
      {t("common.apply")}
    </Button>
  );

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <div
        ref={anchorRef}
        className="inline-flex max-w-full min-w-0 sm:shrink-0"
      >
        <PopoverTrigger
          className={cn(
            "inline-flex min-h-9 min-w-0 cursor-pointer flex-wrap items-center gap-x-2 rounded-xs border border-control-border bg-background px-3 py-1.5 text-sm text-control",
            "hover:bg-control-bg focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent",
            applied && "rounded-r-none",
            compact && TAP_AREA_MD
          )}
        >
          <CalendarIcon className="size-4 shrink-0 text-control-light" />
          <span className={cn(applied && "text-control-light")}>{label}</span>
          {applied && <AppliedRange range={applied} />}
        </PopoverTrigger>
        {applied && (
          <Button
            type="button"
            appearance="outline"
            aria-label={t("time-range.clear")}
            onClick={() => commit(undefined)}
            className={cn(
              "-ml-px self-stretch rounded-l-none px-2 text-control-light max-sm:h-auto",
              compact && TAP_AREA_MD
            )}
          >
            <X className="size-4" />
          </Button>
        )}
      </div>
      <PopoverContent
        align="end"
        anchor={anchorRef}
        aria-label={label}
        className="p-0"
      >
        <form
          noValidate
          onSubmit={handleSubmit}
          className={cn("flex", compact && "w-76 max-w-full flex-col")}
        >
          <QuickRanges
            compact={compact}
            active={
              draftRange
                ? matchQuickRange(draftRange)
                : draft.fromDay || draft.toDay
                  ? undefined
                  : ALL_TIME
            }
            onPick={(id) =>
              commit(id === ALL_TIME ? undefined : quickRange(id))
            }
          />
          <Separator
            orientation={compact ? "horizontal" : "vertical"}
            className={cn(!compact && "h-auto self-stretch")}
          />
          {compact ? (
            <div className="flex flex-col gap-y-3 p-3">
              <div className="flex items-center gap-x-2">
                {dayButton("from")}
                <ArrowRight className="size-3.5 shrink-0 text-control-light" />
                {dayButton("to")}
              </div>
              {draft.showTime && (
                <div className="flex items-center gap-x-2">
                  {timeField("from", "min-w-0 flex-1")}
                  <span className="size-3.5 shrink-0" />
                  {timeField("to", "min-w-0 flex-1")}
                </div>
              )}
              {errorLine}
              {calendar}
              <div className="flex items-center gap-x-2">
                <Button
                  type="button"
                  appearance="secondary"
                  onClick={toggleTime}
                  className={cn("-ml-2 px-2", TAP_AREA_MD)}
                >
                  {draft.showTime
                    ? t("time-range.remove-time")
                    : t("time-range.add-time")}
                </Button>
                {applyButton}
              </div>
            </div>
          ) : (
            <div className="flex flex-col gap-y-3 p-3">
              <div className="flex flex-col gap-y-2">
                {RANGE_ENDS.map((end) => (
                  <div key={end} className="flex items-center gap-x-2">
                    <span className="w-10 shrink-0 text-control-light">
                      {endLabel(end)}
                    </span>
                    {dayButton(end)}
                    {timeField(end, "w-18 shrink-0 text-center")}
                  </div>
                ))}
              </div>
              {errorLine}
              {calendar}
              <div className="flex">{applyButton}</div>
            </div>
          )}
        </form>
      </PopoverContent>
    </Popover>
  );
}

function AppliedRange({ range }: { range: TimeRange }) {
  const { t } = useTranslation();
  const quick = matchQuickRange(range);
  if (quick) {
    return <span className="text-main">{quickRangeLabel(t, quick)}</span>;
  }
  const wholeDays = isWholeDays(range);
  const format = (ts: number) =>
    wholeDays ? formatDay(ts) : `${formatDay(ts)} ${timeOfDay(ts)}`;
  const from = format(range.from);
  const to = format(range.to);
  return (
    <span className="flex flex-wrap items-center gap-x-1.5 text-main tabular-nums">
      <span className="whitespace-nowrap">{from}</span>
      {from !== to && (
        <>
          <ArrowRight className="size-3.5 shrink-0 text-control-light" />
          <span className="whitespace-nowrap">{to}</span>
        </>
      )}
    </span>
  );
}

function QuickRanges({
  compact,
  active,
  onPick,
}: {
  compact: boolean;
  active: QuickOption | undefined;
  onPick: (id: QuickOption) => void;
}) {
  const { t } = useTranslation();
  const label = (id: QuickOption) =>
    id === ALL_TIME ? t("time-range.all-time") : quickRangeLabel(t, id);
  if (compact) {
    return (
      <div
        role="group"
        aria-label={t("time-range.quick-ranges")}
        className="hide-scrollbar flex shrink-0 gap-x-2 overflow-x-auto p-3"
      >
        {QUICK_OPTIONS.map((id) => (
          <Button
            key={id}
            type="button"
            appearance="outline"
            size="sm"
            aria-pressed={active === id}
            onClick={() => onPick(id)}
            className={cn(
              "shrink-0 font-normal",
              TAP_AREA_SM,
              active === id && "border-accent bg-accent/10 text-accent"
            )}
          >
            {label(id)}
          </Button>
        ))}
      </div>
    );
  }
  return (
    <div
      role="group"
      aria-label={t("time-range.quick-ranges")}
      className="flex w-44 shrink-0 flex-col gap-y-1 rounded-l-sm bg-control-bg/50 p-3"
    >
      {QUICK_OPTIONS.map((id) => (
        <Button
          key={id}
          type="button"
          appearance="secondary"
          aria-pressed={active === id}
          onClick={() => onPick(id)}
          className={cn(
            "w-full justify-start px-2 font-normal hover:bg-control-bg-hover",
            active === id && "bg-accent/10 text-accent hover:bg-accent/10"
          )}
        >
          {/* The check keeps its slot when hidden, so labels stay aligned. */}
          <Check
            className={cn("size-4 shrink-0", active !== id && "invisible")}
          />
          {label(id)}
        </Button>
      ))}
    </div>
  );
}

function TimeField({
  value,
  onChange,
  invalid,
  onInvalidChange,
  className,
  ...props
}: Omit<ComponentProps<typeof Input>, "value" | "onChange"> & {
  value: string;
  onChange: (time: string) => void;
  invalid: boolean;
  onInvalidChange: (invalid: boolean) => void;
}) {
  const [text, setText] = useState(value);
  // Follow the committed time, but leave text that already reads as that time
  // alone, so typing "9" is not rewritten to "09:00" mid-entry.
  useEffect(() => {
    setText((current) => (parseTimeOfDay(current) === value ? current : value));
  }, [value]);

  const commit = () => {
    const time = parseTimeOfDay(text);
    onInvalidChange(!time);
    if (time) {
      setText(time);
      onChange(time);
    }
    return !!time;
  };

  return (
    <Input
      {...props}
      type="text"
      inputMode="numeric"
      autoComplete="off"
      spellCheck={false}
      aria-invalid={invalid}
      value={text}
      className={cn("tabular-nums", className)}
      onFocus={(event) => event.target.select()}
      onChange={(event) => {
        const next = event.target.value;
        setText(next);
        const time = parseTimeOfDay(next);
        if (!time) return;
        onChange(time);
        if (invalid) onInvalidChange(false);
      }}
      onBlur={commit}
      onKeyDown={(event) => {
        if (event.key === "Enter" && !commit()) event.preventDefault();
      }}
    />
  );
}
