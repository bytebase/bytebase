import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { useNow } from "@/hooks/useNow";
import { normalizeInstanceName } from "@/lib/resourceName";
import { useAppStore } from "@/stores/app";
import { getTimeForPbTimestampProtoEs } from "@/types";
import {
  formatAbsoluteDateTime,
  nextDaysLeftChangeAt,
  readDaysLeft,
} from "@/utils/datetime";

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

  const expireTimeMs = expireTime
    ? getTimeForPbTimestampProtoEs(expireTime)
    : undefined;
  useNow(
    expireTimeMs === undefined ? undefined : nextDaysLeftChangeAt(expireTimeMs)
  );

  if (expireTimeMs === undefined) {
    return null;
  }

  const formattedExpireTime = formatAbsoluteDateTime(expireTimeMs);
  const daysLeft = readDaysLeft(expireTimeMs);
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
