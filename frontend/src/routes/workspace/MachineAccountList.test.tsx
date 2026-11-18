import { describe, expect, test } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";
import { accountMatchesList } from "./MachineAccountList";

const account = (state: State) => ({
  title: "Deploy Bot",
  email: "deploy@service.bytebase.com",
  state,
});

describe("accountMatchesList", () => {
  test.each([
    [State.ACTIVE, "active", "", true],
    [State.ACTIVE, "inactive", "", false],
    [State.ACTIVE, "all", "", true],
    [State.DELETED, "active", "", false],
    [State.DELETED, "inactive", "", true],
    [State.DELETED, "all", "", true],
    [State.ACTIVE, "active", "deploy bot", true],
    [State.ACTIVE, "active", " DEPLOY@SERVICE ", true],
    [State.ACTIVE, "active", "terraform", false],
  ] as const)("%s in %s searching %j: %s", (state, filter, query, expected) => {
    expect(accountMatchesList(account(state), filter, query)).toBe(expected);
  });
});
