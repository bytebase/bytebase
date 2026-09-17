// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { DurationSchema, timestampFromMs } from "@bufbuild/protobuf/wkt";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  AccessGrant_Status,
  AccessGrantSchema,
} from "@/types/proto-es/v1/access_grant_service_pb";

vi.mock("@/lib/i18n", () => ({
  default: { language: "en-US", t: (key: string) => key },
}));

import {
  accessGrantStatusReading,
  getAccessGrantExpireTimeMs,
} from "./accessGrant";

const HOUR_MS = 3_600_000;
const DAY_MS = 86_400_000;

const grantWithDeadline = (status: AccessGrant_Status, deadlineMs: number) => {
  // Proto timestamps carry sub-millisecond nanos, where a boundary written by
  // hand a fraction late would otherwise hide.
  const expireTime = timestampFromMs(Math.floor(deadlineMs));
  expireTime.nanos += Math.round((deadlineMs % 1) * 1_000_000);
  return create(AccessGrantSchema, {
    status,
    expiration: { case: "expireTime", value: expireTime },
  });
};

const statusOf = (grant: ReturnType<typeof grantWithDeadline>) =>
  accessGrantStatusReading.read({ grant });
const statusChangesAt = (grant: ReturnType<typeof grantWithDeadline>) =>
  accessGrantStatusReading.nextChangeAt({ grant });

describe("access grant status over time", () => {
  const baseMs = new Date("2026-03-02T12:00:00Z").getTime();

  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(baseMs);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test.each([0, 0.5])(
    "an activated grant reads expired from the instant after its deadline (+%s ms)",
    (fractionMs) => {
      const deadlineMs = baseMs + HOUR_MS + fractionMs;
      const grant = grantWithDeadline(AccessGrant_Status.ACTIVE, deadlineMs);

      const changesAtMs = statusChangesAt(grant);
      expect(changesAtMs).toBe(baseMs + HOUR_MS + 1);

      vi.setSystemTime(changesAtMs - 1);
      expect(statusOf(grant)).toBe("ACTIVE");
      expect(statusChangesAt(grant)).toBe(changesAtMs);

      vi.setSystemTime(changesAtMs);
      expect(statusOf(grant)).toBe("EXPIRED");
    }
  );

  test.each([
    ["an expired", AccessGrant_Status.ACTIVE, baseMs - HOUR_MS],
    ["a pending", AccessGrant_Status.PENDING, baseMs + HOUR_MS],
    ["a revoked", AccessGrant_Status.REVOKED, baseMs + HOUR_MS],
  ])(
    "%s grant's status has nothing left to change",
    (_, status, deadlineMs) => {
      const grant = grantWithDeadline(status, deadlineMs);
      const status0 = statusOf(grant);

      expect(statusChangesAt(grant)).toBe(Number.POSITIVE_INFINITY);
      vi.setSystemTime(baseMs + 400 * DAY_MS);
      expect(statusOf(grant)).toBe(status0);
    }
  );

  test("a pending grant's ttl is a duration, not a deadline", () => {
    // Taken as `now + ttl`, it would name a different instant on every render.
    const grant = create(AccessGrantSchema, {
      status: AccessGrant_Status.PENDING,
      expiration: {
        case: "ttl",
        value: create(DurationSchema, { seconds: BigInt(HOUR_MS / 1000) }),
      },
    });

    expect(getAccessGrantExpireTimeMs(grant)).toBeUndefined();
    expect(accessGrantStatusReading.nextChangeAt({ grant })).toBe(
      Number.POSITIVE_INFINITY
    );
  });
});
