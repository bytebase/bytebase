import { ChevronDown } from "lucide-react";
import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useId,
  useMemo,
  useRef,
  useState,
} from "react";
import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { cn } from "@/lib/utils";

interface TilePickerContextValue {
  disabled: boolean;
  onItemClick: () => void;
}

const TilePickerContext = createContext<TilePickerContextValue>({
  disabled: false,
  onItemClick: () => undefined,
});

// Collapsible radio-tile picker that collapses to a single-line summary bar
// showing the active choice and expands into a responsive grid of tiles.
export function CollapsibleTilePicker({
  label,
  summary,
  value,
  onValueChange,
  collapsed: collapsedProp,
  defaultCollapsed = false,
  onCollapsedChange,
  disabled = false,
  children,
}: {
  label: string;
  summary: ReactNode;
  value: string;
  onValueChange: (value: string) => void;
  collapsed?: boolean;
  defaultCollapsed?: boolean;
  onCollapsedChange?: (collapsed: boolean) => void;
  disabled?: boolean;
  children: ReactNode;
}) {
  const panelId = useId();
  const triggerRef = useRef<HTMLButtonElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const isControlled = collapsedProp !== undefined;
  const [internalCollapsed, setInternalCollapsed] = useState(defaultCollapsed);
  const isCollapsed = isControlled ? collapsedProp : internalCollapsed;

  const setCollapsed = useCallback(
    (next: boolean) => {
      if (!isControlled) {
        setInternalCollapsed(next);
      }
      onCollapsedChange?.(next);
    },
    [isControlled, onCollapsedChange]
  );

  const collapse = useCallback(() => {
    if (panelRef.current?.contains(document.activeElement)) {
      triggerRef.current?.focus();
    }
    setCollapsed(true);
  }, [setCollapsed]);

  const contextValue = useMemo(
    () => ({
      disabled,
      onItemClick: collapse,
    }),
    [disabled, collapse]
  );

  return (
    <TilePickerContext.Provider value={contextValue}>
      <Button
        ref={triggerRef}
        type="button"
        appearance="outline"
        className="w-full justify-between"
        disabled={disabled}
        aria-expanded={!isCollapsed}
        aria-controls={panelId}
        onClick={() => setCollapsed(!isCollapsed)}
      >
        <span className="sr-only">{label}: </span>
        <span className="flex min-w-0 items-center gap-2">{summary}</span>
        <ChevronDown
          aria-hidden="true"
          className={cn(
            "size-4 shrink-0 text-control-light transition-transform duration-150 ease-out motion-reduce:transition-none",
            !isCollapsed && "rotate-180"
          )}
        />
      </Button>
      <div
        id={panelId}
        ref={panelRef}
        hidden={isCollapsed}
        inert={isCollapsed ? true : undefined}
      >
        <RadioGroup
          aria-label={label}
          value={value}
          disabled={disabled}
          onValueChange={(nextValue) => onValueChange(String(nextValue))}
          onKeyDown={(event) => {
            if (disabled) return;
            if (event.key === "Enter" || event.key === "Escape") {
              event.preventDefault();
              collapse();
            }
          }}
          className="grid w-full grid-cols-2 items-stretch gap-2 sm:grid-cols-[repeat(auto-fill,minmax(170px,1fr))]"
        >
          {children}
        </RadioGroup>
      </div>
    </TilePickerContext.Provider>
  );
}

export function TileRadioItem({
  value,
  icon,
  badge,
  children,
  disabled = false,
  className,
}: {
  value: string;
  icon?: ReactNode;
  badge?: ReactNode;
  children: ReactNode;
  disabled?: boolean;
  className?: string;
}) {
  const { disabled: groupDisabled, onItemClick } =
    useContext(TilePickerContext);
  const isDisabled = disabled || groupDisabled;

  return (
    <div
      className="contents"
      onClick={() => {
        if (!isDisabled) {
          onItemClick();
        }
      }}
    >
      <RadioGroupItem
        value={value}
        disabled={disabled}
        radioClassName="sr-only"
        contentClassName="flex min-w-0 flex-1 items-center gap-2"
        className={cn(
          "min-h-9 rounded-sm border border-control-border px-3 py-2 select-none",
          "transition-[color,background-color,border-color,box-shadow,transform] duration-150 ease-out motion-reduce:transition-none",
          "has-[>[role=radio]:not([data-checked]):not([data-disabled])]:hover:border-accent/50 has-[>[role=radio]:not([data-checked]):not([data-disabled])]:hover:bg-control-bg",
          "has-[>[role=radio]:not([data-disabled])]:active:scale-[0.98] motion-reduce:active:scale-100",
          "has-[[data-checked]]:border-accent has-[[data-checked]]:bg-accent/5 has-[[data-checked]]:ring-1 has-[[data-checked]]:ring-inset has-[[data-checked]]:ring-accent",
          "has-[[role=radio]:focus-visible]:ring-2 has-[[role=radio]:focus-visible]:ring-accent has-[[role=radio]:focus-visible]:ring-offset-2",
          "has-[>[data-disabled]]:opacity-50",
          className
        )}
      >
        {icon}
        <span className="truncate font-medium text-main">{children}</span>
        {badge && <span className="ml-auto shrink-0">{badge}</span>}
      </RadioGroupItem>
    </div>
  );
}
