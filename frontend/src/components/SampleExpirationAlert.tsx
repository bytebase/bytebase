import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { normalizeInstanceName } from "@/lib/resourceName";
import { useAppStore } from "@/stores/app";
import { getTimeForPbTimestampProtoEs } from "@/types";
import { formatAbsoluteDateTime } from "@/utils";

const DAY_MS = 24 * 60 * 60 * 1000;

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

  if (!expireTime) {
    return null;
  }

  const expireTimeMs = getTimeForPbTimestampProtoEs(expireTime);
  const formattedExpireTime = formatAbsoluteDateTime(expireTimeMs);
  const remainingMs = expireTimeMs - Date.now();
  const description =
    remainingMs > 0
      ? t("instance.sample-expiration-future", {
          count: Math.ceil(remainingMs / DAY_MS),
          time: formattedExpireTime,
        })
      : t("instance.sample-expiration-expired", {
          time: formattedExpireTime,
        });

  return <Alert variant="warning" description={description} />;
}
