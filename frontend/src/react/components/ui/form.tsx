import * as stylex from "@stylexjs/stylex";
import type { ComponentProps } from "react";
import { cn } from "@/react/lib/utils";
import {
  formControlAffixStyle,
  formControlGroupStyle,
  formControlRowStyle,
  formDescriptionStyle,
  formErrorStyle,
  formFieldGroupStyle,
  formFieldRowStyle,
  formFieldStyle,
  formInlineAffixStyle,
  formLabelStyle,
  formMessageStyle,
} from "./styles.stylex";

function FormField({ className, ref, style, ...props }: ComponentProps<"div">) {
  const stylexProps = stylex.props(formFieldStyle());
  return (
    <div
      ref={ref}
      data-slot="form-field"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

function FormFieldGroup({
  className,
  ref,
  style,
  ...props
}: ComponentProps<"div">) {
  const stylexProps = stylex.props(formFieldGroupStyle());
  return (
    <div
      ref={ref}
      data-slot="form-field-group"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

function FormFieldRow({
  className,
  ref,
  style,
  ...props
}: ComponentProps<"div">) {
  const stylexProps = stylex.props(formFieldRowStyle());
  return (
    <div
      ref={ref}
      data-slot="form-field-row"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

function FormControlGroup({
  className,
  ref,
  style,
  ...props
}: ComponentProps<"div">) {
  const stylexProps = stylex.props(formControlGroupStyle());
  return (
    <div
      ref={ref}
      data-slot="form-control-group"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

function FormControlRow({
  className,
  ref,
  style,
  ...props
}: ComponentProps<"div">) {
  const stylexProps = stylex.props(formControlRowStyle());
  return (
    <div
      ref={ref}
      data-slot="form-control-row"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

function FormInlineAffix({
  className,
  ref,
  style,
  ...props
}: ComponentProps<"div">) {
  const stylexProps = stylex.props(formInlineAffixStyle());
  return (
    <div
      ref={ref}
      data-slot="form-inline-affix"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

function FormControlAffix({
  className,
  ref,
  style,
  ...props
}: ComponentProps<"span">) {
  const stylexProps = stylex.props(formControlAffixStyle());
  return (
    <span
      ref={ref}
      data-slot="form-control-affix"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

function FormLabel({
  className,
  htmlFor,
  ref,
  style,
  ...props
}: ComponentProps<"label">) {
  const stylexProps = stylex.props(formLabelStyle());
  return (
    <label
      ref={ref}
      data-slot="form-label"
      htmlFor={htmlFor}
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

function FormDescription({
  className,
  ref,
  style,
  ...props
}: ComponentProps<"p">) {
  const stylexProps = stylex.props(formDescriptionStyle());
  return (
    <p
      ref={ref}
      data-slot="form-description"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

function FormError({
  className,
  ref,
  role = "alert",
  style,
  ...props
}: ComponentProps<"p">) {
  const stylexProps = stylex.props(formErrorStyle());
  return (
    <p
      ref={ref}
      role={role}
      data-slot="form-error"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

function FormMessage({
  className,
  ref,
  role = "alert",
  style,
  ...props
}: ComponentProps<"p">) {
  const stylexProps = stylex.props(formMessageStyle());
  return (
    <p
      ref={ref}
      role={role}
      data-slot="form-message"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

export {
  FormControlAffix,
  FormControlGroup,
  FormControlRow,
  FormDescription,
  FormError,
  FormField,
  FormFieldGroup,
  FormFieldRow,
  FormInlineAffix,
  FormLabel,
  FormMessage,
};
