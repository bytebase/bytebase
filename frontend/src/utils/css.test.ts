import { describe, expect, test } from "vitest";
import { colorToHex, colorToRgbString, hexToColor } from "./css";

describe("google.type.Color helpers", () => {
  test("converts #rrggbb hex to google.type.Color channels", () => {
    expect(hexToColor("#336699")).toMatchObject({
      red: 0.2,
      green: 0.4,
      blue: 0.6,
    });
  });

  test("converts google.type.Color to #rrggbb hex", () => {
    expect(colorToHex(hexToColor("#336699"))).toBe("#336699");
  });

  test("converts google.type.Color to CSS rgb channel string", () => {
    expect(colorToRgbString(hexToColor("#336699"))).toBe("51 102 153");
  });
});
