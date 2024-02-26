import { describe, expect, test } from "vitest";
import { mergedLocalMessage } from "./i18n-messages";

describe("Test i18n messages", () => {
  for (const keyA of Object.keys(mergedLocalMessage)) {
    for (const keyB of Object.keys(mergedLocalMessage)) {
      if (keyA === keyB) {
        continue;
      }

      test(`i18n message for ${keyA} and ${keyB}`, () => {
        const missMatchKey = compareMessages(
          mergedLocalMessage[keyA],
          mergedLocalMessage[keyB]
        );
        let message = "";
        if (missMatchKey !== "") {
          message = `${missMatchKey} not match in ${keyA} and ${keyB}`;
          console.error(message);
        }
        expect(missMatchKey).toBe("");
      });
    }
  }
});

const compareMessages = (
  localA: { [k: string]: any },
  localB: { [k: string]: any }
): string => {
  for (const [key, val] of Object.entries(localA)) {
    if (!localB[key]) {
      return key;
    }
    if (typeof val === "object") {
      if (typeof localB[key] !== "object") {
        return key;
      }
      const missMatch = compareMessages(val, localB[key]);
      if (missMatch) {
        return `${key}.${missMatch}`;
      }
    }
  }

  return "";
};
