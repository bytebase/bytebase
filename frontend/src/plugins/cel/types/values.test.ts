import { describe, expect, it } from "vitest";

import { SQLTypeList } from "./values";

describe("SQLTypeList", () => {
  it("puts the unspecified statement type last for approval CEL rules", () => {
    expect(SQLTypeList.ALL.at(-1)).toBe("STATEMENT_TYPE_UNSPECIFIED");
  });
});
