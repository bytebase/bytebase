import { Button as BaseButton } from "@base-ui/react/button";
import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps, ReactNode } from "react";
import { cn } from "@/react/lib/utils";

/**
 * Big square icon-over-label button used on the SQL Editor Welcome screen.
 * Mirrors frontend/src/views/sql-editor/EditorPanel/Welcome/Button.vue,
 * which wraps NButton at 7rem × 7rem with a vertical flex content slot.
 */
const welcomeButtonVariants = cva(
  "inline-flex flex-col items-center justify-center gap-2 min-w-28 h-28 px-4 rounded-xs text-sm font-medium cursor-pointer transition-colors focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
  {
    variants: {
      variant: {
        primary: "bg-accent text-accent-text hover:bg-accent-hover",
        secondary:
          "border border-control-border bg-transparent text-control hover:bg-control-bg",
      },
    },
    defaultVariants: {
      variant: "primary",
    },
  }
);

type WelcomeButtonProps = Omit<ComponentProps<"button">, "children"> &
  VariantProps<typeof welcomeButtonVariants> & {
    readonly icon: ReactNode;
    readonly children: ReactNode;
  };

export function WelcomeButton({
  icon,
  children,
  variant,
  className,
  ref,
  ...props
}: WelcomeButtonProps) {
  return (
    <BaseButton
      ref={ref}
      className={cn(welcomeButtonVariants({ variant, className }))}
      {...props}
    >
      <span className="flex items-center justify-center">{icon}</span>
      <span>{children}</span>
    </BaseButton>
  );
}
