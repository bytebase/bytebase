import { describe, it } from "vitest";
import { serializeCSV } from "../formats/csv";
import { FIXTURES } from "./fixtures";
import { expectGolden } from "./helpers";

describe("serializeCSV byte-equal goldens", () => {
  for (const id of Object.keys(FIXTURES)) {
    it(id, () => {
      expectGolden(serializeCSV(FIXTURES[id]), "csv", `${id}.csv`);
    });
  }
});
