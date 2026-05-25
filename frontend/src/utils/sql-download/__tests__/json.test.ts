import { describe, expect, it } from "vitest";
import { serializeJSON } from "../formats/json";
import { FIXTURES } from "./fixtures";
import { expectGolden } from "./helpers";

describe("serializeJSON byte-equal goldens", () => {
  for (const id of Object.keys(FIXTURES)) {
    if (id.endsWith("_skip_json")) {
      it.skip(`${id} (NaN/Inf — backend errors out)`, () => {});
      continue;
    }
    it(id, () => {
      expectGolden(serializeJSON(FIXTURES[id]), "json", `${id}.json`);
    });
  }
});

describe("serializeJSON throws on NaN/Inf", () => {
  it("rejects NaN (matches Go json.MarshalIndent UnsupportedValueError)", () => {
    expect(() => serializeJSON(FIXTURES.floats_special_skip_json)).toThrow(
      /SerializationFailed|NaN|Inf/
    );
  });
});
