import { pullAt } from "lodash-es";
import { computed, type Raw, ref, unref, watchEffect } from "vue";
import type { MaybeRef } from "@/types";
import type { VueClass } from "./types";

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

export const useClassStack = () => {
  type StackItem = {
    id: number;
    classes: VueClass;
  };
  const context = {
    serial: 0,
  };
  const stack = ref<Raw<StackItem>[]>([]);
  const override = (classes: MaybeRef<VueClass>) => {
    watchEffect((cleanup) => {
      const id = context.serial++;
      stack.value.push({
        id,
        classes: unref(classes),
      });

      cleanup(() => {
        const index = stack.value.findIndex((item) => item.id === id);
        if (index >= 0) {
          pullAt(stack.value, index);
        }
      });
    });
  };

  const classes = computed(() => {
    return stack.value.map((item) => item.classes);
  });

  return { override, classes };
};
