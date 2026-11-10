import { Code, ConnectError } from "@connectrpc/connect";
import type { Setting } from "@/types/proto-es/v1/setting_service_pb";

const AI_NOT_ENABLED_MESSAGE = "AI is not enabled";

export const isAgentAIConfigurationError = (error: unknown): boolean => {
  return (
    error instanceof ConnectError &&
    error.code === Code.FailedPrecondition &&
    (error.rawMessage === AI_NOT_ENABLED_MESSAGE ||
      error.message === AI_NOT_ENABLED_MESSAGE)
  );
};

export const getAgentAIConfigurationEnabled = (
  setting: Setting | undefined
): boolean | undefined => {
  if (!setting) return undefined;
  if (setting.value?.value.case !== "ai") return false;
  return setting.value.value.value.enabled;
};
