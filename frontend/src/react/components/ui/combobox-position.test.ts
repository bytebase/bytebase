import { describe, expect, test } from "vitest";
import { getPortalDropdownStyle } from "./combobox-position";

describe("getPortalDropdownStyle", () => {
  test("opens upward when there is not enough room below", () => {
    expect(
      getPortalDropdownStyle(
        {
          top: 700,
          left: 80,
          width: 220,
          bottom: 736,
        },
        240,
        800
      )
    ).toEqual({
      position: "fixed",
      left: 80,
      width: 220,
      bottom: 104,
    });
  });

  test("keeps the dropdown below when there is enough room", () => {
    expect(
      getPortalDropdownStyle(
        {
          top: 120,
          left: 24,
          width: 180,
          bottom: 156,
        },
        240,
        800
      )
    ).toEqual({
      position: "fixed",
      left: 24,
      width: 180,
      top: 160,
    });
  });
});
