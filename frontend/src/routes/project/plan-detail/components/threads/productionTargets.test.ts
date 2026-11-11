import { create } from "@bufbuild/protobuf";
import { describe, expect, test, vi } from "vitest";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import { EnvironmentSetting_EnvironmentSchema } from "@/types/proto-es/v1/setting_service_pb";
import type { Environment } from "@/types/v1/environment";
import {
  productionEnvironmentsOf,
  requestMissingDatabases,
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

describe("requestMissingDatabases", () => {
  test("asks once for a name that is cached or already being fetched by another card", async () => {
    const pending: (() => void)[] = [];
    const fetch = vi.fn(
      () =>
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
    expect(fetch.mock.calls).toEqual([
      [["instances/i/databases/a"]],
      [["instances/i/databases/b"]],
    ]);
    for (const resolve of pending) resolve();
    await Promise.all([first, second]);
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
