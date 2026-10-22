import { ExternalLink } from "lucide-react";
import { useTranslation } from "react-i18next";
import { FeatureBadge } from "@/components/FeatureBadge";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  FormControlRow,
  FormError,
  FormField,
  FormLabel,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { formatSharePointLocation } from "@/lib/sharepoint";
import type { SharePointLocation } from "@/types/proto-es/v1/project_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

const INPUT_ID = "project-export-destination";

interface Props {
  value: string;
  onChange: (value: string) => void;
  disabled: boolean;
  // Clearing needs no license, so it stays available after a license lapses.
  canClear: boolean;
  // The location the saved URL resolved to; shown only while the input still
  // holds that URL.
  savedUrl: string;
  savedLocation?: SharePointLocation;
  error?: string;
}

export function ProjectExportDestinationField({
  value,
  onChange,
  disabled,
  canClear,
  savedUrl,
  savedLocation,
  error,
}: Props) {
  const { t } = useTranslation();
  const showLocation =
    !error && !!savedLocation && !!savedUrl && value.trim() === savedUrl;

  return (
    <FormField
      title={
        <FormControlRow className="w-auto">
          <FormLabel htmlFor={INPUT_ID}>
            {t("project.settings.export-destination.self")}
          </FormLabel>
          <FeatureBadge feature={PlanFeature.FEATURE_SHAREPOINT_EXPORT} />
        </FormControlRow>
      }
      description={t("project.settings.export-destination.description")}
    >
      <FormControlRow>
        <Input
          id={INPUT_ID}
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
      {showLocation && savedLocation && (
        <div className="flex flex-col gap-y-1 text-sm">
          <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <Badge variant="success">
              {t("project.settings.export-destination.found")}
            </Badge>
            <a
              href={savedLocation.webUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex min-w-0 items-center gap-x-1 text-accent hover:underline"
            >
              <span className="truncate">
                {formatSharePointLocation(savedLocation)}
              </span>
              <ExternalLink className="size-3 shrink-0" />
            </a>
          </div>
          <span className="text-xs text-control-light">
            {t("project.settings.export-destination.found-description")}
          </span>
        </div>
      )}
    </FormField>
  );
}
