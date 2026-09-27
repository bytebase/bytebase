import type { ComponentProps, ReactNode } from "react";
import { QuickLinkButton } from "@/components/ui/quick-link";

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
    <QuickLinkButton ref={ref} className={className} {...props} icon={icon}>
      {children}
    </QuickLinkButton>
  );
}
