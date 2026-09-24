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
  permissions: {} as Record<string, boolean>,
  fetchPolicy: vi.fn(),
  listPolicies: vi.fn(),
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
    mocks.permissions[permission] ?? true,
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
      refresh?: boolean;
    }) => Promise<StorePolicy | undefined>;
    listPolicies: (params: {
      parentPath: string;
      showDeleted?: boolean;
    }) => Promise<StorePolicy[]>;
    upsertPolicy: (params: {
      parentPath: string;
      policy: Partial<StorePolicy>;
    }) => Promise<StorePolicy>;
  }>()((set, get) => ({
    policies: {},
    getPolicyByParentAndType: ({ parentPath }) => get().policies[parentPath],
    // A failed read caches nothing, as the real store does.
    getOrFetchPolicyByParentAndType: async (params) => {
      mocks.fetchPolicy(params);
      return get().policies[params.parentPath];
    },
    listPolicies: async (params) => mocks.listPolicies(params),
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

const reviewRulePolicy = (
  rules: ReviewRuleType[],
  enforce = true
): Policy =>
  create(PolicySchema, {
    name: `${WORKSPACE}/policies/review_rule`,
    type: PolicyType.REVIEW_RULE,
    enforce,
    policy: {
      case: "reviewRulePolicy",
      value: create(ReviewRulePolicySchema, { rules }),
    },
  });

const seedWorkspaceRow = (policy: Policy | null) => {
  seedPolicies(policy ? { [WORKSPACE]: policy } : {});
  mocks.listPolicies.mockResolvedValue(policy ? [policy] : []);
};

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

const everySwitch = () => screen.getAllByRole("switch");

const findSwitches = () => screen.findAllByRole("switch");

describe("SQLReviewStandardRulesSection", () => {
  beforeEach(() => {
    mocks.permissions = {};
    mocks.fetchPolicy.mockReset();
    mocks.listPolicies.mockReset();
    mocks.upsertPolicy.mockReset();
    mocks.pushNotification.mockReset();
    seedWorkspaceRow(reviewRulePolicy([...STANDARD_RULE_TYPES]));
  });

  test("saves the edited rules as the enforced workspace policy", async () => {
    render(<Harness />);
    await findSwitches();
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
    expect(policy.enforce).toBe(true);
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

  test("switching a rule back is not a change, and revert drops the draft", async () => {
    render(<Harness />);
    await findSwitches();

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
    await findSwitches();

    fireEvent.click(ruleSwitch("require-where"));
    await act(async () => {
      fireEvent.click(screen.getByText("save"));
    });

    expect(screen.getByTestId("dirty")).toHaveTextContent("true");
    expect(mocks.pushNotification).not.toHaveBeenCalled();
  });

  test("a policy switched off through the API shows every rule on", async () => {
    seedWorkspaceRow(reviewRulePolicy([ReviewRuleType.SYNTAX], false));
    render(<Harness />);

    expect(await findSwitches()).toHaveLength(STANDARD_RULE_TYPES.length);
    for (const control of everySwitch()) {
      expect(control).toHaveAttribute("aria-checked", "true");
    }
    expect(
      screen.getByText("sql-review.standard-rules.running")
    ).toBeInTheDocument();
  });

  test("a policy without SYNTAX shows every rule off", async () => {
    seedWorkspaceRow(reviewRulePolicy([ReviewRuleType.REQUIRE_WHERE]));
    render(<Harness />);

    for (const control of await findSwitches()) {
      expect(control).toHaveAttribute("aria-checked", "false");
    }
    expect(
      screen.getByText("sql-review.standard-rules.syntax-off")
    ).toBeInTheDocument();
  });

  test("a failed read shows an error, and retry reads again", async () => {
    seedPolicies({});
    render(<Harness />);

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "sql-review.standard-rules.load-failed"
    );
    expect(screen.queryAllByRole("switch")).toHaveLength(0);

    seedPolicies({ [WORKSPACE]: reviewRulePolicy([...STANDARD_RULE_TYPES]) });
    fireEvent.click(
      screen.getByRole("button", { name: "sql-review.standard-rules.retry" })
    );

    expect(await findSwitches()).toHaveLength(STANDARD_RULE_TYPES.length);
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(mocks.fetchPolicy).toHaveBeenLastCalledWith(
      expect.objectContaining({ parentPath: WORKSPACE, refresh: true })
    );
  });

  test("reads the workspace's row whether or not it is enforced", async () => {
    render(<Harness />);
    await findSwitches();

    expect(mocks.listPolicies).toHaveBeenCalledWith({
      parentPath: WORKSPACE,
      policyType: PolicyType.REVIEW_RULE,
      showDeleted: true,
    });
  });

  // Editing an existing row is an update; the first save creates the row.
  test.each([
    ["an existing row", "bb.policies.update", "bb.policies.create"],
    ["no row", "bb.policies.create", "bb.policies.update"],
  ])(
    "with %s, the switches need %s and not %s",
    async (row, needed, unneeded) => {
      if (row === "no row") {
        // GetPolicy stands in for the missing row; only the list is empty.
        mocks.listPolicies.mockResolvedValue([]);
      }
      mocks.permissions = { [needed]: false };
      const { unmount } = render(<Harness />);
      for (const control of await findSwitches()) {
        expect(control).toHaveAttribute("data-disabled");
      }
      unmount();

      mocks.permissions = { [unneeded]: false };
      render(<Harness />);
      for (const control of await findSwitches()) {
        expect(control).not.toHaveAttribute("data-disabled");
      }
    }
  );

  test.each([["bb.policies.update"], ["bb.policies.create"]])(
    "without the list permission, the switches need %s",
    async (permission) => {
      mocks.permissions = { "bb.policies.list": false, [permission]: false };
      render(<Harness />);

      for (const control of await findSwitches()) {
        expect(control).toHaveAttribute("data-disabled");
      }
      expect(mocks.listPolicies).not.toHaveBeenCalled();
    }
  );

  test("a save records the row, so later saves need only update", async () => {
    mocks.listPolicies.mockResolvedValue([]);
    mocks.permissions = { "bb.policies.update": false };
    render(<Harness />);
    await findSwitches();

    fireEvent.click(ruleSwitch("require-where"));
    await act(async () => {
      fireEvent.click(screen.getByText("save"));
    });

    expect(mocks.upsertPolicy).toHaveBeenCalledTimes(1);
    for (const control of everySwitch()) {
      expect(control).toHaveAttribute("data-disabled");
    }
  });
});
