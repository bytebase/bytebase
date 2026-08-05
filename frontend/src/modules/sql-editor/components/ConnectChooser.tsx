import { Check, Network } from "lucide-react";
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import type { ComboboxOption } from "@/components/ui/combobox";
import {
  getPortalDropdownStyle,
  isPortalDropdownStyleEqual,
  shouldIgnorePortalDropdownScroll,
} from "@/components/ui/combobox-position";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "@/components/ui/layer";
import { SearchInput } from "@/components/ui/search-input";
import { cn } from "@/lib/utils";
import { ConnectionChooserButton } from "./ConnectionChooserButton";

export type { ComboboxOption };

type ConnectChooserProps = {
  readonly value: string;
  readonly onChange: (value: string) => void;
  readonly options: ComboboxOption[];
  readonly isChosen: boolean;
  readonly placeholder: string;
  readonly dropdownClassName?: string;
  readonly dropdownMinWidth?: number;
  readonly openSignal?: number;
  readonly triggerClassName?: string;
  readonly triggerVariant?: "connection" | "run";
};

/**
 * Replaces frontend/src/views/sql-editor/EditorCommon/ConnectChooser.vue.
 * Schema/Container chooser button used inside the SQL Editor toolbar's
 * NButtonGroup. Renders a chosen value (truncated) or a placeholder hint;
 * clicking opens a dropdown popover for selection.
 *
 * Note: uses a custom trigger button + portal dropdown rather than the
 * shared <Combobox> component, because <Combobox> does not expose a
 * renderTrigger prop for custom trigger styling.
 */
export function ConnectChooser({
  value,
  onChange,
  options,
  isChosen,
  placeholder,
  dropdownClassName,
  dropdownMinWidth,
  openSignal,
  triggerClassName,
  triggerVariant = "connection",
}: ConnectChooserProps) {
  const { t } = useTranslation();
  const displayValue = value || t("db.schema.default");

  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const triggerRef = useRef<HTMLButtonElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [dropdownStyle, setDropdownStyle] = useState<React.CSSProperties>({});
  const lastOpenSignalRef = useRef(openSignal);

  const filteredOptions = useMemo(() => {
    if (!search) return options;
    const q = search.toLowerCase();
    return options.filter(
      (o) =>
        o.label.toLowerCase().includes(q) || o.value.toLowerCase().includes(q)
    );
  }, [options, search]);

  // Position dropdown in portal
  useLayoutEffect(() => {
    if (!open || !triggerRef.current) return;

    const updatePosition = () => {
      if (!triggerRef.current) return;
      const rect = triggerRef.current.getBoundingClientRect();
      const dropdownHeight = dropdownRef.current?.offsetHeight ?? 0;
      const nextStyle = getPortalDropdownStyle(
        rect,
        dropdownHeight,
        window.innerHeight,
        {
          minWidth: dropdownMinWidth,
          viewportWidth: window.innerWidth,
        }
      );
      setDropdownStyle((prev) =>
        isPortalDropdownStyleEqual(prev, nextStyle) ? prev : nextStyle
      );
    };

    const handleScroll = (event: Event) => {
      if (shouldIgnorePortalDropdownScroll(event.target, dropdownRef.current)) {
        return;
      }
      updatePosition();
    };

    updatePosition();
    window.addEventListener("resize", updatePosition);
    window.addEventListener("scroll", handleScroll, true);
    return () => {
      window.removeEventListener("resize", updatePosition);
      window.removeEventListener("scroll", handleScroll, true);
    };
  }, [open]);

  // Click outside to close
  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      const target = e.target as Node;
      if (triggerRef.current?.contains(target)) return;
      if (dropdownRef.current?.contains(target)) return;
      setOpen(false);
      setSearch("");
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  // Escape to close
  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setOpen(false);
        setSearch("");
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [open]);

  // Focus search input when opened
  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 10);
    }
  }, [open]);

  useEffect(() => {
    if (openSignal === undefined) return;
    if (lastOpenSignalRef.current === openSignal) return;
    lastOpenSignalRef.current = openSignal;
    setOpen(true);
    setSearch("");
  }, [openSignal]);

  const handleSelect = useCallback(
    (optionValue: string) => {
      onChange(optionValue);
      setOpen(false);
      setSearch("");
    },
    [onChange]
  );

  const dropdownContent = (
    <div
      ref={dropdownRef}
      style={dropdownStyle}
      className={cn(
        "bg-background border border-control-border rounded-sm shadow-lg overflow-hidden",
        LAYER_SURFACE_CLASS,
        dropdownClassName
      )}
    >
      <SearchInput
        ref={inputRef}
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        wrapperClassName="m-2"
        className="h-7"
      />
      <div className="max-h-60 overflow-y-auto">
        {filteredOptions.length === 0 ? (
          <div className="px-3 py-6 text-sm text-control-placeholder text-center">
            —
          </div>
        ) : (
          filteredOptions.map((option) => {
            const isSelected = option.value === value;
            return (
              <button
                key={option.value}
                type="button"
                className={cn(
                  "w-full text-left px-3 py-1.5 text-sm flex items-center gap-x-2 transition-colors",
                  "hover:bg-control-bg",
                  isSelected && "bg-accent/5"
                )}
                onClick={() => handleSelect(option.value)}
              >
                <span
                  className={cn(
                    "truncate flex-1",
                    isSelected && "text-accent font-medium"
                  )}
                >
                  {option.label}
                </span>
                {isSelected && (
                  <Check className="size-4 text-accent shrink-0" />
                )}
              </button>
            );
          })
        )}
      </div>
    </div>
  );

  const triggerContent = (
    <>
      <Network
        className={cn(
          "size-4 shrink-0",
          triggerVariant === "run"
            ? "text-current"
            : isChosen
              ? "text-accent"
              : "text-control-placeholder"
        )}
      />
      {isChosen ? (
        <span
          className={cn(
            "truncate",
            triggerVariant === "connection" && "text-control"
          )}
        >
          {displayValue}
        </span>
      ) : (
        <span
          className={cn(
            "whitespace-nowrap",
            triggerVariant === "connection" && "text-control-placeholder"
          )}
        >
          {placeholder}
        </span>
      )}
    </>
  );

  return (
    <>
      {triggerVariant === "run" ? (
        <Button
          ref={triggerRef}
          variant="default"
          size="sm"
          className={cn(
            "h-7 min-w-[10rem] px-1.5 gap-1 rounded-r-none text-sm",
            triggerClassName
          )}
          aria-label={placeholder}
          aria-expanded={open}
          onClick={() => setOpen((v) => !v)}
        >
          {triggerContent}
        </Button>
      ) : (
        <ConnectionChooserButton
          ref={triggerRef}
          className={cn(
            isChosen ? "max-w-[12rem]" : "max-w-none",
            triggerClassName
          )}
          aria-label={placeholder}
          aria-expanded={open}
          onClick={() => setOpen((v) => !v)}
        >
          {triggerContent}
        </ConnectionChooserButton>
      )}
      {open && createPortal(dropdownContent, getLayerRoot("overlay"))}
    </>
  );
}
