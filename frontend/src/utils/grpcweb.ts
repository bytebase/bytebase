import { ClientError, ServerError, Status } from "nice-grpc-common";

export const getErrorCode = (error: unknown) => {
  if (error instanceof ClientError || error instanceof ServerError) {
    return error.code;
  }
  return Status.UNKNOWN;
};
