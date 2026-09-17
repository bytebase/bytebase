import type { Timestamp as TimestampProtoEs } from "@bufbuild/protobuf/wkt";

// Helper functions for proto-es timestamps (which use bigint for seconds)

/**
 * Milliseconds since the epoch. A timestamp that may be absent needs an explicit
 * fallback: substituting the current time silently would show "now" as when
 * something happened.
 */
export function getTimeForPbTimestampProtoEs(
  timestamp: TimestampProtoEs
): number;
export function getTimeForPbTimestampProtoEs(
  timestamp: TimestampProtoEs | undefined,
  defaultValue: number
): number;
export function getTimeForPbTimestampProtoEs(
  timestamp?: TimestampProtoEs,
  defaultValue = Date.now()
): number {
  if (!timestamp) {
    return defaultValue;
  }
  return Number(timestamp.seconds) * 1000 + timestamp.nanos / 1000000;
}

export const getDateForPbTimestampProtoEs = (
  timestamp?: TimestampProtoEs
): Date | undefined => {
  if (!timestamp) {
    return undefined;
  }
  return new Date(getTimeForPbTimestampProtoEs(timestamp));
};
