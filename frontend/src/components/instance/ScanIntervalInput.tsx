import { create } from "@bufbuild/protobuf";
import type { Duration } from "@bufbuild/protobuf/wkt";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import { Info } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { useTranslation } from "react-i18next";
import { FeatureBadge } from "@/components/FeatureBadge";
import { Button } from "@/components/ui/button";
import { FormField, FormLabel } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Tooltip } from "@/components/ui/tooltip";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

const MIN_SCAN_MINUTES = 30;
const DEFAULT_CUSTOM_MINUTES = 24 * 60;

type ScanIntervalState = {
  mode: "DEFAULT" | "CUSTOM";
  minutes: number | undefined;
};

const extractState = (duration: Duration | undefined): ScanIntervalState => {
  if (!duration || Number(duration.seconds) === 0) {
    return { mode: "DEFAULT", minutes: undefined };
  }
  return {
    mode: "CUSTOM",
    minutes: Math.floor(Number(duration.seconds) / 60),
  };
};

export function ScanIntervalInput({
  scanInterval,
  allowEdit,
  onScanIntervalChange,
}: {
  scanInterval: Duration | undefined;
  allowEdit: boolean;
  onScanIntervalChange: (interval: Duration | undefined) => void;
}) {
  const { t } = useTranslation();
  const switchId = useId();

  const [mode, setMode] = useState<"DEFAULT" | "CUSTOM">(
    () => extractState(scanInterval).mode
  );
  const [minutes, setMinutes] = useState<number | undefined>(
    () => extractState(scanInterval).minutes
  );
  const [isValid, setIsValid] = useState(true);

  useEffect(() => {
    const s = extractState(scanInterval);
    setMode(s.mode);
    setMinutes(s.minutes);
    setIsValid(true);
  }, [scanInterval]);

  const handleModeChange = (targetMode: "DEFAULT" | "CUSTOM") => {
    if (targetMode === mode) return;
    setMode(targetMode);
    if (targetMode === "DEFAULT") {
      onScanIntervalChange(create(DurationSchema, { seconds: BigInt(0) }));
    } else {
      setMinutes(DEFAULT_CUSTOM_MINUTES);
      onScanIntervalChange(
        create(DurationSchema, { seconds: BigInt(DEFAULT_CUSTOM_MINUTES * 60) })
      );
    }
  };

  const handleMinuteChange = (value: string) => {
    const num = parseInt(value, 10);
    if (Number.isNaN(num)) {
      setMinutes(undefined);
      setIsValid(false);
      return;
    }
    setMinutes(num);
    if (num < MIN_SCAN_MINUTES) {
      setIsValid(false);
      return;
    }
    setIsValid(true);
    onScanIntervalChange(create(DurationSchema, { seconds: BigInt(num * 60) }));
  };

  return (
    <FormField
      title={
        <span className="inline-flex items-center gap-x-2">
          {t("instance.scan-interval.self")}
          <FeatureBadge
            feature={PlanFeature.FEATURE_CUSTOM_INSTANCE_SYNC_TIME}
          />
          <Tooltip
            content={t("instance.scan-interval.description")}
            popupClassName="max-w-96"
          >
            <Button
              type="button"
              appearance="link"
              size="xs"
              className="-ml-1 w-6 p-0"
              aria-label={t("instance.scan-interval.self")}
            >
              <Info className="size-3.5" />
            </Button>
          </Tooltip>
        </span>
      }
    >
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
        <Switch
          id={switchId}
          aria-label={t("instance.scan-interval.self")}
          checked={mode === "CUSTOM"}
          disabled={!allowEdit}
          onCheckedChange={(checked) =>
            handleModeChange(checked ? "CUSTOM" : "DEFAULT")
          }
        />
        <FormLabel htmlFor={switchId} className="font-normal!">
          {t("common.custom")}
        </FormLabel>
        {mode === "CUSTOM" && (
          <>
            <Input
              type="number"
              aria-label={t("common.minutes")}
              aria-invalid={!isValid || undefined}
              value={minutes ?? ""}
              className="w-20"
              placeholder={`>= ${MIN_SCAN_MINUTES}`}
              disabled={!allowEdit}
              onChange={(e) => handleMinuteChange(e.target.value)}
            />
            {!isValid ? (
              <span className="text-error text-sm">
                {t("instance.scan-interval.min-value", {
                  value: MIN_SCAN_MINUTES,
                })}
              </span>
            ) : (
              <span className="text-sm">{t("common.minutes")}</span>
            )}
          </>
        )}
      </div>
    </FormField>
  );
}
