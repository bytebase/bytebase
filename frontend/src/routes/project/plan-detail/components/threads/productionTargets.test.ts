import { create } from "@bufbuild/protobuf";
import { describe, expect, test, vi } from "vitest";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import { EnvironmentSetting_EnvironmentSchema } from "@/types/proto-es/v1/setting_service_pb";
import type { Environment } from "@/types/v1/environment";
import {
  groupTargetsByEnvironment,
  productionEnvironmentsOf,
  requestMissingDatabases,
  type TargetGroup,
} from "./productionTargets";

const environment = (
  id: string,
  isProtected: boolean,
  order: number
): Environment => ({
  ...create(EnvironmentSetting_EnvironmentSchema, {
    name: `environments/${id}`,
    id,
    title: id,
    tags: isProtected ? { protected: "protected" } : {},
  }),
  color: "",
  order,
});
// The workspace's order: prod-eu first, then prod-us, then test.
const environments = [
  environment("prod-eu", true, 0),
  environment("prod-us", true, 1),
  environment("test", false, 2),
];
const database = (name: string, env: string) =>
  create(DatabaseSchema$, {
    name,
    effectiveEnvironment: `environments/${env}`,
  });
const databasesByName = {
  "instances/i/databases/app_us": database(
    "instances/i/databases/app_us",
    "prod-us"
  ),
  "instances/i/databases/app_eu": database(
    "instances/i/databases/app_eu",
    "prod-eu"
  ),
  "instances/i/databases/app_test": database(
    "instances/i/databases/app_test",
    "test"
  ),
};
const resources = { databasesByName, environmentList: environments };

describe("productionEnvironmentsOf", () => {
  test.each([
    {
      targets: [
        "instances/i/databases/app_test",
        "instances/i/databases/app_us",
      ],
      feature: true,
      expected: ["prod-us"],
    },
    {
      targets: ["instances/i/databases/app_us", "instances/i/databases/app_eu"],
      feature: true,
      expected: ["prod-eu", "prod-us"],
    },
    {
      targets: ["instances/i/databases/app_test"],
      feature: true,
      expected: [],
    },
    { targets: ["instances/i/databases/app_us"], feature: false, expected: [] },
    { targets: ["instances/i/databases/unknown"], feature: true, expected: [] },
    { targets: [], feature: true, expected: [] },
  ])(
    "targets $targets with the environment-tier feature $feature: $expected",
    ({ targets, feature, expected }) => {
      expect(
        productionEnvironmentsOf(targets, resources, feature).map(
          (environment) => environment.id
        )
      ).toEqual(expected);
    }
  );
});

