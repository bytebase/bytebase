import { create } from "@bufbuild/protobuf";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { STANDARD_RULE_TYPES } from "@/components/sql-review/standardRules";
import {
  type Policy,
  PolicyResourceType,
  PolicySchema,
  PolicyType,
  ReviewRulePolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";

const WORKSPACE = "workspaces/ws";

const mocks = vi.hoisted(() => ({
  canUpdate: { value: true },
  upsertPolicy: vi.fn(),
  pushNotification: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/hooks/useAppState", () => ({
  useWorkspaceResourceName: () => "workspaces/ws",
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: (permission: string) =>
    permission === "bb.policies.update" ? mocks.canUpdate.value : true,
}));

vi.mock("@/stores", () => ({ pushNotification: mocks.pushNotification }));

vi.mock("@/stores/app", async () => {
  const { create: createStore } = await import("zustand");
  type StorePolicy = import("@/types/proto-es/v1/org_policy_service_pb").Policy;
  const useAppStore = createStore<{
    policies: Record<string, StorePolicy>;
    getPolicyByParentAndType: (params: {
      parentPath: string;
    }) => StorePolicy | undefined;
    getOrFetchPolicyByParentAndType: (params: {
      parentPath: string;
    }) => Promise<StorePolicy | undefined>;
    upsertPolicy: (params: {
      parentPath: string;
      policy: Partial<StorePolicy>;
    }) => Promise<StorePolicy>;
  }>()((set, get) => ({
    policies: {},
    getPolicyByParentAndType: ({ parentPath }) => get().policies[parentPath],
    getOrFetchPolicyByParentAndType: async ({ parentPath }) =>
      get().policies[parentPath],
    upsertPolicy: async (params) => {
      await mocks.upsertPolicy(params);
      const saved = params.policy as StorePolicy;
      set((state) => ({
        policies: { ...state.policies, [params.parentPath]: saved },
      }));
      return saved;
    },
  }));
  return { useAppStore };
});

const { useAppStore } = await import("@/stores/app");
// The mocked store holds only the policy cache.
const seedPolicies = (policies: Record<string, Policy>) =>
  (
    useAppStore as unknown as {
      setState: (state: { policies: Record<string, Policy> }) => void;
    }
  ).setState({ policies });
const { SQLReviewStandardRulesSection, useWorkspaceStandardRules } =
  await import("./SQLReviewStandardRulesSection");

const reviewRulePolicy = (rules: ReviewRuleType[]): Policy =>
  create(PolicySchema, {
    name: `${WORKSPACE}/policies/review_rule`,
    type: PolicyType.REVIEW_RULE,
    policy: {
      case: "reviewRulePolicy",
      value: create(ReviewRulePolicySchema, { rules }),
    },
  });

function Harness() {
  const standardRules = useWorkspaceStandardRules(true);
  return (
    <>
      <SQLReviewStandardRulesSection standardRules={standardRules} />
      <output data-testid="dirty">{String(standardRules.isDirty)}</output>
      <button type="button" onClick={() => void standardRules.save()}>
        save
      </button>
      <button type="button" onClick={standardRules.revert}>
        revert
      </button>
    </>
  );
}

const ruleSwitch = (rule: string) =>
  screen.getByRole("switch", {
    name: `sql-review.standard-rules.rule.${rule}.title`,
  });

describe("SQLReviewStandardRulesSection", () => {
  beforeEach(() => {
    mocks.canUpdate.value = true;
    mocks.upsertPolicy.mockReset();
    mocks.pushNotification.mockReset();
    seedPolicies({ [WORKSPACE]: reviewRulePolicy([...STANDARD_RULE_TYPES]) });
  });

  test("saves the edited rules as the workspace policy", async () => {
    render(<Harness />);
    expect(screen.getByTestId("dirty")).toHaveTextContent("false");

    fireEvent.click(ruleSwitch("disallow-truncate"));
    expect(screen.getByTestId("dirty")).toHaveTextContent("true");

    await act(async () => {
      fireEvent.click(screen.getByText("save"));
    });

    expect(mocks.upsertPolicy).toHaveBeenCalledTimes(1);
    const { parentPath, policy } = mocks.upsertPolicy.mock.calls[0][0];
    expect(parentPath).toBe(WORKSPACE);
    expect(policy.type).toBe(PolicyType.REVIEW_RULE);
    expect(policy.resourceType).toBe(PolicyResourceType.WORKSPACE);
    expect(policy.policy.value.rules).toEqual(
      STANDARD_RULE_TYPES.filter(
        (rule) => rule !== ReviewRuleType.DISALLOW_TRUNCATE
      )
    );
    expect(screen.getByTestId("dirty")).toHaveTextContent("false");
    expect(ruleSwitch("disallow-truncate")).toHaveAttribute(
      "aria-checked",
      "false"
    );
  });

  test("switching a rule back is not a change, and revert drops the draft", () => {
    render(<Harness />);

    fireEvent.click(ruleSwitch("require-where"));
    fireEvent.click(ruleSwitch("require-where"));
    expect(screen.getByTestId("dirty")).toHaveTextContent("false");

    fireEvent.click(ruleSwitch("syntax"));
    expect(screen.getByTestId("dirty")).toHaveTextContent("true");
    fireEvent.click(screen.getByText("revert"));
    expect(screen.getByTestId("dirty")).toHaveTextContent("false");
    expect(ruleSwitch("syntax")).toHaveAttribute("aria-checked", "true");
  });

  test("keeps the draft when the save fails", async () => {
    mocks.upsertPolicy.mockRejectedValue(new Error("denied"));
    render(<Harness />);

    fireEvent.click(ruleSwitch("require-where"));
    await act(async () => {
      fireEvent.click(screen.getByText("save"));
    });

    expect(screen.getByTestId("dirty")).toHaveTextContent("true");
    expect(mocks.pushNotification).not.toHaveBeenCalled();
  });

  test("locks the switches without bb.policies.update", () => {
    mocks.canUpdate.value = false;
    render(<Harness />);

    for (const control of screen.getAllByRole("switch")) {
      expect(control).toHaveAttribute("data-disabled");
    }
  });
});
