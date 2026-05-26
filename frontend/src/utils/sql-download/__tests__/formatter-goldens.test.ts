import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { describe, expect, it } from "vitest";
import {
  RowValue_TimestampSchema,
  RowValue_TimestampTZSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { formatFloat32, formatTimestamp, formatTimestampTZ } from "../value";

const here = dirname(fileURLToPath(import.meta.url));
const formattersDir = resolve(here, "goldens", "formatters");

function readTSV(filename: string): string[][] {
  const text = readFileSync(resolve(formattersDir, filename), "utf-8");
  return text
    .split("\n")
    .map((l) => l.trim())
    .filter((l) => l.length > 0 && !l.startsWith("#"))
    .map((l) => l.split("\t"));
}

function float32FromHex(hex: string): number {
  if (hex.length !== 8) throw new Error(`bad hex ${hex}`);
  const u = Number.parseInt(hex, 16);
  const buf = new ArrayBuffer(4);
  new DataView(buf).setUint32(0, u, false);
  return new DataView(buf).getFloat32(0, false);
}

describe("formatter cross-side goldens", () => {
  it("formatFloat32 matches Go strconv.FormatFloat(_, 'f', -1, 32)", () => {
    for (const [hex, expected] of readTSV("float32.tsv")) {
      const got = formatFloat32(float32FromHex(hex));
      expect(got, `bits=${hex}`).toBe(expected);
    }
  });

  it('formatTimestamp matches Go time.Format("2006-01-02 15:04:05.000000")', () => {
    for (const [secStr, nanosStr, expected] of readTSV("timestamp.tsv")) {
      const ts = create(RowValue_TimestampSchema, {
        googleTimestamp: create(TimestampSchema, {
          seconds: BigInt(secStr),
          nanos: Number.parseInt(nanosStr, 10),
        }),
      });
      const got = formatTimestamp(ts);
      expect(got, `seconds=${secStr} nanos=${nanosStr}`).toBe(expected);
    }
  });

  it("formatTimestampTZ matches Go time.RFC3339Nano in FixedZone", () => {
    for (const [secStr, nanosStr, zoneRaw, offsetStr, expected] of readTSV(
      "timestamptz.tsv"
    )) {
      const zone = zoneRaw === "-" ? "" : zoneRaw;
      const tz = create(RowValue_TimestampTZSchema, {
        googleTimestamp: create(TimestampSchema, {
          seconds: BigInt(secStr),
          nanos: Number.parseInt(nanosStr, 10),
        }),
        zone,
        offset: Number.parseInt(offsetStr, 10),
      });
      const got = formatTimestampTZ(tz);
      expect(
        got,
        `seconds=${secStr} nanos=${nanosStr} zone=${zoneRaw} offset=${offsetStr}`
      ).toBe(expected);
    }
  });
});
