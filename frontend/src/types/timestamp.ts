import { Timestamp } from "./proto/google/protobuf/timestamp";
import type { Timestamp as TimestampProtoEs } from "@bufbuild/protobuf/wkt";

export const getTimeForPbTimestamp = (
  timestamp?: Timestamp,
  defaultValue = Date.now()
): number => {
  if (!timestamp) {
    return defaultValue;
  }
  return timestamp.seconds.toNumber() * 1000 + timestamp.nanos / 1000000;
};

export const getDateForPbTimestamp = (
  timestamp?: Timestamp
): Date | undefined => {
  if (!timestamp) {
    return undefined;
  }
  return new Date(getTimeForPbTimestamp(timestamp));
};

// Helper functions for proto-es timestamps (which use bigint for seconds)
export const getTimeForPbTimestampProtoEs = (
  timestamp?: TimestampProtoEs,
  defaultValue = Date.now()
): number => {
  if (!timestamp) {
    return defaultValue;
  }
  return Number(timestamp.seconds) * 1000 + timestamp.nanos / 1000000;
};

export const getDateForPbTimestampProtoEs = (
  timestamp?: TimestampProtoEs
): Date | undefined => {
  if (!timestamp) {
    return undefined;
  }
  return new Date(getTimeForPbTimestampProtoEs(timestamp));
};
