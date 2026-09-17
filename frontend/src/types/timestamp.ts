import type { Timestamp as TimestampProtoEs } from "@bufbuild/protobuf/wkt";

// Helper functions for proto-es timestamps (which use bigint for seconds)

/**
 * Milliseconds since the epoch. An absent timestamp has no reading: without a
 * fallback the result is `undefined`, which the type makes the caller handle.
 * Substituting a time instead -- the current one, or the epoch -- would show a
 * plausible-looking moment that never happened.
 */
export function getTimeForPbTimestampProtoEs(
  timestamp: TimestampProtoEs
): number;
export function getTimeForPbTimestampProtoEs(
  timestamp: TimestampProtoEs | undefined,
  defaultValue: number
): number;
export function getTimeForPbTimestampProtoEs(
  timestamp: TimestampProtoEs | undefined
): number | undefined;
export function getTimeForPbTimestampProtoEs(
  timestamp?: TimestampProtoEs,
  defaultValue?: number
): number | undefined {
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
