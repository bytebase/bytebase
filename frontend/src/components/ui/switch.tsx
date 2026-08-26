import { Switch as BaseSwitch } from "@base-ui/react/switch";
import { cn } from "@/lib/utils";

type SwitchSize = "xs" | "sm" | "md" | "lg";

const SWITCH_SIZES: Record<SwitchSize, { root: string; thumb: string }> = {
  xs: {
    root: "h-3 w-5",
    thumb: "h-2 w-2 translate-x-0.5 data-[checked]:translate-x-[10px]",
  },
  sm: {
    root: "h-4 w-7",
    thumb: "h-3 w-3 translate-x-0.5 data-[checked]:translate-x-[14px]",
  },
  md: {
    root: "h-5 w-9",
    thumb: "h-4 w-4 translate-x-0.5 data-[checked]:translate-x-[18px]",
  },
  lg: {
    root: "h-6 w-11",
    thumb: "h-5 w-5 translate-x-0.5 data-[checked]:translate-x-[22px]",
  },
};

interface SwitchProps {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  disabled?: boolean;
  className?: string;
  size?: SwitchSize;
  /**
   * Accessible name for the control. A switch renders no text of its own, so
   * without this — or a `<label>` wrapping it — assistive technology announces
   * an unnamed switch and the person cannot tell which setting they are
   * toggling. Pass it whenever the visible title sits in a sibling element.
   *
   * `id` alone does not substitute: Base UI renders the switch as a `<span>`,
   * and `<label for>` cannot name a span.
   */
  "aria-label"?: string;
  "aria-labelledby"?: string;
  id?: string;
}

function Switch({
  checked,
  onCheckedChange,
  disabled,
  className,
  size = "md",
  id,
  "aria-label": ariaLabel,
  "aria-labelledby": ariaLabelledBy,
}: SwitchProps) {
  const sizeClasses = SWITCH_SIZES[size];

  return (
    <BaseSwitch.Root
      checked={checked}
      onCheckedChange={onCheckedChange}
      disabled={disabled}
      id={id}
      aria-label={ariaLabel}
      aria-labelledby={ariaLabelledBy}
      className={cn(
        "relative inline-flex cursor-pointer items-center rounded-full transition-colors",
        sizeClasses.root,
        // Off-track: a muted, text-derived neutral (`control` @ 30%) rather than
        // the border token — so a saturated custom `border` color doesn't turn
        // the off state into a colored fill. Checked uses the accent.
        "bg-control/30 data-[checked]:bg-accent",
        "focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2",
        "disabled:cursor-not-allowed disabled:opacity-50",
        className
      )}
    >
      <BaseSwitch.Thumb
        className={cn(
          "block rounded-full bg-background shadow-sm transition-transform",
          sizeClasses.thumb
        )}
      />
    </BaseSwitch.Root>
  );
}

export type { SwitchProps };
export { Switch };
