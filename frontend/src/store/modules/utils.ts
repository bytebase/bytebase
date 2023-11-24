import { ClientError } from "nice-grpc-web";
import { t } from "@/plugins/i18n";
import { extractGrpcErrorMessage } from "@/utils/grpcweb";
import {
  useEnvironmentV1Store,
  useInstanceV1Store,
  usePolicyV1Store,
  useProjectV1Store,
  useRoleStore,
  useSettingV1Store,
  useUserStore,
} from ".";
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

export const initCommonModelStores = async () => {
  await Promise.all([
    useUserStore().fetchUserList(),
    useSettingV1Store().fetchSettingList(),
    useRoleStore().fetchRoleList(),
    useEnvironmentV1Store().fetchEnvironments(),
    useInstanceV1Store().fetchInstanceList(),
    useProjectV1Store().fetchProjectList(true),
    usePolicyV1Store().getOrFetchPolicyByName("policies/WORKSPACE_IAM"),
  ]);
};
