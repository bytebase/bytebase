import { ClientError } from "nice-grpc-web";
import { t } from "@/plugins/i18n";
import { extractGrpcErrorMessage } from "@/utils/grpcweb";
import { pushNotification } from "./notification";

export const useGracefulRequest = async <T>(
  fn: () => Promise<T>
): Promise<T> => {
  try {
    const result = await fn();
    return result;
  } catch (err) {
    const description = extractGrpcErrorMessage(err);
    if (err instanceof ClientError) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description,
      });
    }
    throw err;
  }
};
