import { ChevronDown } from "lucide-react";
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";
import {
  getPortalDropdownStyle,
  isPortalDropdownStyleEqual,
  shouldIgnorePortalDropdownScroll,
} from "@/react/components/ui/combobox-position";
import { Input } from "@/react/components/ui/input";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "@/react/components/ui/layer";
import { cn } from "@/react/lib/utils";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { ColumnMetadata } from "@/types/proto-es/v1/database_service_pb";
import { getDataTypeSuggestionList } from "@/utils";
import { INLINE_EDIT_INPUT_CLASS } from "../common";

interface Props {
  column: ColumnMetadata;
  engine: Engine;
  readonly: boolean;
  onUpdateValue: (value: string) => void;
}

/**
 * Free-text type input with a clickable suggestion dropdown, mirroring the old
 * Vue `DropdownInput`: users can type any custom type (e.g. `varchar(255)`) or
 * pick one of the engine's known types from the dropdown. The native
 * `<input list>` it replaced never opened reliably on click.
 *
 * Focus/click handlers live on the wrapping container because Base UI's Input
 * can swallow `onClick`/`onFocus`; React focus/click events still bubble there.
 */
export function DataTypeCell({
  column,
  engine,
  readonly,
  onUpdateValue,
}: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [open, setOpen] = useState(false);
  const [dropdownStyle, setDropdownStyle] = useState<React.CSSProperties>({});

  const value = column.type ?? "";

  // The type as first rendered. We only filter once the user edits the value
  // away from this; an unchanged value shows the full list (matches Vue's
  // `allowFilter`). Otherwise opening e.g. a `bigint` cell would filter to just
  // "bigint" while a `timestamp(...)` cell — matching nothing — shows them all.
  const originalTypeRef = useRef(value);

  const allSuggestions = useMemo(
    () => getDataTypeSuggestionList(engine),
    [engine]
  );

  const suggestions = useMemo(() => {
    const q = value.trim().toLowerCase();
    if (!q || value === originalTypeRef.current) return allSuggestions;
    const filtered = allSuggestions.filter((t) => t.toLowerCase().includes(q));
    // Fall back to the full list when nothing matches the in-progress text so
    // the dropdown still offers the engine's types instead of going empty.
    return filtered.length > 0 ? filtered : allSuggestions;
  }, [allSuggestions, value]);

  // Position the portaled dropdown under the cell and keep it pinned while the
  // surrounding table scrolls. Mirrors the shared Combobox's portal handling.
  useLayoutEffect(() => {
    if (!open || !containerRef.current) return;

    const updatePosition = () => {
      if (!containerRef.current) return;
      const rect = containerRef.current.getBoundingClientRect();
      const height = dropdownRef.current?.offsetHeight ?? 0;
      const next = getPortalDropdownStyle(rect, height, window.innerHeight);
      setDropdownStyle((prev) =>
        isPortalDropdownStyleEqual(prev, next) ? prev : next
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
  }, [open, suggestions]);

  // Close on outside click (covers both the cell and the portaled dropdown).
  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      const target = e.target as Node;
      if (containerRef.current?.contains(target)) return;
      if (dropdownRef.current?.contains(target)) return;
      setOpen(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  // Open the dropdown on focus/click and close it on Escape. Attached to the
  // native input element (not the wrapper) so the wrapper stays a plain layout
  // div, and because Base UI's Input can swallow React onFocus/onClick props.
  useEffect(() => {
    const el = inputRef.current;
    if (!el || readonly) return;
    const handleOpen = () => setOpen(true);
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    el.addEventListener("focus", handleOpen);
    el.addEventListener("click", handleOpen);
    el.addEventListener("keydown", handleKeyDown);
    return () => {
      el.removeEventListener("focus", handleOpen);
      el.removeEventListener("click", handleOpen);
      el.removeEventListener("keydown", handleKeyDown);
    };
  }, [readonly]);

  const handleSelect = useCallback(
    (type: string) => {
      onUpdateValue(type);
      setOpen(false);
    },
    [onUpdateValue]
  );

  return (
    <div ref={containerRef} className="relative">
      <Input
        ref={inputRef}
        value={value}
        disabled={readonly}
        placeholder="column type"
        size="xs"
        className={cn(INLINE_EDIT_INPUT_CLASS, "pr-7")}
        onChange={(e) => onUpdateValue(e.target.value)}
      />
      {!readonly && (
        <ChevronDown className="pointer-events-none absolute right-2 top-1/2 size-4 -translate-y-1/2 text-control-light" />
      )}
      {open &&
        !readonly &&
        suggestions.length > 0 &&
        createPortal(
          <div
            ref={dropdownRef}
            style={dropdownStyle}
            className={cn(
              "max-h-60 overflow-y-auto rounded-sm border border-control-border bg-background py-1 shadow-lg",
              LAYER_SURFACE_CLASS
            )}
          >
            {suggestions.map((type) => (
              <button
                key={type}
                type="button"
                className={cn(
                  "block w-full px-3 py-1 text-left text-sm hover:bg-control-bg",
                  type === value && "bg-accent/5 text-accent"
                )}
                // onMouseDown (not onClick) so selection happens before the
                // input's blur / outside-click would close the dropdown.
                onMouseDown={(e) => {
                  e.preventDefault();
                  handleSelect(type);
                }}
              >
                {type}
              </button>
            ))}
          </div>,
          getLayerRoot("overlay")
        )}
    </div>
  );
}
