// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  type ListPoliciesRequest,
  ListPoliciesResponseSchema,
  PolicyResourceType,
  PolicySchema,
  PolicyType,
  ReviewRulePolicySchema,
  type UpdatePolicyRequest,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import { createPolicySlice } from "./policy";

const mocks = vi.hoisted(() => ({
  listPolicies: vi.fn(),
  updatePolicy: vi.fn(),
}));

vi.mock("@/api", () => ({
  orgPolicyServiceClientConnect: {
    listPolicies: mocks.listPolicies,
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

  test("lists the rows of a parent, including ones switched off", async () => {
    const row = create(PolicySchema, {
      name: "workspaces/ws/policies/review_rule",
      type: PolicyType.REVIEW_RULE,
      enforce: false,
    });
    mocks.listPolicies.mockResolvedValue(
      create(ListPoliciesResponseSchema, { policies: [row] })
    );
    const store = createStore();

    const policies = await store.listPolicies({
      parentPath: "workspaces/ws",
      policyType: PolicyType.REVIEW_RULE,
      showDeleted: true,
    });

    const request = mocks.listPolicies.mock.calls[0][0] as ListPoliciesRequest;
    expect(request.parent).toBe("workspaces/ws");
    expect(request.policyType).toBe(PolicyType.REVIEW_RULE);
    expect(request.showDeleted).toBe(true);
    expect(policies).toEqual([row]);
  });
});
