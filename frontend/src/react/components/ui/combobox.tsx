import { Check, ChevronDown, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { cn } from "@/react/lib/utils";
import { SearchInput } from "./search-input";

export interface ComboboxOption {
  value: string;
  label: string;
  /** Secondary text shown below the label */
  description?: string;
  /** Custom render for the option row */
  render?: () => React.ReactNode;
  /** Whether this option is disabled (shown but not selectable) */
  disabled?: boolean;
}

export interface ComboboxGroup {
  label: string;
  options: ComboboxOption[];
}

// ---- Single-select props ----
interface ComboboxSingleProps {
  multiple?: false;
  value: string;
  onChange: (value: string) => void | Promise<void>;
}

// ---- Multi-select props ----
interface ComboboxMultiProps {
  multiple: true;
  value: string[];
  onChange: (value: string[]) => void;
}

type ComboboxBaseProps = {
  /** Flat options list OR grouped options. */
  options: ComboboxOption[] | ComboboxGroup[];
  placeholder?: string;
  /** Custom render for the selected value in the trigger (single mode only) */
  renderValue?: (option: ComboboxOption) => React.ReactNode;
  /** Text shown when no options match the search */
  noResultsText?: string;
  /** Server-side search callback. When provided, filters via this instead of client-side. */
  onSearch?: (query: string) => void;
  className?: string;
  disabled?: boolean;
  /** Render dropdown via portal (use when inside overflow:hidden containers like modals) */
  portal?: boolean;
};

type ComboboxProps = ComboboxBaseProps &
  (ComboboxSingleProps | ComboboxMultiProps);

function isGrouped(
  options: ComboboxOption[] | ComboboxGroup[]
): options is ComboboxGroup[] {
  return options.length > 0 && "options" in options[0];
}

function flattenOptions(
  options: ComboboxOption[] | ComboboxGroup[]
): ComboboxOption[] {
  if (isGrouped(options)) {
    return options.flatMap((g) => g.options);
  }
  return options;
}

export function Combobox(props: ComboboxProps) {
  const {
    options,
    placeholder = "",
    renderValue,
    noResultsText,
    onSearch,
    className,
    disabled,
    portal,
  } = props;
  const multiple = props.multiple === true;

  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [dropdownStyle, setDropdownStyle] = useState<React.CSSProperties>({});

  const allOptions = useMemo(() => flattenOptions(options), [options]);

  const selectedValues: string[] = multiple
    ? (props as ComboboxMultiProps).value
    : (props as ComboboxSingleProps).value
      ? [(props as ComboboxSingleProps).value]
      : [];

  const selectedOptions = useMemo(
    () => allOptions.filter((o) => selectedValues.includes(o.value)),
    [allOptions, selectedValues]
  );

  // Position dropdown for portal mode
  useEffect(() => {
    if (!open || !portal || !containerRef.current) return;
    const rect = containerRef.current.getBoundingClientRect();
    setDropdownStyle({
      position: "fixed",
      top: rect.bottom + 4,
      left: rect.left,
      width: rect.width,
    });
  }, [open, portal]);

  // Debounced server-side search
  useEffect(() => {
    if (!onSearch || !open) return;
    const timer = setTimeout(() => onSearch(search), 300);
    return () => clearTimeout(timer);
  }, [search, onSearch, open]);

  // Filter options
  const filteredGroups: ComboboxGroup[] = useMemo(() => {
    const groups = isGrouped(options)
      ? options
      : [{ label: "", options: options as ComboboxOption[] }];

    if (onSearch || !search) return groups;

    const q = search.toLowerCase();
    return groups
      .map((g) => ({
        ...g,
        options: g.options.filter(
          (o) =>
            o.label.toLowerCase().includes(q) ||
            (o.description?.toLowerCase().includes(q) ?? false) ||
            o.value.toLowerCase().includes(q)
        ),
      }))
      .filter((g) => g.options.length > 0);
  }, [options, search, onSearch]);

  // Click outside (handles both container and portal dropdown)
  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      const target = e.target as Node;
      if (containerRef.current?.contains(target)) return;
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

  const handleSelect = useCallback(
    (optionValue: string) => {
      if (multiple) {
        const multiOnChange = (props as ComboboxMultiProps).onChange;
        const current = (props as ComboboxMultiProps).value;
        if (current.includes(optionValue)) {
          multiOnChange(current.filter((v) => v !== optionValue));
        } else {
          multiOnChange([...current, optionValue]);
        }
      } else {
        (props as ComboboxSingleProps).onChange(optionValue);
        setOpen(false);
        setSearch("");
      }
    },
    [multiple, props]
  );

  const handleRemoveChip = useCallback(
    (optionValue: string, e: React.MouseEvent) => {
      e.stopPropagation();
      if (multiple) {
        const multiOnChange = (props as ComboboxMultiProps).onChange;
        const current = (props as ComboboxMultiProps).value;
        multiOnChange(current.filter((v) => v !== optionValue));
      }
    },
    [multiple, props]
  );

  const handleClear = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      if (multiple) {
        (props as ComboboxMultiProps).onChange([]);
      } else {
        (props as ComboboxSingleProps).onChange("");
      }
      setSearch("");
    },
    [multiple, props]
  );

  const hasValue = selectedValues.length > 0;

  // ---- Render trigger ----
  const renderTrigger = () => {
    if (multiple) {
      return (
        <>
          {selectedOptions.map((opt) => (
            <span
              key={opt.value}
              className="inline-flex items-center gap-x-1 rounded-xs bg-gray-100 px-1.5 py-0.5 text-xs"
            >
              {opt.label}
              {!disabled && (
                <button
                  type="button"
                  className="hover:text-error"
                  onClick={(e) => handleRemoveChip(opt.value, e)}
                >
                  <X className="h-3 w-3" />
                </button>
              )}
            </span>
          ))}
          {selectedValues.length === 0 && (
            <span className="text-control-placeholder">{placeholder}</span>
          )}
        </>
      );
    }

    const singleSelected = selectedOptions[0];
    return (
      <span
        className={cn(
          "truncate",
          !singleSelected && "text-control-placeholder"
        )}
      >
        {singleSelected
          ? renderValue
            ? renderValue(singleSelected)
            : singleSelected.label
          : placeholder}
      </span>
    );
  };

  // ---- Render option row ----
  const renderOptionRow = (option: ComboboxOption) => {
    const isSelected = selectedValues.includes(option.value);
    return (
      <button
        key={option.value}
        type="button"
        disabled={option.disabled}
        className={cn(
          "w-full text-left px-3 py-1.5 text-sm flex items-center gap-x-2 transition-colors",
          "hover:bg-gray-50",
          isSelected && "bg-accent/5",
          option.disabled && "opacity-50 cursor-not-allowed"
        )}
        onClick={() => !option.disabled && handleSelect(option.value)}
      >
        {multiple && (
          <div
            className={cn(
              "h-4 w-4 rounded-xs border flex items-center justify-center shrink-0",
              isSelected
                ? "bg-accent border-accent text-white"
                : "border-control-border"
            )}
          >
            {isSelected && <Check className="h-3 w-3" />}
          </div>
        )}
        <div className="flex flex-col min-w-0 flex-1">
          {option.render ? (
            option.render()
          ) : (
            <>
              <span
                className={cn(
                  "truncate",
                  !multiple && isSelected && "text-accent font-medium"
                )}
              >
                {option.label}
              </span>
              {option.description && (
                <span className="text-xs text-control-light truncate">
                  {option.description}
                </span>
              )}
            </>
          )}
        </div>
        {!multiple && isSelected && (
          <Check className="w-4 h-4 text-accent shrink-0" />
        )}
      </button>
    );
  };

  // ---- Render dropdown content ----
  const dropdownContent = (
    <div
      ref={dropdownRef}
      style={portal ? dropdownStyle : undefined}
      className={cn(
        "bg-white border border-control-border rounded-sm shadow-lg overflow-hidden",
        portal ? "z-[999]" : "absolute z-50 mt-1 min-w-full w-max"
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
        {filteredGroups.every((g) => g.options.length === 0) ? (
          <div className="px-3 py-6 text-sm text-control-placeholder text-center">
            {noResultsText ?? "—"}
          </div>
        ) : (
          filteredGroups.map((group) => (
            <div key={group.label}>
              {group.label && (
                <div className="px-3 py-1.5 text-xs font-medium text-control-light uppercase tracking-wide bg-gray-50">
                  {group.label}
                </div>
              )}
              {group.options.map(renderOptionRow)}
            </div>
          ))
        )}
      </div>
    </div>
  );

  return (
    <div ref={containerRef} className={cn("relative", className)}>
      {/* Trigger */}
      <div
        className={cn(
          "flex flex-wrap items-center gap-1 min-h-[2.25rem] w-full rounded-xs border border-control-border bg-white px-2 py-1 text-sm cursor-pointer",
          disabled && "opacity-50 cursor-not-allowed",
          open && "border-accent"
        )}
        onClick={() => {
          if (!disabled) {
            setOpen(!open);
          }
        }}
      >
        {renderTrigger()}
        <div className="flex items-center gap-1 shrink-0 ml-auto">
          {hasValue && !disabled && (
            <span
              className="rounded-full p-0.5 hover:bg-gray-100 transition-colors"
              onClick={handleClear}
            >
              <X className="w-3 h-3 text-control-light" />
            </span>
          )}
          <ChevronDown className="w-4 h-4 text-control-light" />
        </div>
      </div>

      {/* Dropdown */}
      {open &&
        (portal
          ? createPortal(dropdownContent, document.body)
          : dropdownContent)}
    </div>
  );
}
