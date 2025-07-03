import { Code, ConnectError } from "@connectrpc/connect";

export const getErrorCode = (error: unknown) => {
  if (error instanceof ConnectError) {
    return error.code;
  }
  return Code.Unknown;
};

export const extractGrpcErrorMessage = (err: unknown) => {
  if (err instanceof ConnectError) {
    return err.message;
  }
  return String(err);
};
