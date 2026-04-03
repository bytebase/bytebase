import { Switch as BaseSwitch } from "@base-ui/react/switch";
import { cn } from "@/react/lib/utils";

interface SwitchProps {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  disabled?: boolean;
  className?: string;
}

function Switch({
  checked,
  onCheckedChange,
  disabled,
  className,
}: SwitchProps) {
  return (
    <BaseSwitch.Root
      checked={checked}
      onCheckedChange={onCheckedChange}
      disabled={disabled}
      className={cn(
        "relative inline-flex h-5 w-9 cursor-pointer items-center rounded-full transition-colors",
        "bg-gray-300 data-[checked]:bg-blue-600",
        "focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2",
        "disabled:cursor-not-allowed disabled:opacity-50",
        className
      )}
    >
      <BaseSwitch.Thumb
        className={cn(
          "block h-4 w-4 rounded-full bg-white shadow-sm transition-transform",
          "translate-x-0.5 data-[checked]:translate-x-[18px]"
        )}
      />
    </BaseSwitch.Root>
  );
}

export { Switch };
export type { SwitchProps };
