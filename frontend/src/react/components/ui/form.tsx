import * as stylex from "@stylexjs/stylex";
import type { ComponentProps, ReactNode } from "react";
import { cn } from "@/react/lib/utils";
import {
  formControlGroupStyle,
  formControlRowStyle,
  formErrorStyle,
  formFieldDescriptionStyle,
  formFieldGroupStyle,
  formFieldHeaderStyle,
  formFieldStyle,
  formFieldTitleStyle,
  formLabelStyle,
  formSectionContentStyle,
  formSectionHeaderStyle,
  formSectionStyle,
  formSectionTitleStyle,
} from "./styles.stylex";

interface FormFieldProps extends Omit<ComponentProps<"div">, "title"> {
  title?: ReactNode;
  description?: ReactNode;
}

/**
 * Groups one logical form field with its optional title, description, control,
 * and validation feedback.
 *
 * @example
 * ```tsx
 * <FormField title={fieldTitle} description={fieldDescription}>
 *   <Input value={name} onChange={(event) => setName(event.target.value)} />
 *   {error && <FormError>{error}</FormError>}
 * </FormField>
 * ```
 */
function FormField({
  children,
  className,
  description,
  ref,
  style,
  title,
  ...props
}: FormFieldProps) {
  const fieldStylexProps = stylex.props(formFieldStyle());
  const headerStylexProps = stylex.props(formFieldHeaderStyle());
  const descriptionStylexProps = stylex.props(formFieldDescriptionStyle());
  const hasHeader = title !== undefined || description !== undefined;

  return (
    <div
      ref={ref}
      data-slot="form-field"
      className={cn(fieldStylexProps.className, className)}
      style={{ ...fieldStylexProps.style, ...style }}
      {...props}
    >
      {hasHeader && (
        <div
          data-slot="form-field-header"
          className={headerStylexProps.className}
          style={headerStylexProps.style}
        >
          {title !== undefined && <FormTitle>{title}</FormTitle>}
          {description !== undefined && (
            <div
              data-slot="form-field-description"
              className={descriptionStylexProps.className}
              style={descriptionStylexProps.style}
            >
              {description}
            </div>
          )}
        </div>
      )}
      {children}
    </div>
  );
}

/**
 * Renders the visual title for a form field. Prefer `FormField title` for
 * ordinary fields, and use `FormTitle` directly when the title row contains
 * custom layout or actions.
 *
 * @example
 * ```tsx
 * <FormField>
 *   <FormTitle>{fieldTitle}</FormTitle>
 *   <Input value={name} onChange={handleNameChange} />
 * </FormField>
 * ```
 */
function FormTitle({ className, ref, style, ...props }: ComponentProps<"div">) {
  const stylexProps = stylex.props(formFieldTitleStyle());
  return (
    <div
      ref={ref}
      data-slot="form-field-title"
      className={cn(stylexProps.className, className)}
      style={{ ...stylexProps.style, ...style }}
      {...props}
    />
  );
}

/**
 * Stacks related form fields inside a section or dialog.
 *
 * @example
 * ```tsx
 * <FormFieldGroup>
 *   <FormField title={nameTitle}>
 *     <Input value={name} onChange={handleNameChange} />
 *   </FormField>
 *   <FormField title={descriptionTitle}>
 *     <Input value={description} onChange={handleDescriptionChange} />
 *   </FormField>
 * </FormFieldGroup>
 * ```
 */
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

interface FormSectionProps extends Omit<ComponentProps<"section">, "title"> {
  title: ReactNode;
}

/**
 * Wraps a settings page section with a consistent section heading and content
 * column.
 *
 * @example
 * ```tsx
 * <FormSection id="general" title={sectionTitle}>
 *   <FormFieldGroup>
 *     <FormField title={fieldTitle}>
 *       <Input value={name} onChange={handleNameChange} />
 *     </FormField>
 *   </FormFieldGroup>
 * </FormSection>
 * ```
 */
function FormSection({
  children,
  className,
  ref,
  style,
  title,
  ...props
}: FormSectionProps) {
  const sectionStylexProps = stylex.props(formSectionStyle());
  const headerStylexProps = stylex.props(formSectionHeaderStyle());
  const titleStylexProps = stylex.props(formSectionTitleStyle());
  const contentStylexProps = stylex.props(formSectionContentStyle());

  return (
    <section
      ref={ref}
      data-slot="form-section"
      className={cn(sectionStylexProps.className, className)}
      style={{ ...sectionStylexProps.style, ...style }}
      {...props}
    >
      <div
        data-slot="form-section-header"
        className={headerStylexProps.className}
        style={headerStylexProps.style}
      >
        <div
          role="heading"
          aria-level={2}
          data-slot="form-section-title"
          className={titleStylexProps.className}
          style={titleStylexProps.style}
        >
          {title}
        </div>
      </div>
      <div
        data-slot="form-section-content"
        className={contentStylexProps.className}
        style={contentStylexProps.style}
      >
        {children}
      </div>
    </section>
  );
}

/**
 * Stacks multiple control rows that belong to the same field.
 *
 * @example
 * ```tsx
 * <FormControlGroup>
 *   <FormControlRow>
 *     <Input value={key} onChange={handleKeyChange} />
 *     <Input value={value} onChange={handleValueChange} />
 *   </FormControlRow>
 * </FormControlGroup>
 * ```
 */
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

/**
 * Aligns controls horizontally inside a field.
 *
 * @example
 * ```tsx
 * <FormControlRow>
 *   <Input value={parameter.name} onChange={handleNameChange} />
 *   <Input value={parameter.value} onChange={handleValueChange} />
 *   <Button type="button" onClick={handleRemove}>{removeLabel}</Button>
 * </FormControlRow>
 * ```
 */
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

/**
 * Renders a semantic label for a native or shared control.
 *
 * @example
 * ```tsx
 * <FormField>
 *   <FormLabel htmlFor="database-name">{databaseLabel}</FormLabel>
 *   <Input id="database-name" value={database} onChange={handleDatabaseChange} />
 * </FormField>
 * ```
 */
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

/**
 * Renders validation text for a field. Use it for blocking errors only.
 *
 * @example
 * ```tsx
 * <FormField title={fieldTitle}>
 *   <Input value={name} onChange={handleNameChange} />
 *   {nameError && <FormError>{nameError}</FormError>}
 * </FormField>
 * ```
 */
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

export {
  FormControlGroup,
  FormControlRow,
  FormError,
  FormField,
  FormFieldGroup,
  FormLabel,
  FormSection,
  FormTitle,
};
