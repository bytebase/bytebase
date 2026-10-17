import * as stylex from "@stylexjs/stylex";
import {
  Children,
  type ComponentProps,
  createContext,
  forwardRef,
  type ReactNode,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { cn } from "@/lib/utils";
import {
  denseFormContentStyle,
  denseFormHeaderStyle,
  denseFormSectionStyle,
  denseFormTitleStyle,
  formControlGroupStyle,
  formControlRowStyle,
  formErrorStyle,
  formFieldControlStyle,
  formFieldDescriptionStyle,
  formFieldGroupStyle,
  formFieldHeaderStyle,
  formFieldHorizontalStyle,
  formFieldStyle,
  formFieldTitleStyle,
  formLabelStyle,
  formSectionContentStyle,
  formSectionDescriptionStyle,
  formSectionHeaderStyle,
  formSectionNoPaddingStyle,
  formSectionStyle,
  formSectionTitleStyle,
  singleColumnFormContentStyle,
  singleColumnFormHeaderStyle,
  singleColumnFormSectionStyle,
} from "./styles.stylex";

type FormLayout = "vertical" | "horizontal";
type FormDensity = "default" | "compact";

interface FormLayoutContextValue {
  layout: FormLayout;
  density: FormDensity;
}

const FormLayoutContext = createContext<FormLayoutContextValue>({
  layout: "vertical",
  density: "default",
});

interface ResponsiveFormLayoutProps extends Omit<ComponentProps<"div">, "ref"> {
  /**
   * Uses label/control rows when this layout region has enough room. This is
   * intentionally based on the region width, so a docked side panel does not
   * leave fields stranded in a viewport-sized two-column layout.
   * Dependent groups use `inherit` to avoid making another breakpoint decision.
   */
  layout?: "responsive" | "inherit" | FormLayout;
  /**
   * Keeps dependent fields at label hierarchy when they stack inside a dense
   * form control area.
   */
  density?: FormDensity;
  horizontalMinWidth?: number;
}

const ResponsiveFormLayout = forwardRef<
  HTMLDivElement,
  ResponsiveFormLayoutProps
>(function ResponsiveFormLayout(
  {
    children,
    density: densityOverride,
    horizontalMinWidth = 560,
    layout = "responsive",
    ...props
  },
  ref
) {
  const inherited = useContext(FormLayoutContext);
  const density = densityOverride ?? inherited.density;
  const layoutRef = useRef<HTMLDivElement>(null);
  const [isHorizontal, setIsHorizontal] = useState(
    layout === "horizontal" ||
      (layout === "inherit" && inherited.layout === "horizontal")
  );

  useEffect(() => {
    if (layout === "inherit") {
      setIsHorizontal(inherited.layout === "horizontal");
      return;
    }
    if (layout !== "responsive") {
      setIsHorizontal(layout === "horizontal");
      return;
    }
    const element = layoutRef.current;
    if (!element) return;
    const update = (width: number) =>
      setIsHorizontal(width >= horizontalMinWidth);
    update(element.clientWidth);
    const observer = new ResizeObserver((entries) => {
      update(entries[0]?.contentRect.width ?? 0);
    });
    observer.observe(element);
    return () => observer.disconnect();
  }, [horizontalMinWidth, inherited.layout, layout]);

  return (
    <FormLayoutContext.Provider
      value={{
        layout: isHorizontal ? "horizontal" : "vertical",
        density,
      }}
    >
      <div
        ref={(element) => {
          layoutRef.current = element;
          if (typeof ref === "function") {
            ref(element);
          } else if (ref) {
            ref.current = element;
          }
        }}
        {...props}
      >
        {children}
      </div>
    </FormLayoutContext.Provider>
  );
});

interface FormFieldProps extends Omit<ComponentProps<"div">, "title"> {
  title?: ReactNode;
  description?: ReactNode;
  layout?: FormLayout;
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
  layout: layoutOverride,
  ...props
}: FormFieldProps) {
  const inherited = useContext(FormLayoutContext);
  const layout = layoutOverride ?? inherited.layout;
  const fieldStylexProps = stylex.props(
    formFieldStyle(),
    (title !== undefined ||
      Children.toArray(children).some(
        (child) =>
          typeof child === "object" &&
          child !== null &&
          "type" in child &&
          child.type === FormLabel
      )) &&
      layout === "horizontal" &&
      formFieldHorizontalStyle()
  );
  const headerStylexProps = stylex.props(formFieldHeaderStyle());
  const descriptionStylexProps = stylex.props(formFieldDescriptionStyle());
  const controlStylexProps = stylex.props(formFieldControlStyle());
  const childArray = layout === "horizontal" ? Children.toArray(children) : [];
  const firstChild = childArray[0];
  const label =
    layout === "horizontal" &&
    title === undefined &&
    firstChild &&
    typeof firstChild === "object" &&
    "type" in firstChild &&
    firstChild.type === FormLabel
      ? firstChild
      : undefined;
  const controlChildren = label ? childArray.slice(1) : childArray;
  const hasHeader = title !== undefined || description !== undefined || label;

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
          {title !== undefined ? <FormTitle>{title}</FormTitle> : label}
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
      {layout === "horizontal" ? (
        <div
          data-slot="form-field-control"
          className={controlStylexProps.className}
          style={controlStylexProps.style}
        >
          <FormLayoutContext.Provider
            value={{ ...inherited, layout: "vertical" }}
          >
            {controlChildren}
          </FormLayoutContext.Provider>
        </div>
      ) : (
        children
      )}
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
  const { density, layout } = useContext(FormLayoutContext);
  const stylexProps = stylex.props(
    layout === "horizontal" || density === "compact"
      ? formLabelStyle()
      : formFieldTitleStyle()
  );
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
  density = "default",
  className,
  ref,
  style,
  ...props
}: ComponentProps<"div"> & { density?: "default" | "compact" }) {
  const stylexProps = stylex.props(formFieldGroupStyle(density));
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
  description?: ReactNode;
  /** Removes the section's outer vertical padding when its page layout owns spacing. */
  padding?: "default" | "none";
  /**
   * - `default`: on large screens the header takes a 25% column beside the
   *   content; on smaller screens it stacks above.
   * - `single-column`: the header stays above the content at every width,
   *   with the same page section title, for a page that reads as one column.
   * - `stacked`: the dense variant with a 16px title and 16px padding, for
   *   sections inside a dense form such as the instance form.
   */
  layout?: "default" | "single-column" | "stacked";
}