describe("groupTargetsByEnvironment", () => {
  // The workspace's order, which is the rollout order: dev to dr, with prod
  // the one production environment, ahead of a disaster-recovery copy.
  const rollout = [
    environment("dev", false, 0),
    environment("test", false, 1),
    environment("prod", true, 2),
    environment("dr", false, 3),
  ];
  const at = (instance: string, name: string) =>
    `instances/${instance}/databases/${name}`;
  const loaded = (
    entries: [string, string | undefined, string?][]
  ): Record<string, ReturnType<typeof database>> =>
    Object.fromEntries(
      entries.map(([name, env, instanceEnv]) => [
        name,
        create(DatabaseSchema$, {
          name,
          effectiveEnvironment: env && `environments/${env}`,
          instanceResource: instanceEnv
            ? { environment: `environments/${instanceEnv}` }
            : undefined,
        }),
      ])
    );
  const shape = (groups: TargetGroup[]) =>
    groups.map((group) => [
      group.kind === "environment" ? group.environment.id : group.kind,
      group.targets,
    ]);

  test("puts production first, then the rest from the last rollout stage back", () => {
    const targets = [at("i", "a"), at("i", "b"), at("i", "c"), at("i", "d")];
    const databasesByName = loaded([
      [at("i", "a"), "dev"],
      [at("i", "b"), "dr"],
      [at("i", "c"), "prod"],
      [at("i", "d"), "test"],
    ]);
    expect(
      shape(
        groupTargetsByEnvironment(
          targets,
          { databasesByName, environmentList: rollout },
          true
        )
      )
    ).toEqual([
      ["prod", [at("i", "c")]],
      ["dr", [at("i", "b")]],
      ["test", [at("i", "d")]],
      ["dev", [at("i", "a")]],
    ]);
  });

  test("without the environment-tier feature, orders by rollout stage alone", () => {
    const targets = [at("i", "b"), at("i", "c")];
    const databasesByName = loaded([
      [at("i", "b"), "dr"],
      [at("i", "c"), "prod"],
    ]);
    expect(
      shape(
        groupTargetsByEnvironment(
          targets,
          { databasesByName, environmentList: rollout },
          false
        )
      ).map(([id]) => id)
    ).toEqual(["dr", "prod"]);
  });

  test("puts an environment the workspace no longer lists after the listed ones, then unassigned, then unknown", () => {
    const targets = [
      at("i", "gone"),
      at("i", "none"),
      at("i", "old"),
      at("i", "a"),
      "instances/i/databases/s/schemas/public",
    ];
    const databasesByName = loaded([
      [at("i", "none"), undefined],
      [at("i", "old"), "retired"],
      [at("i", "a"), "dev"],
    ]);
    expect(
      shape(
        groupTargetsByEnvironment(
          targets,
          { databasesByName, environmentList: rollout },
          true
        )
      )
    ).toEqual([
      ["dev", [at("i", "a")]],
      ["retired", [at("i", "old")]],
      ["unassigned", [at("i", "none")]],
      ["unknown", [at("i", "gone"), "instances/i/databases/s/schemas/public"]],
    ]);
  });

  test("falls back to the instance's environment, and orders a group by database name, then instance", () => {
    const targets = [
      at("west", "orders"),
      at("east", "users"),
      at("east", "orders"),
    ];
    const databasesByName = loaded([
      [at("west", "orders"), undefined, "prod"],
      [at("east", "users"), "prod"],
      [at("east", "orders"), "prod"],
    ]);
    expect(
      shape(
        groupTargetsByEnvironment(
          targets,
          { databasesByName, environmentList: rollout },
          true
        )
      )
    ).toEqual([
      [
        "prod",
        [at("east", "orders"), at("west", "orders"), at("east", "users")],
      ],
    ]);
  });
});

describe("requestMissingDatabases", () => {
  test("cards that ask in the same tick share one request for all their uncached names", async () => {
    const pending: (() => void)[] = [];
    const fetch = vi.fn(
      (_names: string[]) =>
        new Promise<void>((resolve) => {
          pending.push(resolve);
        })
    );
    const cached = (name: string) => name === "instances/i/databases/cached";
    const first = requestMissingDatabases(
      ["instances/i/databases/a", "instances/i/databases/cached"],
      cached,
      fetch
    );
    const second = requestMissingDatabases(
      ["instances/i/databases/a", "instances/i/databases/b"],
      cached,
      fetch
    );
    await Promise.resolve();
    expect(fetch.mock.calls).toEqual([
      [["instances/i/databases/a", "instances/i/databases/b"]],
    ]);
    // A later card does not ask again for a name that request still holds.
    const third = requestMissingDatabases(
      ["instances/i/databases/a", "instances/i/databases/c"],
      cached,
      fetch
    );
    await Promise.resolve();
    expect(fetch.mock.calls.at(-1)).toEqual([["instances/i/databases/c"]]);
    for (const resolve of pending) resolve();
    await Promise.all([first, second, third]);
  });

  test("a card waits for a name another card's request holds", async () => {
    let release = () => {};
    const fetch = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          release = resolve;
        })
    );
    const first = requestMissingDatabases(
      ["instances/i/databases/held"],
      () => false,
      fetch
    );
    let secondDone = false;
    const second = requestMissingDatabases(
      ["instances/i/databases/held"],
      () => false,
      fetch
    ).then(() => {
      secondDone = true;
    });
    await Promise.resolve();
    await Promise.resolve();
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(secondDone).toBe(false);
    release();
    await Promise.all([first, second]);
    expect(secondDone).toBe(true);
  });

  test("asks again after a request ends, so a name that failed is retried by the next card", async () => {
    const fetch = vi.fn(() => Promise.reject(new Error("not found")));
    await requestMissingDatabases(
      ["instances/i/databases/gone"],
      () => false,
      fetch
    ).catch(() => {});
    await requestMissingDatabases(
      ["instances/i/databases/gone"],
      () => false,
      fetch
    ).catch(() => {});
    expect(fetch).toHaveBeenCalledTimes(2);
  });
});
