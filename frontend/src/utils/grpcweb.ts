import { ClientError, ServerError, Status } from "nice-grpc-common";

export const getErrorCode = (error: unknown) => {
  if (error instanceof ClientError || error instanceof ServerError) {
    return error.code;
  }
  return Status.UNKNOWN;
};

export const extractGrpcErrorMessage = (err: unknown) => {
  const description = err instanceof ClientError ? err.details : String(err);
  return description;
};
