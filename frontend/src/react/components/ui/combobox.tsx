import { Check, ChevronDown, X } from "lucide-react";
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";
import { cn } from "@/react/lib/utils";
import { HighlightLabelText } from "../HighlightLabelText";
import {
  getPortalDropdownStyle,
  isPortalDropdownStyleEqual,
  shouldIgnorePortalDropdownScroll,
} from "./combobox-position";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "./layer";
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
  /** Text shown when no options match the search. Ignored when
   *  `noResultsContent` is provided. */
  noResultsText?: string;
  /** Custom rich empty-state node — replaces `noResultsText`. Useful when
   *  the empty state needs links or conditional copy beyond a flat string. */
  noResultsContent?: React.ReactNode;
  /** Server-side search callback. When provided, filters via this instead of client-side. */
  onSearch?: (query: string) => void;
  className?: string;
  disabled?: boolean;
  clearable?: boolean;
  /** Trigger size — matches the Input component's tier names. Defaults to `md`. */
  size?: "sm" | "md";
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
    noResultsContent,
    onSearch,
    className,
    disabled,
    clearable = true,
    size = "md",
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

  // Position dropdown for portal mode — useLayoutEffect prevents a
  // one-frame flash at a stale/empty position before the browser paints.
  useLayoutEffect(() => {
    if (!open || !portal || !containerRef.current) return;

    const updateDropdownPosition = () => {
      if (!containerRef.current) return;
      const rect = containerRef.current.getBoundingClientRect();
      const dropdownHeight = dropdownRef.current?.offsetHeight ?? 0;
      const nextStyle = getPortalDropdownStyle(
        rect,
        dropdownHeight,
        window.innerHeight
      );
      setDropdownStyle((previousStyle) =>
        isPortalDropdownStyleEqual(previousStyle, nextStyle)
          ? previousStyle
          : nextStyle
      );
    };

    const handleScroll = (event: Event) => {
      if (shouldIgnorePortalDropdownScroll(event.target, dropdownRef.current)) {
        return;
      }
      updateDropdownPosition();
    };

    updateDropdownPosition();
    window.addEventListener("resize", updateDropdownPosition);
    window.addEventListener("scroll", handleScroll, true);

    return () => {
      window.removeEventListener("resize", updateDropdownPosition);
      window.removeEventListener("scroll", handleScroll, true);
    };
  }, [open, portal, filteredGroups]);

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
              className="inline-flex items-center gap-x-1 rounded-xs bg-control-bg px-1.5 py-0.5 text-xs"
            >
              {opt.label}
              {!disabled && (
                <button
                  type="button"
                  className="hover:text-error"
                  onClick={(e) => handleRemoveChip(opt.value, e)}
                >
                  <X className="size-3" />
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
          // `min-w-0` ensures the span can shrink below its content
          // width in the flex trigger row so `truncate` actually
          // applies. Without it, flex's default `min-width: auto`
          // sizes the span to the longest unbreakable text run and the
          // X/chevron get pushed onto a new line in narrow sidebars.
          "truncate min-w-0",
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
          "hover:bg-control-bg",
          isSelected && "bg-accent/5",
          option.disabled && "opacity-50 cursor-not-allowed"
        )}
        onClick={() => !option.disabled && handleSelect(option.value)}
      >
        {multiple && (
          <div
            className={cn(
              "size-4 rounded-xs border flex items-center justify-center shrink-0",
              isSelected
                ? "bg-accent border-accent text-accent-text"
                : "border-control-border"
            )}
          >
            {isSelected && <Check className="size-3" />}
          </div>
        )}
        <div className="flex flex-col min-w-0 flex-1">
          {option.render ? (
            option.render()
          ) : (
            <>
              <HighlightLabelText
                text={option.label}
                keyword={search}
                className={cn(
                  "truncate",
                  !multiple && isSelected && "text-accent font-medium"
                )}
              />
              {option.description && (
                <HighlightLabelText
                  text={option.description}
                  keyword={search}
                  className="text-xs text-control-light truncate"
                />
              )}
            </>
          )}
        </div>
        {!multiple && isSelected && (
          <Check className="size-4 text-accent shrink-0" />
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
        "bg-background border border-control-border rounded-sm shadow-lg overflow-hidden",
        portal && LAYER_SURFACE_CLASS,
        !portal && "absolute z-50 mt-1 min-w-full w-max"
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
          noResultsContent !== undefined ? (
            <div className="p-3">{noResultsContent}</div>
          ) : (
            <div className="px-3 py-6 text-sm text-control-placeholder text-center">
              {noResultsText ?? "—"}
            </div>
          )
        ) : (
          filteredGroups.map((group) => (
            <div key={group.label}>
              {group.label && (
                <div className="px-3 py-1.5 text-xs font-medium text-control-light uppercase tracking-wide bg-control-bg">
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
          // `flex-wrap` only matters for multi-select where chips can spill
          // onto a new line; in single-select mode it lets the X/chevron
          // wrap below a too-narrow label on a narrow container, which
          // looks broken. Keep the trigger on a single line in that case
          // and let the label truncate inside `renderTrigger`.
          "flex items-center gap-1 w-full rounded-xs border border-control-border bg-background py-1 cursor-pointer",
          size === "sm"
            ? "min-h-7 px-2 text-xs leading-4"
            : "min-h-9 px-3 text-sm leading-5",
          multiple && "flex-wrap",
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
          {clearable && hasValue && !disabled && (
            <span
              className="rounded-full p-0.5 hover:bg-control-bg transition-colors"
              onClick={handleClear}
            >
              <X className="size-3 text-control-light" />
            </span>
          )}
          <ChevronDown className="size-4 text-control-light" />
        </div>
      </div>

      {/* Dropdown */}
      {open &&
        (portal
          ? createPortal(dropdownContent, getLayerRoot("overlay"))
          : dropdownContent)}
    </div>
  );
}
