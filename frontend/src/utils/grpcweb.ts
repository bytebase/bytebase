import { ConnectError } from "@connectrpc/connect";
import { ClientError, ServerError, Status } from "nice-grpc-common";

export const getErrorCode = (error: unknown) => {
  if (error instanceof ClientError || error instanceof ServerError) {
    return error.code;
  }
  if (error instanceof ConnectError) {
    return error.code;
  }
  return Status.UNKNOWN;
};

export const extractGrpcErrorMessage = (err: unknown) => {
  if (err instanceof ClientError || err instanceof ConnectError) {
    return err.details;
  }
  return String(err);
};
