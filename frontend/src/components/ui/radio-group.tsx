import { Radio } from "@base-ui/react/radio";
import { RadioGroup as BaseRadioGroup } from "@base-ui/react/radio-group";
import { cn } from "@/lib/utils";

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
  contentClassName,
  radioClassName,
  value,
  ...props
}: React.ComponentProps<typeof Radio.Root> & {
  children?: React.ReactNode;
  contentClassName?: string;
  radioClassName?: string;
}) {
  return (
    <label
      className={cn("flex items-center gap-x-2 cursor-pointer", className)}
    >
      <Radio.Root
        value={value}
        className={cn(
          "flex size-4 shrink-0 items-center justify-center rounded-full border border-control-border",
          "data-[checked]:border-[rgb(var(--color-accent))] data-[checked]:border-[5px]",
          "focus:outline-hidden focus-visible:ring-2 focus-visible:ring-[rgb(var(--color-accent))] focus-visible:ring-offset-2",
          "disabled:cursor-not-allowed disabled:opacity-50",
          radioClassName
        )}
        {...props}
      />
      {children && (
        <div className={cn("text-sm", contentClassName)}>{children}</div>
      )}
    </label>
  );
}

export { RadioGroup, RadioGroupItem };
