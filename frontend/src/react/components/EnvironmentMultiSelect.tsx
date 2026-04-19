import { Check, ChevronDown, X } from "lucide-react";
import type { KeyboardEvent } from "react";
import { useCallback, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { useClickOutside } from "@/react/hooks/useClickOutside";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useEnvironmentV1Store } from "@/store";

export function EnvironmentMultiSelect({
  value,
  onChange,
}: {
  value: string[];
  onChange: (envs: string[]) => void;
}) {
  const { t } = useTranslation();
  const environmentStore = useEnvironmentV1Store();
  const environmentList = useVueState(
    () => environmentStore.environmentList ?? []
  );
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const handleClickOutside = useCallback(() => setOpen(false), []);
  useClickOutside(containerRef, open, handleClickOutside);

  const toggle = (name: string) => {
    onChange(
      value.includes(name) ? value.filter((v) => v !== name) : [...value, name]
    );
  };

  const remove = (name: string) => {
    onChange(value.filter((v) => v !== name));
  };

  // Keyboard activation for the trigger and option rows. Base UI's Select
  // would handle this natively, but we render a custom combobox here because
  // the trigger hosts inline chip buttons (removing a selected env must not
  // toggle the popover). Enter/Space toggle or select; Escape closes.
  const handleTriggerKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      setOpen((o) => !o);
    } else if (e.key === "Escape" && open) {
      e.preventDefault();
      setOpen(false);
    }
  };

  const handleOptionKeyDown = (
    e: KeyboardEvent<HTMLDivElement>,
    name: string
  ) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      toggle(name);
    }
  };

  return (
    <div ref={containerRef} className="relative">
      <div
        role="combobox"
        aria-expanded={open}
        aria-haspopup="listbox"
        tabIndex={0}
        className={cn(
          "flex items-center flex-wrap gap-1 min-h-[2.25rem] w-full rounded-xs border border-control-border bg-transparent px-2 py-1 text-sm cursor-pointer",
          open && "border-accent"
        )}
        onClick={() => setOpen(!open)}
        onKeyDown={handleTriggerKeyDown}
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
              onClick={(e) => {
                e.stopPropagation();
                remove(name);
              }}
            >
              <X className="h-3 w-3" />
            </button>
          </span>
        ))}
        <ChevronDown className="ml-auto h-4 w-4 shrink-0 text-control-light" />
      </div>

      {open && (
        <div
          role="listbox"
          aria-multiselectable="true"
          className="absolute z-50 mt-1 w-full bg-background border border-control-border rounded-sm shadow-lg max-h-60 overflow-auto"
        >
          {environmentList.length === 0 && (
            <div className="px-3 py-2 text-sm text-control-light">
              {t("common.no-data")}
            </div>
          )}
          {environmentList.map((env) => {
            const selected = value.includes(env.name);
            return (
              <div
                key={env.name}
                role="option"
                aria-selected={selected}
                tabIndex={0}
                className="flex items-center gap-x-2 px-3 py-1.5 text-sm hover:bg-control-bg cursor-pointer"
                onClick={() => toggle(env.name)}
                onKeyDown={(e) => handleOptionKeyDown(e, env.name)}
              >
                <div
                  className={cn(
                    "size-4 rounded-xs border flex items-center justify-center shrink-0",
                    selected
                      ? "bg-accent border-accent text-accent-text"
                      : "border-control-border"
                  )}
                >
                  {selected && <Check className="h-3 w-3" />}
                </div>
                <EnvironmentLabel environment={env} />
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
