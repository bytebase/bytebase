import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { FormError, FormField, FormLabel } from "@/components/ui/form";

export function StepHeading({
  question,
  description,
  optional,
}: {
  readonly question: React.ReactNode;
  readonly description?: React.ReactNode;
  readonly optional?: boolean;
}) {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col gap-y-1">
      <h3 className="text-base font-semibold text-main">
        {question}
        {optional && (
          <span className="ml-1 font-normal text-control-light">
            {t("workload-identity.optional")}
          </span>
        )}
      </h3>
      {description !== undefined && (
        <p className="text-sm text-control-light">{description}</p>
      )}
    </div>
  );
}

// One field: its label (with an optional Edit or Reset action), the control,
// the error, then the hint below the control.
export function Field({
  id,
  label,
  optional,
  action,
  hint,
  error,
  children,
}: {
  readonly id?: string;
  readonly label: React.ReactNode;
  readonly optional?: boolean;
  readonly action?: React.ReactNode;
  readonly hint?: React.ReactNode;
  readonly error?: string;
  readonly children: React.ReactNode;
}) {
  const { t } = useTranslation();
  return (
    <FormField>
      <div className="flex items-center justify-between gap-x-2">
        <FormLabel htmlFor={id}>
          {label}
          {optional && (
            <span className="font-normal text-control-light">
              {t("workload-identity.optional")}
            </span>
          )}
        </FormLabel>
        {action}
      </div>
      {children}
      {error && <FormError>{error}</FormError>}
      {hint !== undefined && (
        <p className="text-xs text-control-light">{hint}</p>
      )}
    </FormField>
  );
}

export function TextAction({
  onClick,
  children,
}: {
  readonly onClick: () => void;
  readonly children: React.ReactNode;
}) {
  return (
    <Button
      type="button"
      appearance="link"
      size="sm"
      className="h-auto p-0 text-sm font-medium"
      onClick={onClick}
    >
      {children}
    </Button>
  );
}

// Inline code in hints, filled in through <Trans components={{ code }} />.
export function InlineCode({
  children,
}: {
  readonly children?: React.ReactNode;
}) {
  return (
    <code className="rounded-xs bg-control-bg px-1 font-mono break-all text-main">
      {children}
    </code>
  );
}
