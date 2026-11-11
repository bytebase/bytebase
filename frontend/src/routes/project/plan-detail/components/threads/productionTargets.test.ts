import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import { EnvironmentSetting_EnvironmentSchema } from "@/types/proto-es/v1/setting_service_pb";
import type { Environment } from "@/types/v1/environment";
import { productionEnvironmentsOf } from "./productionTargets";

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
