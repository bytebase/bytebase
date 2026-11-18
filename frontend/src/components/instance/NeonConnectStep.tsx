import { useId } from "react";
import { useTranslation } from "react-i18next";
import { SecretInput } from "@/components/SecretInput";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  FormError,
  FormField,
  FormFieldGroup,
  FormLabel,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { NeonWarning } from "./NeonWarning";
import { formatClock, type NeonRequestFailure } from "./neon-import";

export type NeonConnectFailure = NeonRequestFailure | { kind: "key-empty" };

export function NeonConnectStep({
  apiKey,
  projectId,
  busy,
  failure,
  retryAt,
  destination,
  onApiKeyChange,
  onProjectIdChange,
  onSubmit,
}: {
  apiKey: string;
  projectId: string;
  busy: boolean;
  failure?: NeonConnectFailure;
  // When Neon allows the next request, while that is still ahead.
  retryAt?: number;
  destination: string;
  onApiKeyChange: (value: string) => void;
  onProjectIdChange: (value: string) => void;
  onSubmit: () => void;
}) {
  const { t } = useTranslation();
  const id = useId();
  const keyId = `${id}-key`;
  const keyErrorId = `${id}-key-error`;
  const projectIdId = `${id}-project`;
  const projectErrorId = `${id}-project-error`;

  const keyError =
    failure?.kind === "key-empty"
      ? t("instance.neon.connect.error.key-empty")
      : failure?.kind === "key-invalid"
        ? t("instance.neon.connect.error.key-invalid")
        : "";
  const projectError =
    failure?.kind === "project-id-required"
      ? t("instance.neon.connect.error.project-id-required")
      : failure?.kind === "project-id-invalid"
        ? t("instance.neon.connect.error.project-id-invalid")
        : failure?.kind === "project-id-unreadable"
          ? t("instance.neon.connect.error.project-id-unreadable")
          : "";

  return (
    <form
      className="flex w-full flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit();
      }}
    >
      {/* Lets Enter in either field submit; the visible button is in the page footer. */}
      <Button type="submit" hidden tabIndex={-1} aria-hidden="true" />
      <FormFieldGroup density="compact">
        <FormField
          title={
            <FormLabel htmlFor={keyId}>
              {t("instance.neon.connect.api-key")}
            </FormLabel>
          }
          description={t("instance.neon.connect.api-key-help")}
        >
          <SecretInput
            id={keyId}
            isCreating
            autoComplete="off"
            value={apiKey}
            disabled={busy}
            placeholder="napi_…"
            aria-invalid={!!keyError || undefined}
            aria-describedby={keyError ? keyErrorId : undefined}
            onValueChange={onApiKeyChange}
          />
          {keyError && <FormError id={keyErrorId}>{keyError}</FormError>}
        </FormField>
        <FormField
          title={
            <FormLabel htmlFor={projectIdId} className="flex-wrap">
              {t("instance.neon.connect.project-id")}{" "}
              <span className="font-normal text-control-light">
                {t("instance.neon.optional")}
              </span>
            </FormLabel>
          }
          description={t("instance.neon.connect.project-id-help")}
        >
          <Input
            id={projectIdId}
            value={projectId}
            disabled={busy}
            autoComplete="off"
            spellCheck={false}
            placeholder="autumn-disk-48433112"
            aria-invalid={!!projectError || undefined}
            aria-describedby={projectError ? projectErrorId : undefined}
            onChange={(event) => onProjectIdChange(event.target.value)}
          />
          {projectError && (
            <FormError id={projectErrorId}>{projectError}</FormError>
          )}
        </FormField>
      </FormFieldGroup>

      {failure?.kind === "rate-limited" && (
        <NeonWarning
          title={t("instance.neon.connect.error.rate-limited-title")}
          description={
            retryAt !== undefined
              ? t("instance.neon.connect.error.rate-limited-wait", {
                  time: formatClock(new Date(retryAt)),
                })
              : t("instance.neon.connect.error.rate-limited-now")
          }
        />
      )}
      {failure?.kind === "unavailable" && (
        <NeonWarning
          title={t("instance.neon.connect.error.unavailable-title")}
          description={t("instance.neon.connect.error.unavailable-body")}
        >
          <Button
            type="button"
            appearance="outline"
            size="sm"
            className="mt-2"
            disabled={busy}
            onClick={onSubmit}
          >
            {t("instance.neon.connect.try-again")}
          </Button>
        </NeonWarning>
      )}
      {failure?.kind === "permission-denied" && (
        <Alert
          variant="error"
          title={t("instance.neon.connect.error.permission-title")}
          description={t("instance.neon.connect.error.permission-body", {
            destination,
          })}
        />
      )}
      {(failure?.kind === "other" || failure?.kind === "token-invalid") && (
        <Alert
          variant="error"
          title={t("instance.neon.connect.error.other-title")}
          description={
            failure.message || t("instance.neon.connect.error.other-body")
          }
        />
      )}
    </form>
  );
}
