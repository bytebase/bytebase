// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  PolicyResourceType,
  PolicySchema,
  PolicyType,
  ReviewRulePolicySchema,
  type UpdatePolicyRequest,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import { createPolicySlice } from "./policy";

const mocks = vi.hoisted(() => ({
  updatePolicy: vi.fn(),
}));

vi.mock("@/api", () => ({
  orgPolicyServiceClientConnect: {
    updatePolicy: mocks.updatePolicy,
  },
}));

const createStore = () => {
  const state: Record<string, unknown> = {};
  const set = (updater: unknown) => {
    const patch =
      typeof updater === "function"
        ? (updater as (value: typeof state) => object)(state)
        : updater;
    Object.assign(state, patch);
  };
  const get = () => state;
  Object.assign(
    state,
    createPolicySlice(set as never, get as never, {} as never)
  );
  return state as ReturnType<typeof createPolicySlice>;
};

describe("policy store", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.updatePolicy.mockImplementation(
      async (request: UpdatePolicyRequest) =>
        create(PolicySchema, request.policy)
    );
  });

  test("upserts a review rule policy through its update mask", async () => {
    const store = createStore();

    await store.upsertPolicy({
      parentPath: "projects/p",
      policy: {
        type: PolicyType.REVIEW_RULE,
        resourceType: PolicyResourceType.PROJECT,
        enforce: true,
        policy: {
          case: "reviewRulePolicy",
          value: create(ReviewRulePolicySchema, {
            rules: [ReviewRuleType.SYNTAX],
          }),
        },
      },
    });

    const request = mocks.updatePolicy.mock.calls[0][0] as UpdatePolicyRequest;
    expect(request.policy?.name).toBe("projects/p/policies/review_rule");
    expect(request.policy?.enforce).toBe(true);
    expect(request.updateMask?.paths).toEqual([
      "review_rule_policy",
      "enforce",
    ]);
    expect(request.allowMissing).toBe(true);
    expect(
      store.getPolicyByParentAndType({
        parentPath: "projects/p",
        policyType: PolicyType.REVIEW_RULE,
      })?.policy.value
    ).toEqual(expect.objectContaining({ rules: [ReviewRuleType.SYNTAX] }));
  });
});
