import { create } from "@bufbuild/protobuf";
import { act, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  type Policy,
  PolicyResourceType,
  PolicySchema,
  PolicyType,
  ReviewRulePolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";

const WORKSPACE = "workspaces/ws";
const PROJECT = "projects/p";

const mocks = vi.hoisted(() => ({
  permissions: {} as Record<string, boolean>,
  workspacePermissions: {} as Record<string, boolean>,
  // Parents whose read fails: the store then caches nothing for them.
  unreadable: new Set<string>(),
  upsertPolicy: vi.fn(),
  deletePolicy: vi.fn(),
  fetchPolicy: vi.fn(),
  pushNotification: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/hooks/useAppState", () => ({
  useWorkspaceResourceName: () => "workspaces/ws",
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: (name: string) => ({ name }),
}));

vi.mock("@/hooks/useUnsavedChangesGuard", () => ({
  useUnsavedChangesGuard: vi.fn(),
}));

vi.mock("@/utils", () => ({
  hasProjectPermissionV2: (_project: unknown, permission: string) =>
    mocks.permissions[permission] ?? true,
  hasWorkspacePermissionV2: (permission: string) =>
    mocks.workspacePermissions[permission] ?? true,
}));

vi.mock("@/stores", () => ({ pushNotification: mocks.pushNotification }));
vi.mock("@/stores/modules/v1/common", () => ({
  projectNamePrefix: "projects/",
}));

vi.mock("@/components/RouterLink", () => ({
  RouterLink: ({ children }: { children: ReactNode }) => (
    <a href="/">{children}</a>
  ),
}));

// Mirrors the real store: a fetch of a missing policy caches one with no
// payload and resolves to null, a failed read resolves to undefined and
// leaves the cache as it was, and a delete drops the cached entry.
vi.mock("@/stores/app", async () => {
  const { create: createStore } = await import("zustand");
  const { create: createMessage } = await import("@bufbuild/protobuf");
  const { PolicySchema: Schema } = await import(
    "@/types/proto-es/v1/org_policy_service_pb"
  );
  type StorePolicy = import("@/types/proto-es/v1/org_policy_service_pb").Policy;
  const useAppStore = createStore<{
    policies: Record<string, StorePolicy>;
    getPolicyByParentAndType: (params: {
      parentPath: string;
    }) => StorePolicy | undefined;
    fetchPolicyByParentAndType: (params: {
      parentPath: string;
      refresh?: boolean;
    }) => Promise<StorePolicy | null | undefined>;
    getOrFetchPolicyByParentAndType: (params: {
      parentPath: string;
      refresh?: boolean;
    }) => Promise<StorePolicy | undefined>;
    upsertPolicy: (params: {
      parentPath: string;
      policy: Partial<StorePolicy>;
    }) => Promise<StorePolicy>;
    deletePolicy: (name: string) => Promise<void>;
  }>()((set, get) => ({
    policies: {},
    getPolicyByParentAndType: ({ parentPath }) => get().policies[parentPath],
    fetchPolicyByParentAndType: async (params) => {
      mocks.fetchPolicy(params);
      if (mocks.unreadable.has(params.parentPath)) return undefined;
      if (!get().policies[params.parentPath]) {
        set((state) => ({
          policies: {
            ...state.policies,
            [params.parentPath]: createMessage(Schema, {
              name: `${params.parentPath}/policies/review_rule`,
            }),
          },
        }));
        return null;
      }
      return get().policies[params.parentPath];
    },
    getOrFetchPolicyByParentAndType: async (params) =>
      (await get().fetchPolicyByParentAndType(params)) ?? undefined,
    upsertPolicy: async (params) => {
      await mocks.upsertPolicy(params);
      const saved = {
        ...params.policy,
        name: `${params.parentPath}/policies/review_rule`,
      } as StorePolicy;
      set((state) => ({
        policies: { ...state.policies, [params.parentPath]: saved },
      }));
      return saved;
    },
    deletePolicy: async (name) => {
      await mocks.deletePolicy(name);
      set((state) => ({
        policies: Object.fromEntries(
          Object.entries(state.policies).filter(
            ([, policy]) => policy.name !== name
          )
        ),
      }));
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
const { ProjectSQLReviewPage } = await import("./ProjectSQLReviewPage");

const reviewRulePolicy = (
  parent: string,
  rules: ReviewRuleType[],
  enforce = true
): Policy =>
  create(PolicySchema, {
    name: `${parent}/policies/review_rule`,
    type: PolicyType.REVIEW_RULE,
    enforce,
    policy: {
      case: "reviewRulePolicy",
      value: create(ReviewRulePolicySchema, { rules }),
    },
  });

const workspacePolicy = () =>
  reviewRulePolicy(WORKSPACE, [
    ReviewRuleType.SYNTAX,
    ReviewRuleType.REQUIRE_WHERE,
  ]);

const seedCustomized = () =>
  seedPolicies({
    [WORKSPACE]: workspacePolicy(),
    [PROJECT]: reviewRulePolicy(PROJECT, [ReviewRuleType.SYNTAX]),
  });

const renderLoaded = async () => {
  render(<ProjectSQLReviewPage projectId="p" />);
  return screen.findByRole("switch", {
    name: "sql-review.standard-rules.customize.self",
  });
};

const ruleSwitch = (rule: string) =>
  screen.getByRole("switch", {
    name: `sql-review.standard-rules.rule.${rule}.title`,
  });

const updateButton = () =>
  screen.queryByRole("button", { name: "common.update" });

const update = () =>
  act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "common.update" }));
  });

describe("ProjectSQLReviewPage", () => {
  beforeEach(() => {
    mocks.permissions = {};
    mocks.workspacePermissions = {};
    mocks.unreadable.clear();
    mocks.upsertPolicy.mockReset();
    mocks.deletePolicy.mockReset();
    mocks.fetchPolicy.mockReset();
    mocks.pushNotification.mockReset();
    seedPolicies({ [WORKSPACE]: workspacePolicy() });
  });

  test("a project without its own policy shows the workspace rules as text", async () => {
    const customize = await renderLoaded();

    expect(customize).toHaveAttribute("aria-checked", "false");
    // Only the customize control is a switch; the rules are read-only.
    expect(screen.getAllByRole("switch")).toHaveLength(1);
    expect(screen.getAllByText("sql-review.standard-rules.on")).toHaveLength(2);
    expect(screen.getAllByText("sql-review.standard-rules.off")).toHaveLength(
      9
    );
    expect(
      screen.getByText("sql-review.standard-rules.running")
    ).toBeInTheDocument();
    expect(updateButton()).not.toBeInTheDocument();
  });

  test("customizing starts from the workspace rules and saves an enforced project policy", async () => {
    const customize = await renderLoaded();

    fireEvent.click(customize);
    expect(ruleSwitch("require-where")).toHaveAttribute("aria-checked", "true");
    expect(ruleSwitch("disallow-truncate")).toHaveAttribute(
      "aria-checked",
      "false"
    );
    fireEvent.click(ruleSwitch("disallow-truncate"));
    await update();

    expect(mocks.upsertPolicy).toHaveBeenCalledTimes(1);
    const { parentPath, policy } = mocks.upsertPolicy.mock.calls[0][0];
    expect(parentPath).toBe(PROJECT);
    expect(policy.type).toBe(PolicyType.REVIEW_RULE);
    expect(policy.resourceType).toBe(PolicyResourceType.PROJECT);
    expect(policy.enforce).toBe(true);
    expect(policy.policy.value.rules).toEqual([
      ReviewRuleType.SYNTAX,
      ReviewRuleType.REQUIRE_WHERE,
      ReviewRuleType.DISALLOW_TRUNCATE,
    ]);
    expect(updateButton()).not.toBeInTheDocument();
    expect(customize).toHaveAttribute("aria-checked", "true");
    expect(mocks.pushNotification).toHaveBeenCalledTimes(1);
  });

  test("turning customization off deletes the project policy", async () => {
    seedCustomized();
    const customize = await renderLoaded();
    expect(customize).toHaveAttribute("aria-checked", "true");
    expect(ruleSwitch("require-where")).toHaveAttribute(
      "aria-checked",
      "false"
    );

    fireEvent.click(customize);
    // Back to the workspace rules, shown as text again.
    expect(screen.getAllByRole("switch")).toHaveLength(1);
    await update();

    expect(mocks.upsertPolicy).not.toHaveBeenCalled();
    expect(mocks.deletePolicy).toHaveBeenCalledWith(
      `${PROJECT}/policies/review_rule`
    );
    expect(mocks.fetchPolicy).toHaveBeenLastCalledWith({
      parentPath: PROJECT,
      policyType: PolicyType.REVIEW_RULE,
      refresh: true,
    });
    expect(customize).toHaveAttribute("aria-checked", "false");
  });

  test("switching customization on and back off is not a change", async () => {
    const customize = await renderLoaded();

    fireEvent.click(customize);
    fireEvent.click(customize);
    expect(updateButton()).not.toBeInTheDocument();
  });

  test("a failed save keeps the draft", async () => {
    mocks.upsertPolicy.mockRejectedValue(new Error("denied"));
    const customize = await renderLoaded();

    fireEvent.click(customize);
    await update();

    expect(updateButton()).toBeInTheDocument();
    expect(customize).toHaveAttribute("aria-checked", "true");
    expect(mocks.pushNotification).not.toHaveBeenCalled();
  });

  test("switching customization on needs bb.policies.create", async () => {
    mocks.permissions = { "bb.policies.create": false };
    const customize = await renderLoaded();

    expect(customize).toHaveAttribute("data-disabled");
  });

  test("editing the project's own rules needs bb.policies.update", async () => {
    mocks.permissions = { "bb.policies.update": false };
    seedCustomized();
    const customize = await renderLoaded();

    expect(ruleSwitch("require-where")).toHaveAttribute("data-disabled");
    // Switching customization off is a delete, which this role may do.
    expect(customize).not.toHaveAttribute("data-disabled");
  });

  test("switching customization off needs bb.policies.delete", async () => {
    mocks.permissions = { "bb.policies.delete": false };
    seedCustomized();
    const customize = await renderLoaded();

    expect(customize).toHaveAttribute("data-disabled");
    expect(ruleSwitch("require-where")).not.toHaveAttribute("data-disabled");
  });

  test("a workspace policy switched off through the API counts as every rule", async () => {
    seedPolicies({
      [WORKSPACE]: reviewRulePolicy(WORKSPACE, [ReviewRuleType.SYNTAX], false),
    });
    const customize = await renderLoaded();

    expect(customize).toHaveAttribute("aria-checked", "false");
    expect(screen.getAllByText("sql-review.standard-rules.on")).toHaveLength(
      11
    );
    fireEvent.click(customize);
    expect(ruleSwitch("disallow-truncate")).toHaveAttribute(
      "aria-checked",
      "true"
    );
  });

  test("a project row switched off through the API is updated, not created", async () => {
    mocks.permissions = { "bb.policies.create": false };
    seedPolicies({
      [WORKSPACE]: workspacePolicy(),
      [PROJECT]: reviewRulePolicy(PROJECT, [ReviewRuleType.SYNTAX], false),
    });
    const customize = await renderLoaded();

    // The row is not in force, so the project follows the workspace.
    expect(customize).toHaveAttribute("aria-checked", "false");
    expect(customize).not.toHaveAttribute("data-disabled");
    fireEvent.click(customize);
    expect(ruleSwitch("require-where")).not.toHaveAttribute("data-disabled");
    await update();

    expect(mocks.upsertPolicy).toHaveBeenCalledTimes(1);
    expect(mocks.upsertPolicy.mock.calls[0][0].policy.enforce).toBe(true);
  });

  test("a project row switched off through the API needs bb.policies.update", async () => {
    mocks.permissions = { "bb.policies.update": false };
    seedPolicies({
      [WORKSPACE]: workspacePolicy(),
      [PROJECT]: reviewRulePolicy(PROJECT, [ReviewRuleType.SYNTAX], false),
    });
    const customize = await renderLoaded();

    expect(customize).toHaveAttribute("data-disabled");
  });

  test("a failed read shows an error instead of the switches, and retry reads again", async () => {
    mocks.unreadable.add(PROJECT);
    render(<ProjectSQLReviewPage projectId="p" />);

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "sql-review.standard-rules.load-failed"
    );
    expect(screen.queryAllByRole("switch")).toHaveLength(0);

    mocks.unreadable.clear();
    fireEvent.click(
      screen.getByRole("button", { name: "sql-review.standard-rules.retry" })
    );

    const customize = await screen.findByRole("switch", {
      name: "sql-review.standard-rules.customize.self",
    });
    expect(customize).toHaveAttribute("aria-checked", "false");
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(mocks.fetchPolicy).toHaveBeenCalledWith({
      parentPath: PROJECT,
      policyType: PolicyType.REVIEW_RULE,
      refresh: true,
    });
  });

  test("a project policy without SYNTAX is customized with every rule off", async () => {
    seedPolicies({
      [WORKSPACE]: workspacePolicy(),
      [PROJECT]: reviewRulePolicy(PROJECT, [ReviewRuleType.REQUIRE_WHERE]),
    });
    const customize = await renderLoaded();

    expect(customize).toHaveAttribute("aria-checked", "true");
    expect(ruleSwitch("require-where")).toHaveAttribute(
      "aria-checked",
      "false"
    );
    expect(
      screen.getByText("sql-review.standard-rules.syntax-off")
    ).toBeInTheDocument();
  });

  test("moving to another project drops the draft and the remembered rules", async () => {
    seedPolicies({
      [WORKSPACE]: workspacePolicy(),
      // Customized with SYNTAX off.
      "projects/q": reviewRulePolicy("projects/q", []),
    });
    const { rerender } = render(<ProjectSQLReviewPage projectId="p" />);
    const customize = await screen.findByRole("switch", {
      name: "sql-review.standard-rules.customize.self",
    });
    fireEvent.click(customize);
    expect(updateButton()).toBeInTheDocument();

    rerender(<ProjectSQLReviewPage projectId="q" />);

    const customizeOther = await screen.findByRole("switch", {
      name: "sql-review.standard-rules.customize.self",
    });
    expect(mocks.fetchPolicy).toHaveBeenCalledWith(
      expect.objectContaining({ parentPath: "projects/q" })
    );
    expect(customizeOther).toHaveAttribute("aria-checked", "true");
    expect(updateButton()).not.toBeInTheDocument();

    // Switching SYNTAX on starts from every rule, not from what the first
    // project had on.
    fireEvent.click(ruleSwitch("syntax"));
    expect(ruleSwitch("disallow-truncate")).toHaveAttribute(
      "aria-checked",
      "true"
    );
  });

  test("a retry that fails again keeps the error, whatever the cache holds", async () => {
    mocks.unreadable.add(PROJECT);
    render(<ProjectSQLReviewPage projectId="p" />);
    await screen.findByRole("alert");

    // The workspace policy read fine and is cached; now only its read fails.
    mocks.unreadable.clear();
    mocks.unreadable.add(WORKSPACE);
    fireEvent.click(
      screen.getByRole("button", { name: "sql-review.standard-rules.retry" })
    );

    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(screen.queryAllByRole("switch")).toHaveLength(0);
    expect(mocks.fetchPolicy).toHaveBeenLastCalledWith({
      parentPath: WORKSPACE,
      policyType: PolicyType.REVIEW_RULE,
      refresh: true,
    });
  });

  test("a role held only on the project loads without the workspace policy", async () => {
    mocks.workspacePermissions = { "bb.policies.get": false };
    const customize = await renderLoaded();

    expect(mocks.fetchPolicy).not.toHaveBeenCalledWith(
      expect.objectContaining({ parentPath: WORKSPACE })
    );
    expect(screen.queryByRole("alert")).toHaveTextContent(
      "sql-review.standard-rules.workspace-rules-hidden"
    );
    expect(
      screen.queryByText("sql-review.standard-rules.on")
    ).not.toBeInTheDocument();

    // Customizing starts from every rule, as nothing else is known.
    fireEvent.click(customize);
    expect(screen.getAllByRole("switch")).toHaveLength(12);
    expect(ruleSwitch("disallow-truncate")).toHaveAttribute(
      "aria-checked",
      "true"
    );
  });

  test("undoing an unsaved switch needs no permission", async () => {
    mocks.permissions = { "bb.policies.delete": false };
    const customize = await renderLoaded();

    fireEvent.click(customize);
    expect(customize).not.toHaveAttribute("data-disabled");
    fireEvent.click(customize);
    expect(customize).toHaveAttribute("aria-checked", "false");
    expect(updateButton()).not.toBeInTheDocument();
  });
});
