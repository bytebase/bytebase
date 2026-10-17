import { create } from "@bufbuild/protobuf";
import {
  act,
  fireEvent,
  render,
  screen,
  within,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  type Policy,
  PolicyResourceType,
  PolicySchema,
  PolicyType,
  ReviewAIPolicySchema,
  ReviewRulePolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";

const WORKSPACE = "workspaces/ws";
const PROJECT = "projects/p";

const ruleName = (parent: string) => `${parent}/policies/review_rule`;
const aiName = (parent: string) => `${parent}/policies/review_ai`;

const mocks = vi.hoisted(() => ({
  permissions: {} as Record<string, boolean>,
  workspacePermissions: {} as Record<string, boolean>,
  // Policy names whose read fails: the store then caches nothing for them.
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

// Mirrors the real store: policies are cached by name, a fetch of a missing
// policy caches one with no payload and resolves to null, a failed read
// resolves to undefined and leaves the cache as it was, and a delete drops the
// cached entry.
vi.mock("@/stores/app", async () => {
  const { create: createStore } = await import("zustand");
  const { create: createMessage } = await import("@bufbuild/protobuf");
  const { PolicySchema: Schema, PolicyType: Type } = await import(
    "@/types/proto-es/v1/org_policy_service_pb"
  );
  type StorePolicy = import("@/types/proto-es/v1/org_policy_service_pb").Policy;
  type Find = { parentPath: string; policyType: number; refresh?: boolean };
  const nameOf = (parentPath: string, policyType: number) =>
    `${parentPath}/policies/${Type[policyType].toLowerCase()}`;
  const useAppStore = createStore<{
    policies: Record<string, StorePolicy>;
    serverInfo?: { aiEnabled: boolean; aiSqlReviewEnabled: boolean };
    getPolicyByParentAndType: (find: Find) => StorePolicy | undefined;
    fetchPolicyByParentAndType: (
      find: Find
    ) => Promise<StorePolicy | null | undefined>;
    getOrFetchPolicyByParentAndType: (
      find: Find
    ) => Promise<StorePolicy | undefined>;
    upsertPolicy: (params: {
      parentPath: string;
      policy: Partial<StorePolicy>;
    }) => Promise<StorePolicy>;
    deletePolicy: (name: string) => Promise<void>;
  }>()((set, get) => ({
    policies: {},
    getPolicyByParentAndType: (find) =>
      get().policies[nameOf(find.parentPath, find.policyType)],
    fetchPolicyByParentAndType: async (find) => {
      mocks.fetchPolicy(find);
      const name = nameOf(find.parentPath, find.policyType);
      if (mocks.unreadable.has(name)) return undefined;
      if (!get().policies[name]) {
        set((state) => ({
          policies: {
            ...state.policies,
            [name]: createMessage(Schema, { name }),
          },
        }));
        return null;
      }
      return get().policies[name];
    },
    getOrFetchPolicyByParentAndType: async (find) =>
      (await get().fetchPolicyByParentAndType(find)) ?? undefined,
    upsertPolicy: async (params) => {
      await mocks.upsertPolicy(params);
      const name = nameOf(params.parentPath, params.policy.type!);
      const saved = { ...params.policy, name } as StorePolicy;
      set((state) => ({ policies: { ...state.policies, [name]: saved } }));
      return saved;
    },
    deletePolicy: async (name) => {
      await mocks.deletePolicy(name);
      set((state) => {
        const { [name]: _removed, ...policies } = state.policies;
        return { policies };
      });
    },
  }));
  return { useAppStore };
});

const { useAppStore } = await import("@/stores/app");
// Caches each policy under its name; the keys only label the seed.
const seedPolicies = (policies: Record<string, Policy>) =>
  (
    useAppStore as unknown as {
      setState: (state: { policies: Record<string, Policy> }) => void;
    }
  ).setState({
    policies: Object.fromEntries(
      Object.values(policies).map((policy) => [policy.name, policy])
    ),
  });
const seedServerInfo = (serverInfo?: {
  aiEnabled: boolean;
  aiSqlReviewEnabled: boolean;
}) =>
  (
    useAppStore as unknown as {
      setState: (state: { serverInfo: typeof serverInfo }) => void;
    }
  ).setState({ serverInfo });
const { ProjectSQLReviewPage } = await import("./ProjectSQLReviewPage");

const reviewRulePolicy = (
  parent: string,
  rules: ReviewRuleType[],
  enforce = true
): Policy =>
  create(PolicySchema, {
    name: ruleName(parent),
    type: PolicyType.REVIEW_RULE,
    enforce,
    policy: {
      case: "reviewRulePolicy",
      value: create(ReviewRulePolicySchema, { rules }),
    },
  });

const reviewAIPolicy = (parent: string, content: string): Policy =>
  create(PolicySchema, {
    name: aiName(parent),
    type: PolicyType.REVIEW_AI,
    enforce: true,
    policy: {
      case: "reviewAiPolicy",
      value: create(ReviewAIPolicySchema, { content }),
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

const policyEditor = () =>
  screen.findByRole("textbox", {
    name: "sql-review.ai-review.policy.project-self",
  });

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
    seedServerInfo({ aiEnabled: true, aiSqlReviewEnabled: true });
  });

  // The row of the AI review: its title, its description, and its state.
  const aiReviewRow = () =>
    within(
      screen.getByText("sql-review.ai-review.enable.self").parentElement!
        .parentElement!
    );

  test("shows the AI review as on when the workspace has it on and AI is on", async () => {
    seedServerInfo({ aiEnabled: true, aiSqlReviewEnabled: true });
    await renderLoaded();

    expect(
      aiReviewRow().getByText("sql-review.standard-rules.on")
    ).toBeInTheDocument();
    expect(
      aiReviewRow().getByText("sql-review.ai-review.set-for-workspace", {
        exact: false,
      })
    ).toBeInTheDocument();
    expect(
      await screen.findByRole("link", {
        name: "sql-review.ai-review.view-workspace-setting",
      })
    ).toBeInTheDocument();
    expect(aiReviewRow().queryByRole("switch")).not.toBeInTheDocument();
  });

  test("uses the normal project page width for SQL Review", async () => {
    const { container } = render(<ProjectSQLReviewPage projectId="p" />);
    await screen.findByRole("switch", {
      name: "sql-review.standard-rules.customize.self",
    });

    const page = container.querySelector('[data-slot="project-page-layout"]');
    expect(page?.firstElementChild).not.toHaveClass("mx-auto", "max-w-3xl");
  });

  test("does not add outer padding around the Standard rules section", async () => {
    render(<ProjectSQLReviewPage projectId="p" />);
    await screen.findByRole("switch", {
      name: "sql-review.standard-rules.customize.self",
    });

    const section = screen
      .getByRole("heading", { name: "sql-review.standard-rules.self" })
      .closest('[data-slot="form-section"]');
    expect(getComputedStyle(section!).paddingTop).toBe("0");
    expect(getComputedStyle(section!).paddingBottom).toBe("0");
  });

  test.each([
    ["the workspace has it off", { aiEnabled: true, aiSqlReviewEnabled: false }],
    ["AI is off for the workspace", { aiEnabled: false, aiSqlReviewEnabled: true }],
  ])("shows the AI review as off when %s", async (_name, serverInfo) => {
    seedServerInfo(serverInfo);
    await renderLoaded();

    expect(
      aiReviewRow().getByText("sql-review.standard-rules.off")
    ).toBeInTheDocument();
    expect(
      screen.queryByText("sql-review.ai-review.policy.self")
    ).not.toBeInTheDocument();
  });

  test("the workspace setting link of the AI review needs the workspace policy read", async () => {
    mocks.workspacePermissions["bb.policies.get"] = false;
    await renderLoaded();

    expect(aiReviewRow().queryByRole("link")).not.toBeInTheDocument();
  });

  test("uses the shared project policy editor without duplicating the workspace policy", async () => {
    seedPolicies({
      [WORKSPACE]: workspacePolicy(),
      ai: reviewAIPolicy(WORKSPACE, "Every foreign key needs an index."),
    });
    render(<ProjectSQLReviewPage projectId="p" />);

    expect(
      screen.queryByText("Every foreign key needs an index.")
    ).not.toBeInTheDocument();
    expect(
      screen.getByText("sql-review.ai-review.policy.project-description")
    ).toBeInTheDocument();
    const projectPolicyEditor = await policyEditor();
    expect(projectPolicyEditor).not.toHaveAttribute("readonly");
    expect(
      projectPolicyEditor.closest('[data-slot="form-field"]')
    ).toHaveStyle({ rowGap: "16px" });
  });

  test("a project without its own policy shows the workspace rules as text", async () => {
    const customize = await renderLoaded();

    expect(customize).toHaveAttribute("aria-checked", "false");
    // Only the customize control is a switch; the rules are read-only.
    expect(screen.getAllByRole("switch")).toHaveLength(1);
    expect(screen.getAllByText("sql-review.standard-rules.on")).toHaveLength(3);
    // Nine rules, and the AI review row below them.
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
    expect(mocks.deletePolicy).toHaveBeenCalledWith(ruleName(PROJECT));
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
      12
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
    mocks.unreadable.add(ruleName(PROJECT));
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
    mocks.unreadable.add(ruleName(PROJECT));
    render(<ProjectSQLReviewPage projectId="p" />);
    await screen.findByRole("alert");

    // The workspace policy read fine and is cached; now only its read fails.
    mocks.unreadable.clear();
    mocks.unreadable.add(ruleName(WORKSPACE));
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
    expect(screen.getAllByText("sql-review.standard-rules.on")).toHaveLength(1);
    expect(
      screen.queryByText(
        "sql-review.standard-rules.customize.view-workspace-rules"
      )
    ).not.toBeInTheDocument();

    // Customizing starts from every rule, as nothing else is known.
    fireEvent.click(customize);
    expect(screen.getAllByRole("switch")).toHaveLength(12);
    expect(ruleSwitch("disallow-truncate")).toHaveAttribute(
      "aria-checked",
      "true"
    );
  });

  test("the workspace rules link needs only the workspace policy read", async () => {
    mocks.workspacePermissions = { "bb.reviewConfigs.list": false };
    await renderLoaded();

    expect(
      screen.getByText(
        "sql-review.standard-rules.customize.view-workspace-rules"
      )
    ).toBeInTheDocument();
  });

  test("following the workspace in between keeps the project's own rules to restore", async () => {
    seedCustomized();
    const customize = await renderLoaded();

    fireEvent.click(ruleSwitch("syntax"));
    fireEvent.click(customize);
    expect(screen.getAllByText("sql-review.standard-rules.on")).toHaveLength(3);
    fireEvent.click(customize);
    fireEvent.click(ruleSwitch("syntax"));

    expect(ruleSwitch("syntax")).toHaveAttribute("aria-checked", "true");
    expect(ruleSwitch("require-where")).toHaveAttribute(
      "aria-checked",
      "false"
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

  test("the project AI review policy saves at the project from the footer", async () => {
    render(<ProjectSQLReviewPage projectId="p" />);

    const editor = await policyEditor();
    expect(editor).toHaveValue("");
    expect(editor).toHaveAttribute(
      "placeholder",
      "sql-review.ai-review.policy.project-placeholder"
    );
    fireEvent.change(editor, { target: { value: "No DROP in this project." } });
    await update();

    expect(mocks.upsertPolicy).toHaveBeenCalledOnce();
    const { parentPath, policy } = mocks.upsertPolicy.mock.calls[0][0];
    expect(parentPath).toBe(PROJECT);
    expect(policy.type).toBe(PolicyType.REVIEW_AI);
    expect(policy.resourceType).toBe(PolicyResourceType.PROJECT);
    expect(policy.enforce).toBe(true);
    expect(policy.policy.value.content).toBe("No DROP in this project.");
    expect(updateButton()).not.toBeInTheDocument();
    expect(mocks.pushNotification).toHaveBeenCalledOnce();
  });

  test("the footer saves the rules and the AI review policy, and reports one update", async () => {
    const customize = await renderLoaded();

    fireEvent.click(customize);
    fireEvent.change(await policyEditor(), { target: { value: "No DROP." } });
    await update();

    expect(mocks.upsertPolicy).toHaveBeenCalledTimes(2);
    expect(
      mocks.upsertPolicy.mock.calls.map(([params]) => params.policy.type)
    ).toEqual(
      expect.arrayContaining([PolicyType.REVIEW_RULE, PolicyType.REVIEW_AI])
    );
    expect(updateButton()).not.toBeInTheDocument();
    expect(mocks.pushNotification).toHaveBeenCalledOnce();
  });

  test("a failed AI review policy save keeps its draft and reports no update", async () => {
    mocks.upsertPolicy.mockImplementation(async ({ policy }) => {
      if (policy.type === PolicyType.REVIEW_AI) throw new Error("denied");
    });
    const customize = await renderLoaded();

    fireEvent.click(customize);
    const editor = await policyEditor();
    fireEvent.change(editor, { target: { value: "No DROP." } });
    await update();

    expect(customize).toHaveAttribute("aria-checked", "true");
    expect(editor).toHaveValue("No DROP.");
    expect(updateButton()).toBeInTheDocument();
    expect(mocks.pushNotification).not.toHaveBeenCalled();
  });

  test("cancel reverts the AI review policy too", async () => {
    seedPolicies({
      [WORKSPACE]: workspacePolicy(),
      ai: reviewAIPolicy(PROJECT, "Stored."),
    });
    render(<ProjectSQLReviewPage projectId="p" />);

    const editor = await policyEditor();
    fireEvent.change(editor, { target: { value: "Edited." } });
    fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));

    expect(editor).toHaveValue("Stored.");
    expect(updateButton()).not.toBeInTheDocument();
  });

  test("the footer cannot update while the AI review policy is over the size limit", async () => {
    render(<ProjectSQLReviewPage projectId="p" />);

    fireEvent.change(await policyEditor(), {
      target: { value: "a".repeat(64 * 1024 + 1) },
    });

    expect(
      screen.getByText("sql-review.ai-review.policy.too-large")
    ).toBeInTheDocument();
    expect(updateButton()).toBeDisabled();
  });

  test("the project AI review policy shows as text without bb.policies.update or bb.policies.delete", async () => {
    mocks.permissions = {
      "bb.policies.update": false,
      "bb.policies.delete": false,
    };
    seedPolicies({
      [WORKSPACE]: workspacePolicy(),
      ai: reviewAIPolicy(PROJECT, "No DROP in this project."),
    });
    render(<ProjectSQLReviewPage projectId="p" />);

    expect(
      await screen.findByText("No DROP in this project.")
    ).toBeInTheDocument();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  });

  test("a project role with bb.policies.delete but not bb.policies.update removes the AI review policy from the footer", async () => {
    mocks.permissions = { "bb.policies.update": false };
    seedPolicies({
      [WORKSPACE]: workspacePolicy(),
      ai: reviewAIPolicy(PROJECT, "No DROP in this project."),
    });
    render(<ProjectSQLReviewPage projectId="p" />);

    fireEvent.change(await policyEditor(), { target: { value: "" } });
    await update();

    expect(mocks.deletePolicy).toHaveBeenCalledWith(aiName(PROJECT));
    expect(mocks.upsertPolicy).not.toHaveBeenCalled();
    expect(updateButton()).not.toBeInTheDocument();
    expect(mocks.pushNotification).toHaveBeenCalledOnce();
  });

  test("a project role with bb.policies.delete but not bb.policies.update cannot update changed text", async () => {
    mocks.permissions = { "bb.policies.update": false };
    seedPolicies({
      [WORKSPACE]: workspacePolicy(),
      ai: reviewAIPolicy(PROJECT, "No DROP in this project."),
    });
    render(<ProjectSQLReviewPage projectId="p" />);

    fireEvent.change(await policyEditor(), {
      target: { value: "No DROP or TRUNCATE in this project." },
    });

    expect(
      screen.getByText("sql-review.ai-review.policy.remove-only")
    ).toBeInTheDocument();
    expect(updateButton()).toBeDisabled();
  });

  test("a project without an AI review policy needs bb.policies.create to add one", async () => {
    mocks.permissions = { "bb.policies.create": false };
    render(<ProjectSQLReviewPage projectId="p" />);

    expect(
      await screen.findByText("sql-review.ai-review.policy.project-none")
    ).toBeInTheDocument();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  });
});
