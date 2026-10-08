import { Button as BaseButton } from "@base-ui/react/button";
import { cva } from "class-variance-authority";
import type { ComponentProps, ReactNode } from "react";
import { RouterLink, type RouterLinkProps } from "@/components/RouterLink";
import { cn } from "@/lib/utils";

const quickLinkVariants = cva(
  "flex h-auto items-center justify-center gap-x-2 cursor-pointer border border-control-border rounded-sm bg-background px-4 py-5 text-sm leading-5 font-normal text-main hover:bg-control-bg no-underline focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
);

type QuickLinkContentProps = {
  readonly icon: ReactNode;
  readonly children: ReactNode;
};

type QuickLinkButtonProps = Omit<ComponentProps<"button">, "children"> &
  QuickLinkContentProps;

export function QuickLinkButton({
  icon,
  children,
  className,
  ref,
  style,
  ...props
}: QuickLinkButtonProps) {
  return (
    <BaseButton
      {...props}
      ref={ref}
      className={cn(quickLinkVariants(), className)}
      style={style}
    >
      <span>{icon}</span>
      <span>{children}</span>
    </BaseButton>
  );
}

type QuickLinkLinkProps = Omit<RouterLinkProps, "children"> &
  QuickLinkContentProps;

export function QuickLinkLink({
  icon,
  children,
  className,
  style,
  ...props
}: QuickLinkLinkProps) {
  return (
    <RouterLink
      {...props}
      className={cn(quickLinkVariants(), className)}
      style={style}
    >
      <span>{icon}</span>
      <span>{children}</span>
    </RouterLink>
  );
}
