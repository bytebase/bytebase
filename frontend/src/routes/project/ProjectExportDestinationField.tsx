import { useTranslation } from "react-i18next";
import { FeatureBadge } from "@/components/FeatureBadge";
import { Button } from "@/components/ui/button";
import { FormControlRow, FormError, FormField } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

const INPUT_ID = "project-export-destination";

interface Props {
  value: string;
  onChange: (value: string) => void;
  disabled: boolean;
  // Clearing needs no license, so it stays available after a license lapses.
  canClear: boolean;
  error?: string;
}

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
      <FormControlRow>
        <Input
          id={INPUT_ID}
          aria-label={t("project.settings.export-destination.self")}
          className="w-full max-w-2xl"
          type="url"
          value={value}
          disabled={disabled}
          aria-invalid={!!error}
          placeholder={t("project.settings.export-destination.placeholder")}
          onChange={(event) => onChange(event.target.value)}
        />
        {value && canClear && (
          <Button appearance="outline" onClick={() => onChange("")}>
            {t("common.clear")}
          </Button>
        )}
      </FormControlRow>
      {error && <FormError>{error}</FormError>}
    </FormField>
  );
}
