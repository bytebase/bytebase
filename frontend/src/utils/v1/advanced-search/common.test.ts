import { describe, expect, test } from "vitest";
import {
  buildSearchParamsBySearchText,
  buildSearchTextBySearchParams,
} from "./common";

describe("buildSearchParamsBySearchText", () => {
  test("restores the time range picker's ownership of the created scope", () => {
    expect(
      buildSearchParamsBySearchText("status:OPEN created:1000%2C2000 foo")
    ).toEqual({
      query: "foo",
      scopes: [
        { id: "status", value: "OPEN" },
        { id: "created", value: "1000,2000", readonly: true },
      ],
    });
  });

  test("round-trips a picked range through the search text", () => {
    const params = {
      query: "",
      scopes: [{ id: "created", value: "1000,2000", readonly: true }],
    };

    expect(
      buildSearchParamsBySearchText(buildSearchTextBySearchParams(params))
    ).toEqual(params);
  });
});
