import { Button as BaseButton } from "@base-ui/react/button";
import type { ComponentProps, ReactNode } from "react";
import { QUICK_LINK_TILE_CLASS } from "@/components/ui/quick-link";
import { cn } from "@/lib/utils";

type WelcomeButtonProps = Omit<ComponentProps<"button">, "children"> & {
  readonly icon: ReactNode;
  readonly children: ReactNode;
};

export function WelcomeButton({
  icon,
  children,
  className,
  ref,
  ...props
}: WelcomeButtonProps) {
  return (
    <BaseButton
      ref={ref}
      className={cn(
        QUICK_LINK_TILE_CLASS,
        "focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
        className
      )}
      {...props}
    >
      <span>{icon}</span>
      <span>{children}</span>
    </BaseButton>
  );
}
