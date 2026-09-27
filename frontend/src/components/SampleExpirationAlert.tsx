import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { useTimeReading } from "@/hooks/useTimeReading";
import { normalizeInstanceName } from "@/lib/resourceName";
import { useAppStore } from "@/stores/app";
import { getTimeForPbTimestampProtoEs } from "@/types";
import { daysLeftReading, formatAbsoluteDateTime } from "@/utils/datetime";

type SampleExpirationAlertProps = Readonly<{
  instanceName: string;
}>;

export function SampleExpirationAlert({
  instanceName,
}: SampleExpirationAlertProps) {
  const { t } = useTranslation();
  const sample = useAppStore((state) => state.serverInfo?.sample);
  const canonicalInstanceName = normalizeInstanceName(instanceName);
  const expireTime = sample?.instances.find(
    ({ instance }) => instance === canonicalInstanceName
  )?.expireTime;

  const expireTimeMs = getTimeForPbTimestampProtoEs(expireTime);
  const daysLeft = useTimeReading(daysLeftReading, expireTimeMs);

  if (expireTimeMs === undefined || daysLeft === undefined) {
    return null;
  }

  const formattedExpireTime = formatAbsoluteDateTime(expireTimeMs);
  const description =
    daysLeft.kind === "passed"
      ? t("instance.sample-expiration-expired", {
          time: formattedExpireTime,
        })
      : t("instance.sample-expiration-future", {
          // The last day counts as one.
          count: daysLeft.kind === "days" ? daysLeft.days : 1,
          time: formattedExpireTime,
        });

  return <Alert variant="warning" description={description} />;
}
