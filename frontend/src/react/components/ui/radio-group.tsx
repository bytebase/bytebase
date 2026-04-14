import { Radio } from "@base-ui/react/radio";
import { RadioGroup as BaseRadioGroup } from "@base-ui/react/radio-group";
import { cn } from "@/react/lib/utils";

function RadioGroup({
  className,
  ...props
}: React.ComponentProps<typeof BaseRadioGroup>) {
  return (
    <BaseRadioGroup
      className={cn("flex items-center gap-x-6", className)}
      {...props}
    />
  );
}

function RadioGroupItem({
  children,
  className,
  value,
  ...props
}: React.ComponentProps<typeof Radio.Root> & { children?: React.ReactNode }) {
  return (
    <label
      className={cn("flex items-center gap-x-2 cursor-pointer", className)}
    >
      <Radio.Root
        value={value}
        className={cn(
          "flex size-4 items-center justify-center rounded-full border border-control-border",
          "data-[checked]:border-accent data-[checked]:border-[5px]",
          "focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2",
          "disabled:cursor-not-allowed disabled:opacity-50"
        )}
        {...props}
      />
      {children && <span className="text-sm">{children}</span>}
    </label>
  );
}

export { RadioGroup, RadioGroupItem };
