import { ChevronDown, X } from "lucide-react";
import { useCallback, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Checkbox } from "@/react/components/ui/checkbox";
import { LAYER_SURFACE_CLASS } from "@/react/components/ui/layer";
import { useClickOutside } from "@/react/hooks/useClickOutside";
import { cn } from "@/react/lib/utils";

export interface IssueLabel {
  value: string;
  color: string;
}

interface IssueLabelSelectProps {
  labels: IssueLabel[];
  selected: string[];
  required: boolean;
  onChange: (labels: string[]) => void;
}

/**
 * IssueLabelSelect — label multi-select for issue creation.
 *
 * Rendered as a dropdown with chip-style selected labels. Shared between
 * drawers that need to attach labels to a new issue (e.g. Data Export,
 * Request Role).
 */
export function IssueLabelSelect({
  labels,
  selected,
  required,
  onChange,
}: IssueLabelSelectProps) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const closeDropdown = useCallback(() => setOpen(false), []);
  useClickOutside(containerRef, open, closeDropdown);

  const toggleLabel = (value: string) => {
    onChange(
      selected.includes(value)
        ? selected.filter((l) => l !== value)
        : [...selected, value]
    );
  };

  return (
    <div className="flex flex-col gap-y-2">
      <label className="text-sm font-medium text-control">
        {t("issue.labels")}
        {required && <span className="text-error"> *</span>}
      </label>
      <div ref={containerRef} className="relative">
        <button
          type="button"
          className={cn(
            "w-full flex items-center justify-between gap-2 border border-control-border rounded-sm h-9 px-3 text-sm bg-background text-left transition-colors",
            "hover:border-control-border",
            open && "border-accent shadow-[0_0_0_1px_var(--color-accent)]"
          )}
          onClick={() => setOpen(!open)}
        >
          {selected.length > 0 ? (
            <div className="flex items-center gap-1.5 truncate">
              {selected.map((val) => {
                const label = labels.find((l) => l.value === val);
                return (
                  <span
                    key={val}
                    className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-xs bg-control-bg text-xs"
                  >
                    <span
                      className="size-2.5 rounded-sm shrink-0"
                      style={{ backgroundColor: label?.color }}
                    />
                    {val}
                    <X
                      className="size-3 text-control-placeholder hover:text-control-light"
                      onClick={(e) => {
                        e.stopPropagation();
                        toggleLabel(val);
                      }}
                    />
                  </span>
                );
              })}
            </div>
          ) : (
            <span className="text-control-placeholder">
              {t("common.select")}
            </span>
          )}
          <ChevronDown
            className={cn(
              "size-4 text-control-placeholder shrink-0 transition-transform",
              open && "rotate-180"
            )}
          />
        </button>
        {open && (
          <div
            className={cn(
              "absolute mt-1 w-full bg-background border border-block-border rounded-sm shadow-lg overflow-hidden",
              LAYER_SURFACE_CLASS
            )}
          >
            <div className="max-h-60 overflow-y-auto">
              {labels.length === 0 ? (
                <div className="px-3 py-6 text-sm text-control-placeholder text-center">
                  {t("common.no-data")}
                </div>
              ) : (
                labels.map((label) => {
                  const isSelected = selected.includes(label.value);
                  return (
                    <button
                      key={label.value}
                      type="button"
                      className="w-full text-left px-3 py-2 text-sm flex items-center gap-2 hover:bg-control-bg transition-colors"
                      onClick={() => toggleLabel(label.value)}
                    >
                      <Checkbox checked={isSelected} />
                      <span
                        className="size-4 rounded-sm shrink-0"
                        style={{ backgroundColor: label.color }}
                      />
                      <span>{label.value}</span>
                    </button>
                  );
                })
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
