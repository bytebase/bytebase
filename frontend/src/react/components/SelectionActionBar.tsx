import { type LucideIcon, MoreHorizontal } from "lucide-react";
import { useSyncExternalStore } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
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
   * Override the default responsive cap (1 / 3 / 5 by viewport).
   * Useful when a call site has only 1–2 actions and never wants a
   * More menu — set this to a high number like 99.
   */
  maxVisibleActions?: number;
}

const MQ_SM = "(min-width: 640px)";
const MQ_LG = "(min-width: 1024px)";

function subscribeMatchMedia(query: string) {
  return (cb: () => void) => {
    if (typeof window === "undefined") return () => {};
    const mql = window.matchMedia(query);
    mql.addEventListener("change", cb);
    return () => mql.removeEventListener("change", cb);
  };
}

function matchesMediaQuery(query: string): boolean {
  if (typeof window === "undefined") return true;
  return window.matchMedia(query).matches;
}

function useSelectionMaxVisible(): number {
  const isLg = useSyncExternalStore(
    subscribeMatchMedia(MQ_LG),
    () => matchesMediaQuery(MQ_LG),
    () => true
  );
  const isSm = useSyncExternalStore(
    subscribeMatchMedia(MQ_SM),
    () => matchesMediaQuery(MQ_SM),
    () => true
  );
  if (isLg) return 5;
  if (isSm) return 3;
  return 1;
}

const DESTRUCTIVE_TONE_CLASS =
  "border-error text-error hover:bg-error/10 hover:text-error focus-visible:ring-error";

export function SelectionActionBar({
  count,
  label,
  allSelected,
  onToggleSelectAll,
  actions,
  maxVisibleActions,
}: SelectionActionBarProps) {
  const { t } = useTranslation();
  const defaultMaxVisible = useSelectionMaxVisible();
  const maxVisible = maxVisibleActions ?? defaultMaxVisible;

  if (count <= 0) return null;

  const visibleActions = (actions ?? []).filter((a) => !a.hidden);
  const inlineActions = visibleActions.slice(0, maxVisible);
  const overflowActions = visibleActions.slice(maxVisible);

  return (
    <div
      className={cn(
        "sticky bottom-6 mx-auto w-fit max-w-full",
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
      {(inlineActions.length > 0 || overflowActions.length > 0) && (
        <Separator orientation="vertical" className="h-5 shrink-0" />
      )}
      {/* Actions cluster — inline buttons + optional trailing More dropdown. */}
      <div className="flex items-center gap-x-3 shrink-0">
        {inlineActions.map((action) => {
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
        {overflowActions.length > 0 && (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  variant="outline"
                  size="sm"
                  className="rounded-full"
                  aria-label={t("common.more")}
                >
                  <MoreHorizontal className="size-4" aria-hidden />
                </Button>
              }
            />
            <DropdownMenuContent align="end">
              {overflowActions.map((action) => {
                const Icon = action.icon;
                const item = (
                  <DropdownMenuItem
                    key={action.key}
                    disabled={action.disabled}
                    onClick={action.onClick}
                    className={cn(
                      action.tone === "destructive" &&
                        "text-error data-highlighted:bg-error/10 data-highlighted:text-error"
                    )}
                  >
                    {Icon && <Icon className="size-4" aria-hidden />}
                    {action.label}
                  </DropdownMenuItem>
                );
                if (action.disabled && action.disabledReason) {
                  return (
                    <Tooltip key={action.key} content={action.disabledReason}>
                      <div>{item}</div>
                    </Tooltip>
                  );
                }
                return item;
              })}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
    </div>
  );
}