const formSectionLayoutStyles = {
  default: {
    section: formSectionStyle,
    header: formSectionHeaderStyle,
    title: formSectionTitleStyle,
    content: formSectionContentStyle,
  },
  "single-column": {
    section: singleColumnFormSectionStyle,
    header: singleColumnFormHeaderStyle,
    title: formSectionTitleStyle,
    content: singleColumnFormContentStyle,
  },
  stacked: {
    section: denseFormSectionStyle,
    header: denseFormHeaderStyle,
    title: denseFormTitleStyle,
    content: denseFormContentStyle,
  },
};

/**
 * Wraps a settings page section with a consistent section heading and content
 * column.
 *
 * @example
 * ```tsx
 * <FormSection id="general" title={sectionTitle} description={summary}>
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
  description,
  layout = "default",
  padding = "default",
  ...props
}: FormSectionProps) {
  const styles = formSectionLayoutStyles[layout];
  const sectionStylexProps = stylex.props(
    styles.section(),
    padding === "none" && formSectionNoPaddingStyle()
  );
  const headerStylexProps = stylex.props(styles.header());
  const titleStylexProps = stylex.props(styles.title());
  const descriptionStylexProps = stylex.props(formSectionDescriptionStyle());
  const contentStylexProps = stylex.props(styles.content());

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
        {description !== undefined && (
          <div
            data-slot="form-section-description"
            className={descriptionStylexProps.className}
            style={descriptionStylexProps.style}
          >
            {description}
          </div>
        )}
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
  ResponsiveFormLayout,
};
