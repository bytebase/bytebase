import { Button as BaseButton } from "@base-ui/react/button";
import { cva, type VariantProps } from "class-variance-authority";
import type { ClassValue } from "clsx";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";

const buttonStyles = cva(
  "inline-flex items-center justify-center rounded-xs font-medium whitespace-nowrap cursor-pointer transition-colors focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
  {
    variants: {
      variant: {
        default: "",
        destructive: "",
      },
      appearance: {
        solid: "",
        outline: "border border-control-border bg-transparent",
        secondary: "",
        link: "underline-offset-4 hover:underline",
      },
      size: {
        // `default` is an alias for `md` — both render identically.
        default: "h-9 px-3 text-sm leading-5 gap-1.5",
        xs: "h-6 px-1.5 text-xs leading-4 gap-1",
        sm: "h-7 px-2 text-xs leading-4 gap-1",
        md: "h-9 px-3 text-sm leading-5 gap-1.5",
        lg: "h-10 px-4 text-sm leading-5 gap-1.5",
      },
    },
    compoundVariants: [
      {
        variant: "default",
        appearance: "solid",
        className: "bg-accent text-accent-text hover:bg-accent-hover",
      },
      {
        variant: "default",
        appearance: "outline",
        className: "text-control hover:bg-control-bg",
      },
      {
        variant: "default",
        appearance: "secondary",
        className: "text-control hover:bg-control-bg",
      },
      {
        variant: "default",
        appearance: "link",
        className: "text-accent",
      },
      {
        variant: "destructive",
        appearance: "solid",
        className: "bg-error text-white hover:bg-error-hover",
      },
      {
        variant: "destructive",
        appearance: "outline",
        className: "border-error text-error hover:bg-error hover:text-white",
      },
      {
        variant: "destructive",
        appearance: "secondary",
        className: "text-error hover:bg-error/10",
      },
      {
        variant: "destructive",
        appearance: "link",
        className: "text-error",
      },
    ],
    defaultVariants: {
      variant: "default",
      appearance: "solid",
      size: "default",
    },
  }
);

type ButtonIntent = NonNullable<VariantProps<typeof buttonStyles>["variant"]>;
type ButtonAppearance = NonNullable<
  VariantProps<typeof buttonStyles>["appearance"]
>;
type ButtonVariant =
  | ButtonIntent
  | "outline"
  | "ghost"
  | "ghost-destructive"
  | "link";
type ButtonSize = VariantProps<typeof buttonStyles>["size"];

type ButtonVariantProps = {
  variant?: ButtonVariant;
  appearance?: ButtonAppearance;
  size?: ButtonSize;
  class?: ClassValue;
  className?: ClassValue;
};

const normalizeButtonVariants = ({
  variant = "default",
  appearance,
}: Pick<ButtonVariantProps, "variant" | "appearance">): {
  variant: ButtonIntent;
  appearance: ButtonAppearance;
} => {
  switch (variant) {
    case "outline":
      return { variant: "default", appearance: appearance ?? "outline" };
    case "ghost":
      return { variant: "default", appearance: appearance ?? "secondary" };
    case "ghost-destructive":
      return { variant: "destructive", appearance: appearance ?? "outline" };
    case "link":
      return { variant: "default", appearance: appearance ?? "link" };
    default:
      return { variant, appearance: appearance ?? "solid" };
  }
};

function buttonVariants({
  class: classValue,
  className,
  variant,
  appearance,
  size = "default",
}: ButtonVariantProps = {}) {
  const normalized = normalizeButtonVariants({ variant, appearance });
  return cn(buttonStyles({ ...normalized, size }), classValue, className);
}

type ButtonProps = Omit<ComponentProps<"button">, "className"> &
  ButtonVariantProps;

function Button({
  class: classValue,
  className,
  variant,
  appearance,
  size = "default",
  ref,
  style,
  ...props
}: ButtonProps) {
  return (
    <BaseButton
      {...props}
      ref={ref}
      className={buttonVariants({
        class: classValue,
        className,
        size,
        variant,
        appearance,
      })}
      style={style}
    />
  );
}

export type { ButtonProps };
export { Button, buttonVariants };
