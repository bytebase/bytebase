import { describe, expect, test } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  defaultActiveStateSearchParams,
  getResourceStateFilter,
} from "./resourceStateFilter";

describe("defaultActiveStateSearchParams", () => {
  test("uses active state as the default filter", () => {
    expect(defaultActiveStateSearchParams(undefined)).toEqual({
      query: "",
      scopes: [{ id: "state", value: "ACTIVE" }],
    });
  });

  test("preserves legacy archived and all state query values", () => {
    expect(defaultActiveStateSearchParams("archived").scopes).toEqual([
      { id: "state", value: "DELETED" },
    ]);
    expect(defaultActiveStateSearchParams("all").scopes).toEqual([
      { id: "state", value: "ALL" },
    ]);
  });
});

describe("getResourceStateFilter", () => {
  test("maps an explicit active state to active resources", () => {
    expect(getResourceStateFilter("ACTIVE")).toBe(State.ACTIVE);
  });

  test("maps an explicit deleted state to archived resources", () => {
    expect(getResourceStateFilter("DELETED")).toBe(State.DELETED);
  });

  test("maps all or no state scope to all resources", () => {
    expect(getResourceStateFilter("ALL")).toBeUndefined();
    expect(getResourceStateFilter(undefined)).toBeUndefined();
  });
});
