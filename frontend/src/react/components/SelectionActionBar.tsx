import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import { LAYER_SURFACE_CLASS } from "@/react/components/ui/layer";
import { Separator } from "@/react/components/ui/separator";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";

export interface SelectionAction {
  /** Stable key for React reconciliation. */
  key: string;
  label: string;
  icon?: LucideIcon;
  onClick: () => void;
  disabled?: boolean;
  /**
   * When `disabled` and this is set, the reason is shown as a tooltip
   * on hover/focus.
   */
  disabledReason?: string;
  /** When true, the action is omitted from the bar entirely. */
  hidden?: boolean;
  /**
   * Visual tone. "destructive" applies a red-text/border override on top
   * of the outline variant. Default: "neutral".
   */
  tone?: "neutral" | "destructive";
}

export interface SelectionActionBarProps {
  /** Number of selected items. The bar renders only when count > 0. */
  count: number;
  /**
   * Pre-formatted label (e.g. "2 selected", "3 databases selected").
   * The call site owns i18n + pluralization.
   */
  label: string;
  /**
   * True when every visible item on the current page is selected. Drives
   * the checkbox's checked vs. indeterminate state.
   */
  allSelected: boolean;
  /**
   * Toggles between "select every visible item" and "clear selection".
   * Called by the leading checkbox.
   */
  onToggleSelectAll: () => void;
  /** Declarative actions. Rendered in order. Hidden actions are omitted. */
  actions?: SelectionAction[];
  /**
   * Custom action nodes rendered after `actions`. Used for actions that
   * require richer UI (e.g. InstancesPage's split-dropdown Sync).
   *
   * Style note: declarative `actions` get `rounded-full` automatically so
   * they match the pill surface. Custom buttons rendered here should also
   * use `rounded-full` (or a wrapper Button with `className="rounded-full"`)
   * to stay visually consistent.
   */
  children?: ReactNode;
}

const DESTRUCTIVE_TONE_CLASS =
  "border-error text-error hover:bg-error/10 hover:text-error focus-visible:ring-error";

export function SelectionActionBar({
  count,
  label,
  allSelected,
  onToggleSelectAll,
  actions,
  children,
}: SelectionActionBarProps) {
  if (count <= 0) return null;

  const visibleActions = (actions ?? []).filter((a) => !a.hidden);

  return (
    <div
      className={cn(
        "fixed bottom-6 left-1/2 -translate-x-1/2 max-w-[90vw]",
        "flex items-center gap-x-3 rounded-full bg-background border border-control-border shadow-lg",
        "px-4 py-2",
        LAYER_SURFACE_CLASS
      )}
    >
      {/* Leading cluster — never shrinks; stays visible on narrow viewports
          even when the actions group has to scroll horizontally. */}
      <Checkbox
        checked={allSelected ? true : "indeterminate"}
        onCheckedChange={() => onToggleSelectAll()}
      />
      <span className="shrink-0 text-sm font-medium text-control whitespace-nowrap">
        {label}
      </span>
      {visibleActions.length > 0 && (
        <Separator orientation="vertical" className="h-5 shrink-0" />
      )}
      {/* Actions cluster — `min-w-0` lets flex shrink the container so
          `overflow-x-auto` can actually scroll within the pill's max-width. */}
      <div className="flex items-center gap-x-3 min-w-0 overflow-x-auto">
        {visibleActions.map((action) => {
          const Icon = action.icon;
          const button = (
            <Button
              variant="outline"
              size="sm"
              disabled={action.disabled}
              onClick={action.onClick}
              className={cn(
                "rounded-full",
                action.tone === "destructive" && DESTRUCTIVE_TONE_CLASS
              )}
            >
              {Icon && <Icon className="size-4" aria-hidden />}
              {action.label}
            </Button>
          );
          if (action.disabled && action.disabledReason) {
            return (
              <Tooltip key={action.key} content={action.disabledReason}>
                {button}
              </Tooltip>
            );
          }
          return <div key={action.key}>{button}</div>;
        })}
        {children}
      </div>
    </div>
  );
}
