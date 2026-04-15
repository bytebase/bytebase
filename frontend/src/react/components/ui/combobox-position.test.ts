import { describe, expect, test } from "vitest";
import {
  getPortalDropdownStyle,
  isPortalDropdownStyleEqual,
  shouldIgnorePortalDropdownScroll,
} from "./combobox-position";

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

  test("ignores scroll events that originate from the dropdown itself", () => {
    const dropdown = document.createElement("div");
    const optionList = document.createElement("div");
    dropdown.appendChild(optionList);

    expect(shouldIgnorePortalDropdownScroll(optionList, dropdown)).toBe(true);
    expect(shouldIgnorePortalDropdownScroll(dropdown, dropdown)).toBe(true);
    expect(
      shouldIgnorePortalDropdownScroll(document.createElement("div"), dropdown)
    ).toBe(false);
    expect(shouldIgnorePortalDropdownScroll(null, dropdown)).toBe(false);
  });

  test("treats equivalent portal dropdown styles as unchanged", () => {
    expect(
      isPortalDropdownStyleEqual(
        {
          position: "fixed",
          left: 24,
          width: 180,
          top: 160,
        },
        {
          position: "fixed",
          left: 24,
          width: 180,
          top: 160,
        }
      )
    ).toBe(true);

    expect(
      isPortalDropdownStyleEqual(
        {
          position: "fixed",
          left: 24,
          width: 180,
          top: 160,
        },
        {
          position: "fixed",
          left: 24,
          width: 180,
          bottom: 104,
        }
      )
    ).toBe(false);
  });
});
