import { X } from "lucide-react";
import { useRef } from "react";
import { useTranslation } from "react-i18next";
import { FeatureBadge } from "@/components/FeatureBadge";
import { Button } from "@/components/ui/button";
import {
  FormControlGroup,
  FormControlRow,
  FormError,
  FormField,
  FormLabel,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

interface Props {
  value: string;
  onChange: (value: string) => void;
  disabled: boolean;
  // Clearing needs no license, so it stays available after a license lapses.
  canClear: boolean;
  error?: string;
}

// One row per destination provider, so another provider adds a row.
export function ProjectExportDestinationField({
  value,
  onChange,
  disabled,
  canClear,
  error,
}: Props) {
  const { t } = useTranslation();

  return (
    <FormField
      title={
        <span className="flex items-center gap-x-2">
          {t("project.settings.export-destination.self")}
          <FeatureBadge feature={PlanFeature.FEATURE_SHAREPOINT_EXPORT} />
        </span>
      }
      description={t("project.settings.export-destination.description")}
    >
      <FormControlGroup>
        <DestinationRow
          id="project-export-destination-sharepoint"
          label={t("export-data.destination.sharepoint")}
          placeholder={t(
            "project.settings.export-destination.sharepoint-placeholder"
          )}
          value={value}
          onChange={onChange}
          disabled={disabled}
          canClear={canClear}
          invalid={!!error}
        />
      </FormControlGroup>
      {error && <FormError>{error}</FormError>}
    </FormField>
  );
}

function DestinationRow({
  id,
  label,
  placeholder,
  value,
  onChange,
  disabled,
  canClear,
  invalid,
}: {
  id: string;
  label: string;
  placeholder: string;
  value: string;
  onChange: (value: string) => void;
  disabled: boolean;
  canClear: boolean;
  invalid: boolean;
}) {
  const { t } = useTranslation();
  const inputRef = useRef<HTMLInputElement>(null);
  const showClear = canClear && value !== "";

  return (
    <FormControlRow className="w-full max-w-2xl">
      <FormLabel htmlFor={id} className="w-24 shrink-0">
        {label}
      </FormLabel>
      <div className="relative min-w-0 flex-1">
        <Input
          ref={inputRef}
          id={id}
          type="url"
          className={showClear ? "pr-10" : undefined}
          value={value}
          disabled={disabled}
          aria-invalid={invalid}
          placeholder={placeholder}
          onChange={(event) => onChange(event.target.value)}
        />
        {showClear && (
          <Button
            type="button"
            appearance="secondary"
            size="sm"
            className="absolute right-1 top-1"
            aria-label={t("common.clear")}
            title={t("common.clear")}
            onClick={() => {
              onChange("");
              inputRef.current?.focus();
            }}
          >
            <X size={16} />
          </Button>
        )}
      </div>
    </FormControlRow>
  );
}
