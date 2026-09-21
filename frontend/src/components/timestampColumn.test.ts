// @vitest-environment node
import { describe, expect, test } from "vitest";
import {
  TIMESTAMP_COLUMN_MIN_WIDTH,
  TIMESTAMP_COLUMN_WIDTH,
} from "./timestampColumn";

// jsdom has no font metrics, so these were measured in a browser: every IANA
// zone, the five shipped locales, and every month and local hour of a year,
// at 14px in the macOS system font, plus 32px of cell padding.
const MEASURED_FORM = {
  queue: 183,
  compact: 191,
  operational: 269,
  datetime: 291,
};
const MEASURED_DATE = 140;

const forms = Object.keys(MEASURED_FORM) as (keyof typeof MEASURED_FORM)[];

describe("timestamp column widths", () => {
  test.each(forms)(
    "open a %s column wide enough for its widest rendering",
    (form) => {
      expect(TIMESTAMP_COLUMN_WIDTH[form]).toBeGreaterThanOrEqual(
        MEASURED_FORM[form]
      );
    }
  );

  test("never narrow a column past the date", () => {
    expect(TIMESTAMP_COLUMN_MIN_WIDTH).toBeGreaterThanOrEqual(MEASURED_DATE);
  });

  test.each(forms)("let a reader narrow a %s column at all", (form) => {
    expect(TIMESTAMP_COLUMN_MIN_WIDTH).toBeLessThan(
      TIMESTAMP_COLUMN_WIDTH[form]
    );
  });
});
