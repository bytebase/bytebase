import { Switch as BaseSwitch } from "@base-ui/react/switch";
import { cn } from "@/react/lib/utils";

interface SwitchProps {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  disabled?: boolean;
  className?: string;
  size?: "default" | "small";
}

function Switch({
  checked,
  onCheckedChange,
  disabled,
  className,
  size = "default",
}: SwitchProps) {
  const isSmall = size === "small";

  return (
    <BaseSwitch.Root
      checked={checked}
      onCheckedChange={onCheckedChange}
      disabled={disabled}
      className={cn(
        "relative inline-flex cursor-pointer items-center rounded-full transition-colors",
        isSmall ? "h-4 w-7" : "h-5 w-9",
        "bg-control-border data-[checked]:bg-accent",
        "focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2",
        "disabled:cursor-not-allowed disabled:opacity-50",
        className
      )}
    >
      <BaseSwitch.Thumb
        className={cn(
          "block rounded-full bg-background shadow-sm transition-transform",
          isSmall
            ? "h-3 w-3 translate-x-0.5 data-[checked]:translate-x-[12px]"
            : "h-4 w-4 translate-x-0.5 data-[checked]:translate-x-[18px]"
        )}
      />
    </BaseSwitch.Root>
  );
}

export { Switch };
export type { SwitchProps };
