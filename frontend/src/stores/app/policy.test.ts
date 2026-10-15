// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  PolicyResourceType,
  PolicySchema,
  PolicyType,
  ReviewAIPolicySchema,
  ReviewRulePolicySchema,
  type UpdatePolicyRequest,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import { createPolicySlice } from "./policy";

const mocks = vi.hoisted(() => ({
  getPolicy: vi.fn(),
  updatePolicy: vi.fn(),
}));

vi.mock("@/api", () => ({
  orgPolicyServiceClientConnect: {
    getPolicy: mocks.getPolicy,
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

  test("upserts an AI review policy under policies/review_ai, leaving enforce as stored", async () => {
    const store = createStore();

    await store.upsertPolicy({
      parentPath: "workspaces/ws",
      policy: {
        type: PolicyType.REVIEW_AI,
        resourceType: PolicyResourceType.WORKSPACE,
        enforce: true,
        policy: {
          case: "reviewAiPolicy",
          value: create(ReviewAIPolicySchema, { content: "No DROP." }),
        },
      },
    });

    const request = mocks.updatePolicy.mock.calls[0][0] as UpdatePolicyRequest;
    expect(request.policy?.name).toBe("workspaces/ws/policies/review_ai");
    expect(request.policy?.type).toBe(PolicyType.REVIEW_AI);
    expect(request.policy?.policy.case).toBe("reviewAiPolicy");
    expect(request.policy?.policy.value).toEqual(
      expect.objectContaining({ content: "No DROP." })
    );
    expect(request.updateMask?.paths).toEqual(["review_ai_policy"]);
    expect(request.allowMissing).toBe(true);
    expect(
      store.getPolicyByParentAndType({
        parentPath: "workspaces/ws",
        policyType: PolicyType.REVIEW_AI,
      })?.policy.value
    ).toEqual(expect.objectContaining({ content: "No DROP." }));
  });

  test("fetches a policy, and tells an absent one from a failed read", async () => {
    const store = createStore();
    const find = {
      parentPath: "projects/p",
      policyType: PolicyType.REVIEW_RULE,
      refresh: true,
    };
    const row = create(PolicySchema, {
      name: "projects/p/policies/review_rule",
      type: PolicyType.REVIEW_RULE,
      enforce: true,
    });

    mocks.getPolicy.mockResolvedValueOnce(row);
    await expect(store.fetchPolicyByParentAndType(find)).resolves.toBe(row);
    expect(store.getPolicyByParentAndType(find)).toBe(row);

    // A failed read leaves the last policy read in the cache.
    mocks.getPolicy.mockRejectedValueOnce(
      new ConnectError("gone", Code.Unavailable)
    );
    await expect(
      store.fetchPolicyByParentAndType(find)
    ).resolves.toBeUndefined();
    expect(store.getPolicyByParentAndType(find)).toBe(row);

    // An absent policy caches a stand-in without a payload.
    mocks.getPolicy.mockRejectedValueOnce(
      new ConnectError("missing", Code.NotFound)
    );
    await expect(store.fetchPolicyByParentAndType(find)).resolves.toBeNull();
    expect(store.getPolicyByParentAndType(find)?.policy.case).toBeUndefined();

    mocks.getPolicy.mockRejectedValueOnce(
      new ConnectError("missing", Code.NotFound)
    );
    await expect(
      store.getOrFetchPolicyByParentAndType(find)
    ).resolves.toBeUndefined();
  });
});
