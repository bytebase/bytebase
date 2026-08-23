import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { useAppStore } from "@/stores/app";
import { getTimeForPbTimestampProtoEs } from "@/types";
import { extractInstanceResourceName, formatAbsoluteDateTime } from "@/utils";

const DAY_MS = 24 * 60 * 60 * 1000;

interface SampleExpirationAlertProps {
  instanceName: string;
}

export function SampleExpirationAlert({
  instanceName,
}: SampleExpirationAlertProps) {
  const { t } = useTranslation();
  const sample = useAppStore((state) => state.serverInfo?.sample);
  const expireTime = sample?.expireTime;
  const canonicalInstanceName = `instances/${extractInstanceResourceName(instanceName)}`;

  if (!expireTime || !sample.instances.includes(canonicalInstanceName)) {
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
