import { Timestamp } from "./proto/google/protobuf/timestamp";

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
