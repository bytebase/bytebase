import { create } from "@bufbuild/protobuf";
import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  type Policy,
  PolicySchema,
  PolicyType,
  ReviewAIPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";

const mocks = vi.hoisted(() => ({
  serverInfo: undefined as
    | { aiEnabled: boolean; aiSqlReviewEnabled: boolean }
    | undefined,
  policies: {} as Record<string, Policy>,
  fetch: vi.fn(),
}));

vi.mock("@/stores/app", () => {
  type Find = { parentPath: string; policyType: number };
  const state = () => ({
    serverInfo: mocks.serverInfo,
    getPolicyByParentAndType: (find: Find) =>
      mocks.policies[`${find.parentPath}:${find.policyType}`],
  });
  return {
    useAppStore: Object.assign(
      (selector: (s: ReturnType<typeof state>) => unknown) => selector(state()),
      {
        getState: () => ({
          getOrFetchPolicyByParentAndType: async (find: Find) => {
            mocks.fetch(find);
            return undefined;
          },
        }),
      }
    ),
  };
});

vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => ({ projectId: "p" }),
}));

import { useAIReviewOn } from "./useAIReviewOn";

const projectPolicy = (enabled: boolean | undefined, enforce = true) => {
  mocks.policies[`projects/p:${PolicyType.REVIEW_AI}`] = create(PolicySchema, {
    name: "projects/p/policies/review_ai",
    type: PolicyType.REVIEW_AI,
    enforce,
    policy: {
      case: "reviewAiPolicy",
      value: create(ReviewAIPolicySchema, { content: "", enabled }),
    },
  });
};

const reviewOn = () => renderHook(() => useAIReviewOn()).result.current;

beforeEach(() => {
  mocks.serverInfo = { aiEnabled: true, aiSqlReviewEnabled: false };
  mocks.policies = {};
  mocks.fetch.mockReset();
});

describe("useAIReviewOn", () => {
  test("follows the workspace switch while the project has none", () => {
    expect(reviewOn()).toBe(false);
    mocks.serverInfo = { aiEnabled: true, aiSqlReviewEnabled: true };
    expect(reviewOn()).toBe(true);
    projectPolicy(undefined);
    expect(reviewOn()).toBe(true);
  });

  test("the project's own switch replaces the workspace's", () => {
    projectPolicy(true);
    expect(reviewOn()).toBe(true);
    mocks.serverInfo = { aiEnabled: true, aiSqlReviewEnabled: true };
    projectPolicy(false);
    expect(reviewOn()).toBe(false);
  });

  test("a project policy that is not enforced counts as absent", () => {
    projectPolicy(true, false);
    expect(reviewOn()).toBe(false);
  });

  test("is off while AI is off for the workspace, and reads no policy", () => {
    mocks.serverInfo = { aiEnabled: false, aiSqlReviewEnabled: true };
    projectPolicy(true);
    expect(reviewOn()).toBe(false);
    expect(mocks.fetch).not.toHaveBeenCalled();
  });

  test("reads the project's AI review policy", () => {
    reviewOn();
    expect(mocks.fetch).toHaveBeenCalledWith({
      parentPath: "projects/p",
      policyType: PolicyType.REVIEW_AI,
    });
  });
});
