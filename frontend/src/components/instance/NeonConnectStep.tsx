import { Database, KeyRound } from "lucide-react";
import { useId } from "react";
import { useTranslation } from "react-i18next";
import { SecretInput } from "@/components/SecretInput";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  FormError,
  FormField,
  FormLabel,
  FormSection,
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
    <FormSection
      layout="stacked"
      padding="none"
      title={t("instance.neon.connect.section")}
    >
      <div className="flex flex-col gap-4">
        <form
          className="grid grid-cols-1 gap-6 md:grid-cols-2"
          onSubmit={(event) => {
            event.preventDefault();
            onSubmit();
          }}
        >
          {/* Lets Enter in either field submit; the visible button is in the
              page footer. */}
          <Button type="submit" hidden tabIndex={-1} aria-hidden="true" />
          <FormField
            layout="vertical"
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
            layout="vertical"
            title={
              <FormLabel htmlFor={projectIdId}>
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
        </form>

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

        <div className="grid grid-cols-1 rounded-sm border border-block-border sm:grid-cols-2">
          <div className="flex gap-3 border-b border-block-border p-3 sm:border-r sm:border-b-0">
            <KeyRound className="mt-0.5 size-4 shrink-0 text-control-light" />
            <div>
              <div className="text-sm font-medium text-main">
                {t("instance.neon.connect.not-saved-title")}
              </div>
              <p className="text-xs leading-4 text-control-light">
                {t("instance.neon.connect.not-saved-body")}
              </p>
            </div>
          </div>
          <div className="flex gap-3 p-3">
            <Database className="mt-0.5 size-4 shrink-0 text-control-light" />
            <div>
              <div className="text-sm font-medium text-main">
                {t("instance.neon.connect.credentials-saved-title")}
              </div>
              <p className="text-xs leading-4 text-control-light">
                {t("instance.neon.connect.credentials-saved-body")}
              </p>
            </div>
          </div>
        </div>
      </div>
    </FormSection>
  );
}
