import type { TFunction } from "i18next";
import {
  type ComponentProps,
  createContext,
  type ReactNode,
  useContext,
  useId,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { FormError, FormField } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import type { ValidationErrors } from "./validation";

const validationMessages: Record<string, (t: TFunction) => string> = {
  "absolute-path": (t) => t("instance.validation.absolute-path"),
  "aws-credential": (t) => t("instance.validation.aws-credential"),
  "cloud-sql-host": (t) => t("instance.validation.cloud-sql-host"),
  "forbidden-parameter": (t) => t("instance.validation.forbidden-parameter"),
  "gcp-instance": (t) => t("instance.validation.gcp-instance"),
  "gcp-project": (t) => t("instance.validation.gcp-project"),
  "keytab-resupply": (t) => t("instance.validation.keytab-resupply"),
  "oracle-service": (t) => t("instance.validation.oracle-service"),
  "pem-certificate": (t) => t("instance.validation.pem-certificate"),
  "pem-key": (t) => t("instance.validation.pem-key"),
  required: (t) => t("instance.validation.required"),
  "specific-credential": (t) => t("instance.validation.specific-credential"),
  "tls-pair": (t) => t("instance.validation.tls-pair"),
  "tls-source-conflict": (t) => t("instance.validation.tls-source-conflict"),
};

const ErrorsContext = createContext<ValidationErrors>({});
const FieldContext = createContext<{
  invalid: boolean;
  description?: string;
  label?: string;
}>({ invalid: false });

export function ValidationProvider({
  errors,
  children,
  className,
}: {
  errors: ValidationErrors;
  children: ReactNode;
  className?: string;
}) {
  return (
    <ErrorsContext.Provider value={errors}>
      {className ? <div className={className}>{children}</div> : children}
    </ErrorsContext.Provider>
  );
}

export function ValidationField({
  validationField,
  showErrors = false,
  children,
  ...props
}: ComponentProps<typeof FormField> & {
  validationField?: string | string[];
  showErrors?: boolean;
}) {
  const { t } = useTranslation();
  const errors = useContext(ErrorsContext);
  const [touched, setTouched] = useState(false);
  const id = useId();
  const fields =
    typeof validationField === "string"
      ? [validationField]
      : (validationField ?? []);
  const messages = [
    ...new Set(
      fields.flatMap((field) => (errors[field] ? [errors[field]] : []))
    ),
  ];
  const invalid = (showErrors || touched) && messages.length > 0;
  return (
    <FieldContext.Provider
      value={{
        invalid,
        description: invalid ? id : undefined,
        label: typeof props.title === "string" ? props.title : undefined,
      }}
    >
      <FormField
        {...props}
        data-invalid={invalid || undefined}
        onBlurCapture={(event) => {
          setTouched(true);
          props.onBlurCapture?.(event);
        }}
      >
        {children}
        {invalid && (
          <FormError id={id} aria-live="polite">
            {messages
              .map((message) => validationMessages[message](t))
              .join(" ")}
          </FormError>
        )}
      </FormField>
    </FieldContext.Provider>
  );
}

export function ValidationInput(props: ComponentProps<typeof Input>) {
  const field = useContext(FieldContext);
  return (
    <Input
      {...props}
      aria-label={props["aria-label"] ?? field.label}
      aria-invalid={field.invalid || props["aria-invalid"]}
      aria-describedby={
        [props["aria-describedby"], field.description]
          .filter(Boolean)
          .join(" ") || undefined
      }
    />
  );
}

export function ValidationTextarea(props: ComponentProps<typeof Textarea>) {
  const field = useContext(FieldContext);
  return (
    <Textarea
      {...props}
      aria-label={props["aria-label"] ?? field.label}
      aria-invalid={field.invalid || props["aria-invalid"]}
      aria-describedby={
        [props["aria-describedby"], field.description]
          .filter(Boolean)
          .join(" ") || undefined
      }
    />
  );
}
