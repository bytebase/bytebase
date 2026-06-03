import { describe, expect, test } from "vitest";
import { buildAccessGrantFilter } from "./accessGrant";

const FIXED_NOW = new Date("2026-06-01T00:00:00Z");

describe("buildAccessGrantFilter", () => {
  test("empty filter returns empty string", () => {
    expect(buildAccessGrantFilter(undefined)).toBe("");
    expect(buildAccessGrantFilter({})).toBe("");
  });

  test("unmask=true emits literal CEL boolean", () => {
    expect(buildAccessGrantFilter({ unmask: true })).toBe("unmask == true");
  });

  test("unmask=false emits literal CEL boolean", () => {
    expect(buildAccessGrantFilter({ unmask: false })).toBe("unmask == false");
  });

  test("export=true emits literal CEL boolean", () => {
    expect(buildAccessGrantFilter({ export: true })).toBe("export == true");
  });

  test("export=false emits literal CEL boolean", () => {
    expect(buildAccessGrantFilter({ export: false })).toBe("export == false");
  });

  test("unmask and export combine with &&", () => {
    expect(buildAccessGrantFilter({ unmask: true, export: true })).toBe(
      "unmask == true && export == true"
    );
  });

  test("unmask combines with other filters in order", () => {
    expect(
      buildAccessGrantFilter({
        creator: "users/dev@example.com",
        unmask: true,
        export: false,
      })
    ).toBe(
      'creator == "users/dev@example.com" && unmask == true && export == false'
    );
  });

  // Sanity check: filter values use the literal token `true`/`false` so the
  // server-side filter parser treats them as CEL booleans, not strings. This
  // matches the backend's `store.GetListAccessGrantFilter` switch which
  // expects `value.(bool)` for these two fields. Quoting them would silently
  // change the SQL emitted (or trigger a "must be a boolean" error).
  test("never quotes the boolean literal", () => {
    expect(buildAccessGrantFilter({ unmask: true })).not.toContain('"true"');
    expect(buildAccessGrantFilter({ export: false })).not.toContain('"false"');
  });

  test("absent unmask/export fields produce no emission", () => {
    // Only target, no unmask/export.
    expect(
      buildAccessGrantFilter({ target: "instances/i/databases/d" }, FIXED_NOW)
    ).toBe('target == "instances/i/databases/d"');
  });
});
