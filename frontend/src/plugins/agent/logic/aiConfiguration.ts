import { Code, ConnectError } from "@connectrpc/connect";

const AI_NOT_ENABLED_MESSAGE = "AI is not enabled";

export const isAgentAIConfigurationError = (error: unknown): boolean => {
  return (
    error instanceof ConnectError &&
    error.code === Code.FailedPrecondition &&
    (error.rawMessage === AI_NOT_ENABLED_MESSAGE ||
      error.message === AI_NOT_ENABLED_MESSAGE)
  );
};
