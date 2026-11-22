// @vitest-environment node
import { describe, expect, it } from "vitest";
import {
  CEL_ATTRIBUTE_RESOURCE_DATABASE,
  CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
} from "@/utils/cel-attributes";
import type { Factor } from "./factor";
import { getOperatorListByFactor } from "./operator";

describe("getOperatorListByFactor", () => {
  it.each<Factor>([
    CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
    CEL_ATTRIBUTE_RESOURCE_DATABASE,
  ])("supports string matching against %s", (factor) => {
    expect(getOperatorListByFactor(factor)).toEqual(
      expect.arrayContaining([
        "contains",
        "@not_contains",
        "matches",
        "startsWith",
        "endsWith",
      ])
    );
  });
});
