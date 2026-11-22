import { describe, expect, test } from "vitest";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { getPlanCreateAgentContextQuery } from "./agentContext";

const makeSpec = (id: string, targets: string[]) =>
  ({
    id,
    config: { case: "changeDatabaseConfig", value: { targets } },
  }) as unknown as Plan_Spec;

describe("getPlanCreateAgentContextQuery", () => {
  test("tracks the selected spec without replacing the plan's initial targets", () => {
    expect(
      getPlanCreateAgentContextQuery(
        {
          databaseList: "instances/i1/databases/db-a",
          sqlStorageKey: "keep-this-query",
        },
        [
          makeSpec("spec-a", ["instances/i1/databases/db-a"]),
          makeSpec("spec-b", ["instances/i1/databases/db-b"]),
        ],
        "spec-b"
      )
    ).toEqual({
      agentDatabaseName: "instances/i1/databases/db-b",
      databaseList: "instances/i1/databases/db-a",
      sqlStorageKey: "keep-this-query",
    });
  });

  test("clears stale Agent bindings when the selected spec has no targets", () => {
    expect(
      getPlanCreateAgentContextQuery(
        {
          databaseList: "instances/i1/databases/db-a",
          agentDatabaseName: "instances/i1/databases/db-a",
        },
        [makeSpec("spec-a", [])],
        "spec-a"
      )
    ).toEqual({
      agentDatabaseName: "",
      databaseList: "instances/i1/databases/db-a",
    });
  });

  test("does not bind a group or multi-target spec to one database", () => {
    expect(
      getPlanCreateAgentContextQuery(
        { databaseList: "instances/i1/databases/db-a" },
        [
          makeSpec("spec-a", [
            "instances/i1/databases/db-a",
            "instances/i1/databases/db-b",
          ]),
        ],
        "spec-a"
      )
    ).toEqual({
      agentDatabaseName: "",
      databaseList: "instances/i1/databases/db-a",
    });
  });

  test("does not replace an already current Agent binding", () => {
    const query = {
      agentDatabaseName: "instances/i1/databases/db-a",
      databaseList: "instances/i1/databases/db-a",
      keep: "x",
    };
    expect(
      getPlanCreateAgentContextQuery(
        query,
        [makeSpec("spec-a", ["instances/i1/databases/db-a"])],
        "spec-a"
      )
    ).toBeUndefined();
  });
});
