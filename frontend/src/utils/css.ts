import { create } from "@bufbuild/protobuf";
import { type Color, ColorSchema } from "@/types/proto-es/google/type/color_pb";

// Tailwind CSS default breakpoints
// https://tailwindcss.com/docs/responsive-design
export const TailwindBreakpoints = {
  sm: 640,
  md: 768,
  lg: 1024,
  xl: 1280,
  "2xl": 1536,
} as const;

export const rgbToHex = (r: number, g: number, b: number) => {
  const hex = [r, g, b]
    .map((decimal) => decimal.toString(16).padStart(2, "0"))
    .join("");
  return `#${hex}`;
};

export const hexToRgb = (hex: string) => {
  hex = hex.replace(/^#/, "");

  const hexValues: string[] = [];
  if (hex.length === 3) {
    hexValues.push(hex.charAt(0) + hex.charAt(0));
    hexValues.push(hex.charAt(1) + hex.charAt(1));
    hexValues.push(hex.charAt(2) + hex.charAt(2));
  } else {
    hexValues.push(hex.charAt(0) + hex.charAt(1));
    hexValues.push(hex.charAt(2) + hex.charAt(3));
    hexValues.push(hex.charAt(4) + hex.charAt(5));
  }
  return hexValues.map((str) => parseInt(str, 16));
};

const channelToByte = (channel: number): number =>
  Math.max(0, Math.min(255, Math.round(channel * 255)));

export const colorToHex = (color: Color): string =>
  rgbToHex(
    channelToByte(color.red),
    channelToByte(color.green),
    channelToByte(color.blue)
  );

export const colorToRgbString = (color: Color): string =>
  [color.red, color.green, color.blue].map(channelToByte).join(" ");

export const hexToColor = (hex: string): Color => {
  const [red, green, blue] = hexToRgb(hex);
  return create(ColorSchema, {
    red: red / 255,
    green: green / 255,
    blue: blue / 255,
  });
};

/**
 *
 * @param name css variable name including "--" prefix
 * @param convertColorFromTailwindToHex Typically, tailwindcss use "r g b" format colors to combine with opacity values.
 * @returns
 */
export const callCssVariable = (
  name: string,
  convertColorFromTailwindToHex = true
) => {
  const value = getComputedStyle(document.documentElement)
    .getPropertyValue(name)
    .trim();
  if (convertColorFromTailwindToHex) {
    const matches = value.match(/^(\d+) (\d+) (\d+)$/);
    if (matches) {
      const r = parseInt(matches[1], 10);
      const g = parseInt(matches[2], 10);
      const b = parseInt(matches[3], 10);
      return rgbToHex(r, g, b);
    }
  }

  return value;
};
