// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { QueryDataPolicySchema } from "@/types/proto-es/v1/org_policy_service_pb";
import { getEffectiveQueryDataPolicy } from "./queryDataPolicy";

describe("getEffectiveQueryDataPolicy", () => {
  test.each([
    { workspaceLimit: 500, projectLimit: 50, expected: 50 },
    { workspaceLimit: 50, projectLimit: 500, expected: 50 },
    { workspaceLimit: 0, projectLimit: 500, expected: 500 },
    { workspaceLimit: 500, projectLimit: -1, expected: 500 },
    { workspaceLimit: -1, projectLimit: 0, expected: Number.MAX_VALUE },
  ])(
    "merges workspace $workspaceLimit and project $projectLimit row caps",
    ({ workspaceLimit, projectLimit, expected }) => {
      const policy = getEffectiveQueryDataPolicy(
        create(QueryDataPolicySchema, { maximumResultRows: workspaceLimit }),
        create(QueryDataPolicySchema, { maximumResultRows: projectLimit })
      );

      expect(policy.maximumResultRows).toBe(expected);
    }
  );

  test("defaults missing policies to unrestricted rows and disabled flags", () => {
    expect(getEffectiveQueryDataPolicy()).toEqual({
      disableCopyData: false,
      disableDownload: false,
      allowAdminDataSource: false,
      maximumResultRows: Number.MAX_VALUE,
    });
  });

  test("uses a cap from either policy when the other is missing", () => {
    const policy = create(QueryDataPolicySchema, { maximumResultRows: 75 });

    expect(getEffectiveQueryDataPolicy(policy).maximumResultRows).toBe(75);
    expect(
      getEffectiveQueryDataPolicy(undefined, policy).maximumResultRows
    ).toBe(75);
  });

  test("keeps non-row policy flags scoped to the workspace", () => {
    const workspacePolicy = create(QueryDataPolicySchema, {
      disableCopyData: true,
      disableDownload: false,
      allowAdminDataSource: true,
      maximumResultRows: 500,
    });
    const projectPolicy = create(QueryDataPolicySchema, {
      disableCopyData: false,
      disableDownload: true,
      allowAdminDataSource: false,
      maximumResultRows: 100,
    });

    expect(getEffectiveQueryDataPolicy(workspacePolicy, projectPolicy)).toEqual(
      {
        disableCopyData: true,
        disableDownload: false,
        allowAdminDataSource: true,
        maximumResultRows: 100,
      }
    );
    expect(getEffectiveQueryDataPolicy(undefined, projectPolicy)).toEqual({
      disableCopyData: false,
      disableDownload: false,
      allowAdminDataSource: false,
      maximumResultRows: 100,
    });
  });
});
