import { Check, ChevronDown, Search, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useClickOutside } from "@/react/hooks/useClickOutside";
import { useEscapeKey } from "@/react/hooks/useEscapeKey";
import { cn } from "@/react/lib/utils";

export interface ComboboxOption {
  value: string;
  label: string;
  /** Secondary text shown below the label */
  description?: string;
  /** Custom render for the option row */
  render?: () => React.ReactNode;
}

interface ComboboxProps {
  value: string;
  options: ComboboxOption[];
  placeholder?: string;
  onChange: (value: string) => void | Promise<void>;
  /** Custom render for the selected value in the trigger */
  renderValue?: (option: ComboboxOption) => React.ReactNode;
  /** Text shown when no options match the search */
  noResultsText?: string;
  /** Server-side search callback. When provided, filters via this instead of client-side. */
  onSearch?: (query: string) => void;
  className?: string;
  disabled?: boolean;
}

export function Combobox({
  value,
  options,
  placeholder = "",
  onChange,
  renderValue,
  noResultsText,
  onSearch,
  className,
  disabled,
}: ComboboxProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const selectedOption = useMemo(
    () => options.find((o) => o.value === value),
    [options, value]
  );

  // Debounced server-side search
  useEffect(() => {
    if (!onSearch || !open) return;
    const timer = setTimeout(() => onSearch(search), 300);
    return () => clearTimeout(timer);
  }, [search, onSearch, open]);

  const filtered = useMemo(() => {
    if (onSearch) return options; // server handles filtering
    if (!search) return options;
    const q = search.toLowerCase();
    return options.filter(
      (o) =>
        o.label.toLowerCase().includes(q) ||
        (o.description?.toLowerCase().includes(q) ?? false) ||
        o.value.toLowerCase().includes(q)
    );
  }, [options, search]);

  const closeDropdown = useCallback(() => {
    setOpen(false);
    setSearch("");
  }, []);
  useClickOutside(containerRef, open, closeDropdown);
  useEscapeKey(open, closeDropdown);

  const handleSelect = useCallback(
    (optionValue: string) => {
      onChange(optionValue);
      setOpen(false);
      setSearch("");
    },
    [onChange]
  );

  const handleClear = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      onChange("");
      setSearch("");
    },
    [onChange]
  );

  return (
    <div ref={containerRef} className={cn("relative", className)}>
      {/* Trigger */}
      <button
        type="button"
        disabled={disabled}
        className={cn(
          "w-full flex items-center justify-between gap-2 border border-gray-300 rounded-md h-9 px-3 text-sm bg-white text-left transition-colors",
          "hover:border-gray-400",
          "disabled:opacity-50 disabled:pointer-events-none",
          open && "border-accent shadow-[0_0_0_1px_var(--color-accent)]"
        )}
        onClick={() => {
          if (!disabled) {
            setOpen(!open);
            if (!open) {
              setTimeout(() => inputRef.current?.focus(), 0);
            }
          }
        }}
      >
        <span className={cn("truncate", !selectedOption && "text-gray-400")}>
          {selectedOption
            ? renderValue
              ? renderValue(selectedOption)
              : selectedOption.label
            : placeholder}
        </span>
        <div className="flex items-center gap-1 shrink-0">
          {value && !disabled && (
            <span
              className="rounded-full p-0.5 hover:bg-gray-100 transition-colors"
              onClick={handleClear}
            >
              <X className="w-3 h-3 text-gray-400" />
            </span>
          )}
          <ChevronDown
            className={cn(
              "w-4 h-4 text-gray-400 transition-transform",
              open && "rotate-180"
            )}
          />
        </div>
      </button>

      {/* Dropdown */}
      {open && (
        <div className="absolute z-50 mt-1 w-full bg-white border border-gray-200 rounded-md shadow-lg overflow-hidden">
          {/* Search */}
          <div className="flex items-center gap-2 px-3 py-2 border-b border-gray-100">
            <Search className="w-4 h-4 text-gray-400 shrink-0" />
            <input
              ref={inputRef}
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={placeholder}
              className="w-full text-sm outline-none bg-transparent placeholder:text-gray-400"
            />
          </div>
          {/* Options */}
          <div className="max-h-60 overflow-y-auto">
            {filtered.length === 0 ? (
              <div className="px-3 py-6 text-sm text-gray-400 text-center">
                {noResultsText ?? "—"}
              </div>
            ) : (
              filtered.map((option) => {
                const isSelected = option.value === value;
                return (
                  <button
                    key={option.value}
                    type="button"
                    className={cn(
                      "w-full text-left px-3 py-2 text-sm flex items-center justify-between gap-2 transition-colors",
                      "hover:bg-gray-50",
                      isSelected && "bg-accent/5"
                    )}
                    onClick={() => handleSelect(option.value)}
                  >
                    <div className="flex flex-col min-w-0">
                      {option.render ? (
                        option.render()
                      ) : (
                        <>
                          <span
                            className={cn(
                              "truncate",
                              isSelected && "text-accent font-medium"
                            )}
                          >
                            {option.label}
                          </span>
                          {option.description && (
                            <span className="text-xs text-gray-400 truncate">
                              {option.description}
                            </span>
                          )}
                        </>
                      )}
                    </div>
                    {isSelected && (
                      <Check className="w-4 h-4 text-accent shrink-0" />
                    )}
                  </button>
                );
              })
            )}
          </div>
        </div>
      )}
    </div>
  );
}
