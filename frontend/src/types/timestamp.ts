import type { Timestamp as TimestampProtoEs } from "@bufbuild/protobuf/wkt";

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
