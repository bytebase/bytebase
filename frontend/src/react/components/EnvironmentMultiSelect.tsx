import { Check, ChevronDown, X } from "lucide-react";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { useEnvironmentList } from "@/react/hooks/useAppState";
import { cn } from "@/react/lib/utils";

export function EnvironmentMultiSelect({
  value,
  onChange,
}: Readonly<{
  value: string[];
  onChange: (envs: string[]) => void;
}>) {
  const { t } = useTranslation();
  const environmentList = useEnvironmentList();
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const toggle = (name: string) => {
    onChange(
      value.includes(name) ? value.filter((v) => v !== name) : [...value, name]
    );
  };

  const remove = (name: string) => {
    onChange(value.filter((v) => v !== name));
  };

  // Use native <button> elements for the trigger and option rows so they
  // get keyboard (Enter/Space) and focus semantics for free — SonarCloud's
  // S6848 / S1082 a11y rules require interactive content to be native.
  // The selected-chip X buttons are rendered as siblings of the trigger
  // (not children) to avoid nested <button>, which is invalid HTML.
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <div ref={containerRef} className="relative">
        <div
          className={cn(
            "flex items-center flex-wrap gap-1 min-h-[2.25rem] w-full rounded-xs border border-control-border bg-transparent px-2 py-1 text-sm",
            open && "border-accent"
          )}
        >
          {value.length === 0 && (
            <span className="text-control-placeholder">
              {t("environment.select")}
            </span>
          )}
          {value.map((name) => (
            <span
              key={name}
              className="inline-flex items-center gap-x-1 rounded-xs border border-control-border px-1 py-0.5 text-xs"
            >
              <EnvironmentLabel environmentName={name} className="text-xs" />
              <button
                type="button"
                className="text-control-light hover:text-control"
                onClick={() => remove(name)}
                aria-label={t("common.delete")}
              >
                <X className="h-3 w-3" />
              </button>
            </span>
          ))}
          <PopoverTrigger
            render={
              <button
                type="button"
                className="ml-auto flex items-center text-control-light cursor-pointer"
                aria-label={t("environment.select")}
              />
            }
          >
            <ChevronDown className="h-4 w-4 shrink-0" />
          </PopoverTrigger>
        </div>
      </div>
      <PopoverContent
        anchor={containerRef}
        side="bottom"
        align="start"
        initialFocus={false}
        finalFocus={false}
        style={{ width: "var(--anchor-width)" }}
        className="max-h-60 overflow-auto p-0"
      >
        {environmentList.length === 0 && (
          <div className="px-3 py-2 text-sm text-control-light">
            {t("common.no-data")}
          </div>
        )}
        {environmentList.map((env) => {
          const selected = value.includes(env.name);
          return (
            <button
              key={env.name}
              type="button"
              className="flex w-full cursor-pointer items-center gap-x-2 px-3 py-1.5 text-left text-sm hover:bg-control-bg"
              onClick={() => toggle(env.name)}
              aria-pressed={selected}
            >
              <div
                className={cn(
                  "flex size-4 shrink-0 items-center justify-center rounded-xs border",
                  selected
                    ? "border-accent bg-accent text-accent-text"
                    : "border-control-border"
                )}
              >
                {selected && <Check className="h-3 w-3" />}
              </div>
              <EnvironmentLabel environment={env} />
            </button>
          );
        })}
      </PopoverContent>
    </Popover>
  );
}
